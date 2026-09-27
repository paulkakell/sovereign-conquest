'use strict';

const assert = require('node:assert/strict');
const test = require('node:test');
const finalize = require('./finalize-release.cjs');
const { SOURCE_BRANCHES, REQUIRED_WORKFLOWS, PREVIOUS_SHA } = finalize;
const MAIN_SHA = 'a'.repeat(40);
const RELEASE_SHA = 'b'.repeat(40);
const [FIRST_BRANCH, FIRST_SHA] = Object.entries(SOURCE_BRANCHES)[0];
const missing = () => { const error = new Error('Not found'); error.status = 404; throw error; };

function fixture() {
  const state = {
    main: MAIN_SHA,
    version: '01.06.04',
    previousVersion: '01.06.03',
    branches: new Map(Object.entries(SOURCE_BRANCHES).map(([name, sha]) => [name, { sha, protected: false }])),
    tags: new Map(),
    releases: new Map(),
    mutations: [],
    messages: [],
    openPulls: new Map(),
    closedPulls: [],
    branchReads: new Map(),
    comparisons: new Map(),
    runs: REQUIRED_WORKFLOWS.map(([name, path], index) => ({
      id: index + 1, name, path, head_sha: MAIN_SHA, head_branch: 'main',
      event: 'push', status: 'completed', conclusion: 'success', run_attempt: 1,
    })),
  };
  const github = { rest: { repos: {}, actions: {}, git: {}, pulls: {} } };
  github.rest.repos.getBranch = async ({ branch }) => {
    const reads = (state.branchReads.get(branch) || 0) + 1;
    state.branchReads.set(branch, reads);
    if (state.onBranchRead) state.onBranchRead(branch, reads);
    if (branch === 'main') return { data: { commit: { sha: state.main }, protected: true } };
    const entry = state.branches.get(branch);
    if (!entry) missing();
    return { data: { commit: { sha: entry.sha }, protected: entry.protected } };
  };
  github.rest.repos.getContent = async ({ path, ref }) => ({ data: {
    type: 'file', encoding: 'base64',
    content: Buffer.from(path === 'VERSION' ?
      (ref === PREVIOUS_SHA ? state.previousVersion : state.version) : '# Release 01.06.04\nValidated consolidation.\n').toString('base64'),
  } });
  github.rest.actions.listWorkflowRunsForRepo = async args => {
    assert.equal(args.head_sha, MAIN_SHA);
    assert.equal(args.branch, 'main');
    return { data: { workflow_runs: state.runs } };
  };
  github.rest.pulls.list = async ({ state: filter, head, base }) => {
    const branch = head.slice(head.indexOf(':') + 1);
    if (filter === 'closed') {
      assert.equal(base, 'main');
      return { data: state.closedPulls };
    }
    return { data: state.openPulls.get(branch) || [] };
  };
  github.paginate = async (method, args) => {
    const response = await method(args);
    return response.data.workflow_runs || response.data;
  };
  github.rest.repos.compareCommitsWithBasehead = async ({ basehead }) => {
    const [base, head] = basehead.split('...');
    assert.equal(head, MAIN_SHA);
    return { data: state.comparisons.get(base) || { status: 'ahead', behind_by: 0 } };
  };
  github.rest.git.getRef = async ({ ref }) => {
    const tag = state.tags.get(ref);
    if (!tag) missing();
    return { data: { object: typeof tag === 'string' ? { sha: tag, type: 'commit' } : tag } };
  };
  github.rest.git.getTag = async ({ tag_sha }) => ({ data: { object: { type: 'commit', sha: state.annotatedTags[tag_sha] } } });
  github.rest.git.createRef = async ({ ref, sha }) => {
    assert.equal(state.tags.has(ref.slice(5)), false, 'Existing tags must never be replaced');
    state.mutations.push({ action: 'tag', ref, sha });
    state.tags.set(ref.slice(5), sha);
    return { data: {} };
  };
  github.rest.repos.getReleaseByTag = async ({ tag }) => {
    if (!state.releases.has(tag)) missing();
    return { data: state.releases.get(tag) };
  };
  github.rest.repos.createRelease = async args => {
    state.mutations.push({ action: 'release', ...args });
    state.releases.set(args.tag_name, args);
    return { data: args };
  };
  github.rest.git.deleteRef = async ({ ref }) => {
    assert.ok(ref.startsWith('heads/'));
    const branch = ref.slice(6);
    assert.notEqual(branch, 'main');
    assert.ok(state.branches.has(branch));
    state.mutations.push({ action: 'delete', branch });
    state.branches.delete(branch);
    return { data: {} };
  };
  const args = {
    github,
    context: {
      repo: { owner: 'paulkakell', repo: 'sovereign-conquest' },
      payload: { workflow_run: {
        head_sha: MAIN_SHA, head_branch: 'main', conclusion: 'success',
        head_repository: { full_name: 'paulkakell/sovereign-conquest' },
      } },
    },
    core: { info: message => state.messages.push(message) },
  };
  return { state, args, run: () => finalize(args) };
}

