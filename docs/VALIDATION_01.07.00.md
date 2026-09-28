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

## Hosted validation

All release checks passed on implementation commit
`6f079814aee3d7570ae3d292300035ba3615d12a` in
[PR #34](https://github.com/paulkakell/sovereign-conquest/pull/34):

- [CI run 36382572707](https://github.com/paulkakell/sovereign-conquest/actions/runs/36382572707): full Go and race suites against PostgreSQL 16, all seven database integration tests, JavaScript/form/release tests, formatting, module verification, vet, Gosec, Govulncheck, and all nine Compose contract tests passed.
- [Build Validation run 36382572709](https://github.com/paulkakell/sovereign-conquest/actions/runs/36382572709): fresh combined/API/web image builds, runtime inventories, vulnerability policy, SBOM export, password/startup/gameplay smoke tests for both API images, and CodeQL high/critical gate passed.
- [Database Startup run 36382572667](https://github.com/paulkakell/sovereign-conquest/actions/runs/36382572667): shell lint and fresh/existing PostgreSQL volume/credential checks passed.

Database integration cases cover real authorization, account/player persistence,
atomic rejection/rollback, missing-audit rollback, password resets, stale token
denial, suspension/ban/restoration, concurrent demotions, pagination, bootstrap
rename, migration reversal and repeatable backfill. They are also run under the
Go race detector. Local tests skip these cases when SC_TEST_DATABASE_URL is unset;
they were executed successfully in the hosted service.

The four-worker parallel session benchmark completed 17,781 operations and reported
67,738 ns/op. This is a localhost PostgreSQL benchmark, not end-to-end HTTP latency
or a production load-capacity guarantee. It exercises the new indexed session
lookup without bypassing the database.

The documentation follow-up does not change application code. The PR's checks run
again for that commit, and all main-branch release workflows must pass before the
finalizer creates v01.07.00. The existing immutable v01.06.13 release and source
`609a60c6b932784ebb02dc461653f8501bb8a967` remain available for the documented
maintenance rollback.

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
