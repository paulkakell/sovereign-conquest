# Sovereign Conquest 01.06.02 Development Release

This prerelease exists for real-world validation before any merge to `main`.

## Additive
- Add a branch-scoped development release workflow.
- Publish immutable `dev-01.06.02`, moving `dev`, and source-SHA GHCR tags.
- Publish provenance, an SPDX SBOM, and a validation report.
- Verify an unauthenticated GHCR pull and a second runtime smoke test.
- Add a pinned gosec static application security testing gate.
- Add a permanent fresh-PostgreSQL integration test that validates startup, readiness, registration, state retrieval, and a live command.

## Fixes
- Align application, browser, test, and documentation versions at 01.06.02.
- Move build and validation jobs to Go 1.26.6.
- Correct the standalone bug-report page paths and version.
- Correct the Protectorate startup query so PostgreSQL receives a contiguous `$1` parameter rather than an unused `$1` followed by `$2`.
- Replace opaque release smoke polling with health-aware container diagnostics and captured PostgreSQL and application logs.
- Resolve the pull-request CodeQL failure by excluding trusted client network addresses from structured request logs.

## Security
- Upgrade chi to 5.3.1, jwt/v5 to 5.3.1, pgx/v5 to 5.10.0, and x/crypto to 0.55.0.
- Regenerate `go.sum` and `server/vendor`.
- Require clean gosec, reachable-code, CodeQL, and container vulnerability gates before publication.
- Keep proxy-derived client addresses ephemeral: they remain available for server-side throttling and proxy normalization but are not retained in logs.
- Add a regression test that fails if a trusted client address is written to structured log output.

## Compatibility
- No stable tag or `main` branch publication occurs.
- Development deployment requires PostgreSQL and environment-specific secrets.
- Production mode requires verified database transport.
- The logging correction is non-breaking and does not change API, database, command, or configuration contracts.

## Known limitations
- The season-reset transaction still needs its planned atomic error-handling and durable audit-event rewrite.
- Startup DDL remains transitional pending numbered migrations.

## References
- SC-REL-012, SC-SEC-005, SC-DEP-003, SC-DB-005
- PR: `#1`
- GitHub Code Scanning alert: `#1`
- Dependency remediation: `6fe80acb730006ba834bc322b192ebd776933239`
- Protectorate startup fix: `2e8bfc0e4ae8efad001643dbb8b48b90fe05e5b0`
- Logging remediation: `e9e49bce495c5f7619063b9305cda2b9caaf9b3e`
- Regression test: `7df590cae5bd734929b43e0a481b999c36af09e3`
- Audit base: `7cec9eaf47e835e6555fdd3f8284c7b73b0c20c3`
