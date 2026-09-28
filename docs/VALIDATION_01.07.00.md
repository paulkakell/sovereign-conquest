# Validation for 01.07.00

Base: v01.06.13 / 609a60c. Branch: feature/admin-user-management.

## Local validation

Completed locally with Go 1.27.1:

- Full available Go unit/regression and race suites, go vet, gofmt and module verification.
- Gosec: no findings. Govulncheck: zero reachable or imported-package vulnerabilities; one advisory affects an unused package in a required module.
- All 248 JavaScript tests, including nine account form tests and 20 release safeguards.
- All 26 Python container-policy and database-recovery failure tests.
- JavaScript and shell syntax, HTML ID/asset wiring, and diff whitespace checks.
- Clean CGO-disabled API build from the committed source archive with an empty build cache, vendored modules, GOPROXY=off, GOSUMDB=off and the normal buildvcs=false flag. Result is a statically linked Linux executable.

Docker/PostgreSQL are unavailable in this local environment. Seven real
PostgreSQL integration tests are implemented but skipped locally because
SC_TEST_DATABASE_URL is unset. CI is configured to provision PostgreSQL 16 for the
full Go suite/race run and the parallel session lookup benchmark. Cases include
real authorization, updates/rollback, password resets, stale token denial,
suspension/ban/restoration, concurrent demotions, pagination, bootstrap rename
and migration reversal.

Hosted PostgreSQL/benchmark, clean container builds, Compose/database startup,
image vulnerability and CodeQL checks remain pending. Automatic approval review
rejected the branch push, requiring explicit authorization before publishing this
commit to the public repository. No branch, pull request, release tag or image
was published for 01.07.00. The existing v01.06.13 immutable GitHub release and
source target 609a60c6b932784ebb02dc461653f8501bb8a967 were verified as available.

Implementation commit: 5ed8d261c38cc34a6fc2dda0871080b5f12ca257. A following
local documentation commit records these results and corrects rollback guidance.

The control-browser preview was blocked by ERR_BLOCKED_BY_CLIENT for localhost.
No rendered visual review is claimed. Automated DOM component tests exercise
changed-field saves, byte/precision handling, password resets, moderation forms,
self-lockout controls and late-response cleanup.

## Security and compatibility review

Administrator roles/status are read from PostgreSQL, not trusted from JWT claims
or browser controls. The actor is rechecked under a transaction lock for admin
writes. Existing sessions end on credential, role or moderation changes, including
bootstrap credential recovery. Soft wipe retains its separate secret requirement.
Password-required accounts are limited to state/password change.

SQL uses bound values and explicit field allowlists. Body sizes, data types,
numeric ranges, cargo/turn invariants, foreign keys and revision conflicts are
checked. Frontend renders account strings with textContent. Audit and structured
logs exclude credentials/hashes. There are no runtime dependency changes.

Migration is additive, idempotent and defaults existing accounts to active.
Account and player edits plus audit are atomic. IDs and system timestamps remain
immutable. The former API routes and response fields remain, with an additive
replacement token on password changes. The client must use it for later requests.

Performance impact is one indexed session query per authenticated request.
Pagination is bounded at 50; admin mutation locking does not lock normal gameplay.
Configuration, health probes, existing logging and metrics remain stable.

Rollback must close public ingress because older binaries ignore access restrictions.
See USER_MANAGEMENT.md. The v01.06.13 source/image baseline is retained.
Hosted gates supply fresh image builds, Compose/database startup tests, image
vulnerability scans and CodeQL before any release tag/publication is finalized.
