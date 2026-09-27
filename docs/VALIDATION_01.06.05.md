# Validation for 01.06.05

Review date: 2026-09-27. Baseline: `d3a8e23c9e7975895d55d21cb8d513821cacc9d9`
(`v01.06.04`, PR #23). This record separates local checks from hosted release gates.

## Completed local checks

| Check | Result |
| --- | --- |
| `node --test scripts/*.test.cjs` | All 44 tests passed, including 20 new release safeguards |
| All `.github/*.yml` files, recursively | 16 files parsed with duplicate-key rejection |
| Issue forms | Three forms have required names, descriptions, unique field IDs and labels, supported field types, and boolean validations |
| Discussion forms | Five filenames match the live category slugs; Polls intentionally has no form |
| Chooser and funding | Contact URLs target this repository; blank issues disabled; funding recipient is `paulkakell` |
| Browser and release JavaScript | Syntax checks passed |
| Release smoke/build shell scripts | Syntax checks passed |
| Release metadata | Root, API, web assets, web contract assertions, and smoke-test version aligned at 01.06.05 |
| Whitespace | `git diff --check` passed |

The local environment has no Go toolchain, Docker daemon, or PostgreSQL server.
Downloading the official Go toolchain was unavailable under its network policy.
No local Go-suite, dependency-scan, container-build, or database-runtime pass is
claimed. These checks run in the existing hosted workflows before release.

## Hosted gates

| Workflow | Required evidence |
| --- | --- |
| CI | JavaScript syntax, all release tests, Go formatting, module checksums, full unit/regression suite, race detection, vet, Gosec, Govulncheck, and Compose configuration |
| Build Validation | Combined, API-only, and web-only clean container builds; fresh PostgreSQL integration; CodeQL |
| Publish GHCR Image | Build and publication, provenance attestation, high/critical Trivy gate, and published-image smoke test |
| Finalize Release 01.06.05 | All three workflows successful on the exact current main SHA before creating tag and release |

Use the pull request checks and the
[Actions page](https://github.com/paulkakell/sovereign-conquest/actions) for the
final commit's results. This source record is written before those runs and does
not substitute for their recorded outcomes. A future failed run must not be
represented as passed based on these local results.

The release-finalization tests cover missing or failed gates, unfinished runs,
wrong commits and workflow paths, pull-request events, later failed runs, fork
triggers, moved main, immutable tags, prior release preservation, repeat execution,
draft/prerelease conflicts, API errors, and empty release notes. The older 24
branch-consolidation safeguards remain intact.

## Security review

Public forms warn against posting credentials, complete environment files,
private player data, or exploit instructions. The security policy uses the
repository's verified private-reporting endpoint and invents no contact address,
bounty, or response guarantee. Funding targets the verified repository owner;
no unrelated donation destination is configured.

The new publication workflow has only `actions: read` and `contents: write`.
It checks out the triggering SHA with persisted credentials disabled, accepts
successful workflows from this repository's main branch, validates required
workflow paths and exact revisions, and rechecks main before each write. It
does not run pull-request source under a privileged trigger, delete refs,
change permissions, or create a payment account. Existing tags are never moved.

Authentication, authorization, game input validation, logging, deployment
secrets, and dependency declarations are unchanged except reported version
metadata. The existing findings in `CODE_REVIEW_01.06.04.md` remain unresolved.
The previous scan documented a module-only advisory in an unused OpenPGP package;
use the fresh hosted Govulncheck output for this release's actual finding set.
Do not describe unchanged dependencies as universally vulnerability-free.

## Configuration, compatibility, and operations

No environment variable, feature flag, default gameplay setting, API contract,
command, query, schema, or migration changes. Dependency manifests, checksums,
and vendored files remain identical to the baseline. Rebuilding lockfiles is
unnecessary. Source and runtime compatibility remain subject to the hosted gates.

No core logic or I/O changes, so additional performance/load testing is not
required for this scope. Existing logging assertions run in the Go suite; no
claim is made about external metrics or alerts on a deployed server. No live
database upgrade or restore is performed by this community configuration release.

Discussions and private vulnerability reporting were verified enabled. Category
names and slugs were read from GitHub. Sponsors enrollment and payout status are
not established by adding `FUNDING.yml`; verify the actual destination separately.
GitHub form rendering must be checked after the files reach the default branch.

## Rollback

The previous Git tag and published GitHub release `v01.06.04` were verified
available. The prior workflow recorded the source-specific application image;
this local environment did not pull it. Revert the release merge or select the
previous image as described in `RELEASE_NOTES_01.06.05.md`. No database reversal
is needed. Preserve existing Discussions and private reporting when reverting
the community files.
