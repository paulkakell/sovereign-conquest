'use strict';

// This release creates its own tag and notes only. It never deletes branches or
// modifies repository permissions, funding accounts, or existing releases.
const VERSION = '01.06.08';
const PREVIOUS_SHA = 'c3de81af4d471e7cf327e6f43df96ff321a6b04a';
const { REQUIRED_WORKFLOWS: BASE_WORKFLOWS } = require('./finalize-release.cjs');
const REQUIRED_WORKFLOWS = [...BASE_WORKFLOWS, ['Database Startup', '.github/workflows/db-startup.yml']];

async function absentOn404(call) {
  try { return await call(); } catch (error) {
    if (error.status === 404) return null;
    throw error;
  }
}

async function finalize({ github, context, core }) {
  const { owner, repo } = context.repo;
  const repository = { owner, repo };
  const trigger = context.payload.workflow_run;
  const skip = reason => { core.info(`Database release skipped: ${reason}`); return { skipped: reason }; };
  if (owner !== 'paulkakell' || repo !== 'sovereign-conquest') return skip('unexpected repository');
  if (!trigger || trigger.head_branch !== 'main' || trigger.conclusion !== 'success' ||
      trigger.head_repository?.full_name !== `${owner}/${repo}`) return skip('untrusted or unsuccessful trigger');
  const sha = trigger.head_sha;
  if (!/^[0-9a-f]{40}$/.test(sha)) throw new Error('Invalid triggering commit');
  const getMain = async () => (await github.rest.repos.getBranch({ ...repository, branch: 'main' })).data.commit.sha;
  const assertMain = async () => {
    if (await getMain() !== sha) throw new Error('Main moved; refusing release publication');
  };
  if (await getMain() !== sha) return skip('main has moved');

  const readFile = async path => {
    const { data } = await github.rest.repos.getContent({ ...repository, path, ref: sha });
    if (Array.isArray(data) || data.type !== 'file' || data.encoding !== 'base64') {
      throw new Error(`Expected a base64 file at ${path}`);
    }
    return Buffer.from(data.content, 'base64').toString('utf8').trim();
  };
  if (await readFile('VERSION') !== VERSION) return skip('different release version');

  const runs = await github.paginate(github.rest.actions.listWorkflowRunsForRepo, {
    ...repository, branch: 'main', head_sha: sha, per_page: 100,
  });
  for (const [name, path] of REQUIRED_WORKFLOWS) {
    const matching = runs.filter(run => run.name === name && run.path === path &&
      run.head_sha === sha && run.head_branch === 'main' &&
      ['push', 'workflow_dispatch', 'schedule'].includes(run.event));
    matching.sort((a, b) => b.id - a.id || (b.run_attempt || 1) - (a.run_attempt || 1));
    if (!matching[0] || matching[0].status !== 'completed' || matching[0].conclusion !== 'success') {
      return skip(`${name} has not succeeded on current main`);
    }
  }

  const tagTarget = async tag => {
    const ref = await absentOn404(() => github.rest.git.getRef({ ...repository, ref: `tags/${tag}` }));
    if (!ref) return null;
    let object = ref.data.object;
    for (let depth = 0; object.type === 'tag' && depth < 5; depth++) {
      object = (await github.rest.git.getTag({ ...repository, tag_sha: object.sha })).data.object;
    }
    if (object.type !== 'commit') throw new Error(`Tag ${tag} does not resolve to a commit`);
    return object.sha;
  };
  if (await tagTarget('v01.06.06') !== PREVIOUS_SHA) throw new Error('Previous release tag is missing or changed');
  const notes = await readFile(`docs/RELEASE_NOTES_${VERSION}.md`);
  if (!notes) throw new Error('Release notes are empty');
  const tag = `v${VERSION}`;
  const existingTag = await tagTarget(tag);
  if (existingTag && existingTag !== sha) throw new Error('Release tag points elsewhere; refusing to move it');
  const release = await absentOn404(() => github.rest.repos.getReleaseByTag({ ...repository, tag }));
  if (release && (!existingTag || release.data.draft || release.data.prerelease)) {
    throw new Error('Existing release is inconsistent, a draft, or a prerelease');
  }
  if (!existingTag) {
    await assertMain();
    await github.rest.git.createRef({ ...repository, ref: `refs/tags/${tag}`, sha });
  }
  if (!release) {
    await assertMain();
    await github.rest.repos.createRelease({
      ...repository, tag_name: tag, target_commitish: sha,
      name: `Sovereign Conquest ${VERSION}`, body: notes, draft: false, prerelease: false,
    });
  }
  return { version: VERSION, sha };
}

module.exports = finalize;
module.exports.VERSION = VERSION;
module.exports.PREVIOUS_SHA = PREVIOUS_SHA;

module.exports.REQUIRED_WORKFLOWS = REQUIRED_WORKFLOWS;