function mergedReleaseBranch(state) {
  state.branches.set('release-01.06.04', { sha: RELEASE_SHA, protected: false });
  state.closedPulls.push({
    number: 99, merged_at: '2026-09-27T20:00:00Z', base: { ref: 'main' },
    head: { ref: 'release-01.06.04', sha: RELEASE_SHA, repo: { full_name: 'paulkakell/sovereign-conquest' } },
  });
}

test('validated main preserves rollback, creates release, and deletes only exact incorporated sources', async () => {
  const { state, run } = fixture();
  mergedReleaseBranch(state);
  state.branches.set('unrelated-work', { sha: 'c'.repeat(40), protected: false });
  const result = await run();
  assert.equal(result.version, '01.06.04');
  assert.equal(result.sha, MAIN_SHA);
  assert.equal(result.deleted.length, 10);
  assert.equal(state.tags.get('tags/v01.06.03'), PREVIOUS_SHA);
  assert.equal(state.tags.get('tags/v01.06.04'), MAIN_SHA);
  assert.equal(state.releases.get('v01.06.04').body, '# Release 01.06.04\nValidated consolidation.\n');
  assert.equal(state.branches.has('unrelated-work'), true);
  assert.deepEqual(state.mutations.slice(0, 3).map(item => item.action), ['tag', 'tag', 'release']);
  assert.ok(state.branchReads.get(FIRST_BRANCH) >= 3, 'Cleanup must re-read each candidate');
});

test('a repeated successful run is idempotent and removed branches are accepted', async () => {
  const { state, run } = fixture();
  await run();
  const count = state.mutations.length;
  const result = await run();
  assert.deepEqual(result.deleted, []);
  assert.equal(state.mutations.length, count);
});

for (const conclusion of ['failure', 'cancelled', null]) {
  test(`unsuccessful or unfinished gate (${conclusion}) performs no mutations`, async () => {
    const { state, run } = fixture();
    state.runs[1].conclusion = conclusion;
    state.runs[1].status = conclusion ? 'completed' : 'in_progress';
    assert.ok((await run()).skipped);
    assert.deepEqual(state.mutations, []);
  });
}

test('all three required workflow results must belong to this exact main SHA', async () => {
  const { state, run } = fixture();
  state.runs[2].head_sha = 'd'.repeat(40);
  assert.ok((await run()).skipped);
  assert.deepEqual(state.mutations, []);
});

test('a later failed workflow run overrides an earlier success', async () => {
  const { state, run } = fixture();
  state.runs.push({ ...state.runs[0], id: 500, conclusion: 'failure' });
  assert.ok((await run()).skipped);
  assert.deepEqual(state.mutations, []);
});

test('pull-request workflow results cannot satisfy main release gates', async () => {
  const { state, run } = fixture();
  state.runs[0].event = 'pull_request';
  assert.ok((await run()).skipped);
  assert.deepEqual(state.mutations, []);
});

test('moved main at entry performs no mutations', async () => {
  const { state, run } = fixture();
  state.main = 'd'.repeat(40);
  assert.ok((await run()).skipped);
  assert.deepEqual(state.mutations, []);
});

