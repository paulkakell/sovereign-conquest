# Branch audit for 01.06.04

Reviewed 2026-09-27 against main `03625c68a141f649aa87480b13d61527febceff1`.
All nine source branches contained unique work; none was obsolete before integration.
They become cleanup candidates only after the exact tips below are ancestors of
validated main and their pull requests are no longer open.

| PR | Branch | Reviewed source commit | Decision |
| --- | --- | --- | --- |
| #22 | `docs/roadmap-00.02.01` | `d21268f7fc1527695081c492509f89eb2dcf6f95` | Integrate in 01.06.04, then remove incorporated ref |
| #21 | `dependabot/go_modules/server/github.com/jackc/pgx/v5-5.11.0` | `f0545239667ed3e06f9870336a62106a5f6defb4` | Integrate in 01.06.04, then remove incorporated ref |
| #20 | `dependabot/go_modules/server/golang.org/x/crypto-0.57.0` | `ce83b04070601b44456d42d5bfdcb91905080b4c` | Integrate in 01.06.04, then remove incorporated ref |
| #19 | `dependabot/docker/server/golang-1.27.1-alpine3.24` | `77fc2ee76b175db42e65d280e235472e75d97f55` | Integrate in 01.06.04, then remove incorporated ref |
| #18 | `dependabot/docker/golang-1.27.1-alpine3.24` | `9d7ece09d1baeaf18ed98ff786780fbcf4ccbe95` | Integrate in 01.06.04, then remove incorporated ref |
| #14 | `dependabot/go_modules/server/github.com/go-chi/chi/v5-5.3.2` | `19ba0748a0973980d51e186b2132566c2ee1bf43` | Integrate in 01.06.04, then remove incorporated ref |
| #13 | `dependabot/github_actions/actions/attest-build-provenance-4` | `a3eac5b108720e441e4b0c13bd58b2f995abf418` | Integrate in 01.06.04, then remove incorporated ref |
| #12 | `dependabot/github_actions/aquasecurity/trivy-action-0.36.0` | `7088569329e7fda3e22ba394c4e380ed7799623b` | Integrate in 01.06.04, then remove incorporated ref |
| #11 | `dependabot/github_actions/docker/setup-buildx-action-4` | `202f886af9b9e52eb5eff61dbd3214a773bfb0e3` | Integrate in 01.06.04, then remove incorporated ref |

## Required companion changes

- PRs #11, #12, and #13 failed tests because the workflow contract asserted the old action versions. Update those assertions together with the actions.
- PR #18 failed the combined Dockerfile test's old compiler assertion. Align both Docker builders, CI, CodeQL, and the compiler assertion at Go 1.27.1.
- Resolve the shared `go.mod` conflict by retaining chi 5.3.2, pgx 5.11.0, and x/crypto 0.57.0. Verify regenerated vendor content matches the integrated sources.
- The additional automated AI review failed on PR #22 because its requested model was unsupported. Its job log reports an HTTP 400 model error, not a code finding. CI, containers, and CodeQL passed on that source commit. The independent code review is recorded separately.

## Recovery and final state

The release commit preserves every source tip in its parent ancestry. Branch deletion
removes names, not incorporated commits. Recreate any listed branch from its full
source commit if necessary. A separate temporary `release-01.06.04` branch is removed
only when its exact head is proven merged into main. No wildcard deletion is used.

The one-release finalizer is deliberately retained as an auditable, idempotent record.
It skips other application versions and does not target future branches.
The merged release pull request, tag, Actions logs, and live branch listing provide
execution evidence; this document is the reviewed decision record.
