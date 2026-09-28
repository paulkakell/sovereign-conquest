'use strict';

const assert = require('node:assert/strict');
const test = require('node:test');
const finalize = require('./finalize-code-scanning-release.cjs');
const { REQUIRED_WORKFLOWS } = finalize;
const MAIN = 'a'.repeat(40);
const missing = () => { const error = new Error('Not found'); error.status = 404; throw error; };

function fixture() {
  const state = {
    main: MAIN, reads: 0, version: finalize.VERSION, notes: '# Database release',
    tags: new Map([['tags/v01.06.10', finalize.PREVIOUS_SHA]]), release: null, writes: [],
    runs: REQUIRED_WORKFLOWS.map(([name, path], index) => ({
      id: index + 1, name, path, head_sha: MAIN, head_branch: 'main',
      event: 'push', status: 'completed', conclusion: 'success',
    })),
  };
  const github = { rest: { repos: {}, git: {}, actions: {} } };
  github.rest.repos.getBranch = async () => {
    state.reads++;
    if (state.moveAt === state.reads) state.main = 'b'.repeat(40);
    return { data: { commit: { sha: state.main } } };
  };
  github.rest.repos.getContent = async ({ path, ref }) => {
    assert.equal(ref, MAIN);
    return { data: { type: 'file', encoding: 'base64',
      content: Buffer.from(path === 'VERSION' ? state.version : state.notes).toString('base64') } };
  };
  github.rest.actions.listWorkflowRunsForRepo = async () => {};
  github.paginate = async (_method, args) => {
    assert.equal(args.head_sha, MAIN);
    return state.runs;
  };
  github.rest.git.getRef = async ({ ref }) => {
    if (state.readError) throw state.readError;
    if (!state.tags.has(ref)) missing();
    const target = state.tags.get(ref);
    return { data: { object: typeof target === 'string' ? { type: 'commit', sha: target } : target } };
  };
  github.rest.git.getTag = async () => ({ data: { object: { type: 'commit', sha: finalize.PREVIOUS_SHA } } });
  github.rest.git.createRef = async ({ ref, sha }) => {
    assert.equal(state.tags.has(ref.slice(5)), false);
    state.tags.set(ref.slice(5), sha);
    state.writes.push('tag');
  };
  github.rest.repos.getReleaseByTag = async () => {
    if (!state.release) missing();
    return { data: state.release };
  };
  github.rest.repos.createRelease = async args => {
    assert.equal(args.target_commitish, MAIN);
    assert.equal(args.body, state.notes);
    state.release = args;
    state.writes.push('release');
  };
  const context = { repo: { owner: 'paulkakell', repo: 'sovereign-conquest' }, payload: { workflow_run: {
    head_sha: MAIN, head_branch: 'main', conclusion: 'success',
    head_repository: { full_name: 'paulkakell/sovereign-conquest' },
  } } };
  return { state, context, run: () => finalize({ github, context, core: { info() {} } }) };
}

test('publishes only the validated version and preserves previous source', async () => {
  const { state, run } = fixture();
  assert.deepEqual(await run(), { version: '01.06.11', sha: MAIN });
  assert.deepEqual(state.writes, ['tag', 'release']);
  assert.equal(state.tags.get('tags/v01.06.10'), finalize.PREVIOUS_SHA);
  assert.equal(state.tags.get('tags/v01.06.11'), MAIN);
  await run();
  assert.deepEqual(state.writes, ['tag', 'release']);
});

for (const alteration of ['failed', 'unfinished', 'missing', 'wrong-sha', 'wrong-path', 'pr-event', 'newer-failure']) {
  test(`does not publish with a ${alteration} workflow gate`, async () => {
    const { state, run } = fixture();
    if (alteration === 'failed') state.runs[0].conclusion = 'failure';
    if (alteration === 'unfinished') state.runs[0].status = 'in_progress';
    if (alteration === 'missing') state.runs.pop();
    if (alteration === 'wrong-sha') state.runs[0].head_sha = 'b'.repeat(40);
    if (alteration === 'wrong-path') state.runs[0].path = 'untrusted.yml';
    if (alteration === 'pr-event') state.runs[0].event = 'pull_request';
    if (alteration === 'newer-failure') state.runs.push({ ...state.runs[0], id: 100, conclusion: 'failure' });
    assert.ok((await run()).skipped);
    assert.deepEqual(state.writes, []);
  });
}

test('fork workflow triggers cannot publish', async () => {
  const { state, context, run } = fixture();
  context.payload.workflow_run.head_repository.full_name = 'someone/fork';
  assert.ok((await run()).skipped);
  assert.deepEqual(state.writes, []);
});

test('a future version does not reuse this release', async () => {
  const { state, run } = fixture();
  state.version = '01.06.12';
  assert.ok((await run()).skipped);
  assert.deepEqual(state.writes, []);
});

test('moved main at entry is skipped', async () => {
  const { state, run } = fixture();
  state.moveAt = 1;
  assert.ok((await run()).skipped);
  assert.deepEqual(state.writes, []);
});

for (const moveAt of [2, 3]) {
  test(`rechecks main before write ${moveAt - 1}`, async () => {
    const { state, run } = fixture();
    state.moveAt = moveAt;
    await assert.rejects(run, /Main moved/);
    assert.deepEqual(state.writes, moveAt === 2 ? [] : ['tag']);
  });
}

test('conflicting release tags are never moved', async () => {
  const { state, run } = fixture();
  state.tags.set('tags/v01.06.11', 'c'.repeat(40));
  await assert.rejects(run, /refusing to move/);
  assert.deepEqual(state.writes, []);
});

test('requires the preserved previous release', async () => {
  const { state, run } = fixture();
  state.tags.delete('tags/v01.06.10');
  await assert.rejects(run, /Previous release tag/);
  assert.deepEqual(state.writes, []);
});

test('accepts an annotated previous release tag', async () => {
  const { state, run } = fixture();
  state.tags.set('tags/v01.06.10', { type: 'tag', sha: 'c'.repeat(40) });
  await run();
  assert.deepEqual(state.writes, ['tag', 'release']);
});

for (const flag of ['draft', 'prerelease']) {
  test(`does not replace an existing ${flag} release`, async () => {
    const { state, run } = fixture();
    state.tags.set('tags/v01.06.11', MAIN);
    state.release = { [flag]: true };
    await assert.rejects(run, /Existing release/);
    assert.deepEqual(state.writes, []);
  });
}

test('API errors are not interpreted as missing objects', async () => {
  const { state, run } = fixture();
  state.readError = Object.assign(new Error('Forbidden'), { status: 403 });
  await assert.rejects(run, /Forbidden/);
  assert.deepEqual(state.writes, []);
});

test('empty notes cannot be released', async () => {
  const { state, run } = fixture();
  state.notes = ' ';
  await assert.rejects(run, /Release notes are empty/);
  assert.deepEqual(state.writes, []);
});
