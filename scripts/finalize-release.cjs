'use strict';

// A bounded audit record for the branch consolidation authorized for 01.06.04.
// Future releases must use their own reviewed release process and allowlist.
const VERSION = '01.06.04';
const PREVIOUS_SHA = '03625c68a141f649aa87480b13d61527febceff1';
const RELEASE_BRANCH = 'release-01.06.04';
const SOURCE_BRANCHES = Object.freeze({
  'dependabot/docker/golang-1.27.1-alpine3.24': '9d7ece09d1baeaf18ed98ff786780fbcf4ccbe95',
  'dependabot/docker/server/golang-1.27.1-alpine3.24': '77fc2ee76b175db42e65d280e235472e75d97f55',
  'dependabot/github_actions/actions/attest-build-provenance-4': 'a3eac5b108720e441e4b0c13bd58b2f995abf418',
  'dependabot/github_actions/aquasecurity/trivy-action-0.36.0': '7088569329e7fda3e22ba394c4e380ed7799623b',
  'dependabot/github_actions/docker/setup-buildx-action-4': '202f886af9b9e52eb5eff61dbd3214a773bfb0e3',
  'dependabot/go_modules/server/github.com/go-chi/chi/v5-5.3.2': '19ba0748a0973980d51e186b2132566c2ee1bf43',
  'dependabot/go_modules/server/github.com/jackc/pgx/v5-5.11.0': 'f0545239667ed3e06f9870336a62106a5f6defb4',
  'dependabot/go_modules/server/golang.org/x/crypto-0.57.0': 'ce83b04070601b44456d42d5bfdcb91905080b4c',
  'docs/roadmap-00.02.01': 'd21268f7fc1527695081c492509f89eb2dcf6f95',
});
const REQUIRED_WORKFLOWS = Object.freeze([
  ['CI', '.github/workflows/ci.yml'],
  ['Build Validation', '.github/workflows/build-validation.yml'],
  ['Publish GHCR Image', '.github/workflows/publish-ghcr.yml'],
]);

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
  const skip = reason => { core.info(`Release finalization skipped: ${reason}`); return { skipped: reason }; };
  if (owner !== 'paulkakell' || repo !== 'sovereign-conquest') return skip('unexpected repository');
  if (!trigger || trigger.head_branch !== 'main' || trigger.conclusion !== 'success' ||
      trigger.head_repository?.full_name !== `${owner}/${repo}`) return skip('untrusted or unsuccessful trigger');
  const sha = trigger.head_sha;
  if (!/^[0-9a-f]{40}$/.test(sha)) throw new Error('Invalid triggering commit');

  const getMain = async () => (await github.rest.repos.getBranch({ ...repository, branch: 'main' })).data.commit.sha;
  const assertMain = async () => {
    if (await getMain() !== sha) throw new Error('Main moved; refusing to finalize or delete branches');
  };
  if (await getMain() !== sha) return skip('main has moved');

  const readFileAt = async (path, ref) => {
    const { data } = await github.rest.repos.getContent({ ...repository, path, ref });
    if (Array.isArray(data) || data.type !== 'file' || data.encoding !== 'base64') {
      throw new Error(`Expected a base64 file at ${path}`);
    }
    return Buffer.from(data.content, 'base64').toString('utf8');
  };
  if ((await readFileAt('VERSION', sha)).trim() !== VERSION) return skip('release version no longer matches');

  const runs = await github.paginate(github.rest.actions.listWorkflowRunsForRepo, {
    ...repository, branch: 'main', head_sha: sha, per_page: 100,
  });
  for (const [name, path] of REQUIRED_WORKFLOWS) {
    const matching = runs.filter(run => run.name === name && run.path === path &&
      run.head_sha === sha && run.head_branch === 'main' &&
      ['push', 'workflow_dispatch', 'schedule'].includes(run.event));
    matching.sort((a, b) => b.id - a.id || (b.run_attempt || 1) - (a.run_attempt || 1));
    const latest = matching[0];
    if (!latest || latest.status !== 'completed' || latest.conclusion !== 'success') {
      return skip(`${name} has not succeeded on current main`);
    }
  }

  if ((await readFileAt('VERSION', PREVIOUS_SHA)).trim() !== '01.06.03') {
    throw new Error('Previous release source does not declare version 01.06.03');
  }
  const notes = await readFileAt(`docs/RELEASE_NOTES_${VERSION}.md`, sha);
  if (!notes.trim()) throw new Error('Release notes are empty');

  const getBranch = branch => absentOn404(() => github.rest.repos.getBranch({ ...repository, branch }));
  const listPulls = options => github.paginate(github.rest.pulls.list, { ...repository, per_page: 100, ...options });
  const candidates = Object.entries(SOURCE_BRANCHES);
  if (await getBranch(RELEASE_BRANCH)) {
    const merged = (await listPulls({ state: 'closed', base: 'main', head: `${owner}:${RELEASE_BRANCH}` }))
      .filter(pr => pr.merged_at && pr.base.ref === 'main' && pr.head.ref === RELEASE_BRANCH &&
        pr.head.repo?.full_name === `${owner}/${repo}`);
    if (merged.length !== 1) throw new Error('Release branch must have exactly one merged PR into main');
    candidates.push([RELEASE_BRANCH, merged[0].head.sha]);
  }

  const validateBranch = async (branch, expected) => {
    if (branch === 'main') throw new Error('Refusing to delete main');
    const result = await getBranch(branch);
    if (!result) return false;
    if (result.data.protected) throw new Error(`Protected branch retained: ${branch}`);
    if (result.data.commit.sha !== expected) throw new Error(`Branch changed; retained: ${branch}`);
    if ((await listPulls({ state: 'open', head: `${owner}:${branch}` })).length) {
      throw new Error(`Open PR still uses branch: ${branch}`);
    }
    const comparison = await github.rest.repos.compareCommitsWithBasehead({
      ...repository, basehead: `${expected}...${sha}`,
    });
    if (!['ahead', 'identical'].includes(comparison.data.status) || comparison.data.behind_by !== 0) {
      throw new Error(`Branch is not fully incorporated into main: ${branch}`);
    }
    return true;
  };
  // Validate the entire cleanup set before creating tags, a release, or deletions.
  const cleanup = [];
  for (const [branch, expected] of candidates) {
    if (await validateBranch(branch, expected)) cleanup.push([branch, expected]);
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
  const tags = [['v01.06.03', PREVIOUS_SHA], [`v${VERSION}`, sha]];
  const missingTags = [];
  for (const [tag, target] of tags) {
    const existing = await tagTarget(tag);
    if (existing && existing !== target) throw new Error(`Existing tag ${tag} points elsewhere; refusing to move it`);
    if (!existing) missingTags.push([tag, target]);
  }
  const existingRelease = await absentOn404(() => github.rest.repos.getReleaseByTag({ ...repository, tag: `v${VERSION}` }));
  if (existingRelease && (existingRelease.data.draft || existingRelease.data.prerelease)) {
    throw new Error('Existing release is a draft or prerelease; refusing to finalize cleanup');
  }

  for (const [tag, target] of missingTags) {
    await assertMain();
    // createRef never overwrites an existing ref. A concurrent creation fails safely.
    await github.rest.git.createRef({ ...repository, ref: `refs/tags/${tag}`, sha: target });
    core.info(`Preserved ${tag} at ${target}`);
  }
  if (!existingRelease) {
    await assertMain();
    await github.rest.repos.createRelease({
      ...repository, tag_name: `v${VERSION}`, target_commitish: sha,
      name: `Sovereign Conquest ${VERSION}`, body: notes, draft: false, prerelease: false,
    });
  }

  const deleted = [];
  for (const [branch, expected] of cleanup) {
    await assertMain();
    if (!await validateBranch(branch, expected)) continue;
    // GitHub deleteRef has no conditional-SHA option. Re-read immediately before
    // deletion; concurrent branch writers must not repurpose these release refs.
    const latest = await getBranch(branch);
    if (!latest) continue;
    if (latest.data.protected || latest.data.commit.sha !== expected) {
      throw new Error(`Branch changed during cleanup; retained: ${branch}`);
    }
    await assertMain();
    await absentOn404(() => github.rest.git.deleteRef({ ...repository, ref: `heads/${branch}` }));
    deleted.push(branch);
    core.info(`Deleted incorporated branch ${branch} at ${expected}`);
  }
  return { version: VERSION, sha, deleted };
}

module.exports = finalize;
module.exports.SOURCE_BRANCHES = SOURCE_BRANCHES;
module.exports.REQUIRED_WORKFLOWS = REQUIRED_WORKFLOWS;
module.exports.PREVIOUS_SHA = PREVIOUS_SHA;
