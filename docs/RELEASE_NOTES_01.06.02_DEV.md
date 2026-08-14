# Sovereign Conquest 01.06.02 Development Release

This prerelease exists for real-world validation before any merge to `main`.

## Additive
- Add a branch-scoped development release workflow.
- Publish immutable `dev-01.06.02`, moving `dev`, and source-SHA GHCR tags.
- Publish provenance, an SPDX SBOM, and a validation report.
- Verify an unauthenticated GHCR pull and a second runtime smoke test.
- Add CodeQL analysis.

## Fixes
- Align application, browser, test, and documentation versions at 01.06.02.
- Move build and validation jobs to Go 1.26.6.
- Correct the standalone bug-report page paths and version.

## Security
- Upgrade chi to 5.3.1, jwt/v5 to 5.3.1, pgx/v5 to 5.10.0, and x/crypto to 0.55.0.
- Regenerate `go.sum` and `server/vendor`.
- Require clean reachable-code and container vulnerability gates before publication.

## Compatibility
- No stable tag or `main` branch publication occurs.
- Development deployment requires PostgreSQL and environment-specific secrets.
- Production mode requires verified database transport.

## Known limitations
- The season-reset transaction still needs its planned atomic error-handling and durable audit-event rewrite.
- Startup DDL remains transitional pending numbered migrations.

## References
- SC-REL-012, SC-SEC-005, SC-DEP-003
- Dependency remediation: `6fe80acb730006ba834bc322b192ebd776933239`
- Audit base: `7cec9eaf47e835e6555fdd3f8284c7b73b0c20c3`