test('main moving during preflight prevents every write', async () => {
  const { state, run } = fixture();
  state.onBranchRead = (branch, reads) => {
    if (branch === 'main' && reads === 2) state.main = 'd'.repeat(40);
  };
  await assert.rejects(run, /Main moved/);
  assert.deepEqual(state.mutations, []);
});

test('changed source branch aborts the entire cleanup before writes', async () => {
  const { state, run } = fixture();
  state.branches.get(FIRST_BRANCH).sha = 'd'.repeat(40);
  await assert.rejects(run, /Branch changed/);
  assert.deepEqual(state.mutations, []);
});

test('branch tip changing after preflight is retained without any deletion', async () => {
  const { state, run } = fixture();
  state.onBranchRead = (branch, reads) => {
    if (branch === FIRST_BRANCH && reads === 3) state.branches.get(branch).sha = 'd'.repeat(40);
  };
  await assert.rejects(run, /Branch changed during cleanup/);
  assert.equal(state.mutations.some(item => item.action === 'delete'), false);
});

test('unmerged release branch cannot be deleted', async () => {
  const { state, run } = fixture();
  mergedReleaseBranch(state);
  state.closedPulls[0].merged_at = null;
  await assert.rejects(run, /exactly one merged PR/);
  assert.deepEqual(state.mutations, []);
});

test('open PRs prevent source deletion', async () => {
  const { state, run } = fixture();
  state.openPulls.set(FIRST_BRANCH, [{ number: 25 }]);
  await assert.rejects(run, /Open PR still uses branch/);
  assert.deepEqual(state.mutations, []);
});

test('nonancestor branches prevent deletion', async () => {
  const { state, run } = fixture();
  state.comparisons.set(FIRST_SHA, { status: 'diverged', behind_by: 1 });
  await assert.rejects(run, /not fully incorporated/);
  assert.deepEqual(state.mutations, []);
});

test('protected branches are retained without attempting to alter protection', async () => {
  const { state, run } = fixture();
  state.branches.get(FIRST_BRANCH).protected = true;
  await assert.rejects(run, /Protected branch/);
  assert.deepEqual(state.mutations, []);
});

test('existing conflicting tags are never moved or followed by deletion', async () => {
  const { state, run } = fixture();
  state.tags.set('tags/v01.06.04', 'd'.repeat(40));
  await assert.rejects(run, /refusing to move/);
  assert.deepEqual(state.mutations, []);
});

test('matching annotated rollback tags are retained', async () => {
  const { state, run } = fixture();
  state.tags.set('tags/v01.06.03', { type: 'tag', sha: 'e'.repeat(40) });
  state.annotatedTags = { ['e'.repeat(40)]: PREVIOUS_SHA };
  await run();
  assert.equal(state.mutations.filter(item => item.action === 'tag').length, 1);
});

for (const flag of ['draft', 'prerelease']) {
  test(`an existing ${flag} release prevents all mutations and cleanup`, async () => {
    const { state, run } = fixture();
    state.releases.set('v01.06.04', { [flag]: true });
    await assert.rejects(run, /draft or prerelease/);
    assert.deepEqual(state.mutations, []);
  });
}

test('main moving after release creation still prevents branch deletion', async () => {
  const { state, run } = fixture();
  state.onBranchRead = (branch, reads) => {
    if (branch === 'main' && reads === 5) state.main = 'd'.repeat(40);
  };
  await assert.rejects(run, /Main moved/);
  assert.equal(state.releases.has('v01.06.04'), true);
  assert.equal(state.mutations.some(item => item.action === 'delete'), false);
});

test('prior version is verified before preserving the rollback tag', async () => {
  const { state, run } = fixture();
  state.previousVersion = '01.06.02';
  await assert.rejects(run, /does not declare version/);
  assert.deepEqual(state.mutations, []);
});

test('this bounded workflow performs no work for another release version', async () => {
  const { state, run } = fixture();
  state.version = '01.06.05';
  assert.ok((await run()).skipped);
  assert.deepEqual(state.mutations, []);
});

test('fork-origin workflow triggers cannot authorize finalization', async () => {
  const { state, args, run } = fixture();
  args.context.payload.workflow_run.head_repository.full_name = 'someone/sovereign-conquest';
  assert.ok((await run()).skipped);
  assert.deepEqual(state.mutations, []);
});
