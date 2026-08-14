## 01.06.01 (2026-08-13)

Fix
- Fresh database startup now establishes the season dependency before creating player season indexes.
- Browser compatibility corrects resource transfers, citadel upgrades, corporation banking, market filters, event fields, activity fields, protected attachment downloads, and the required administrator password-change flow.
- Remove the duplicate Docker publishing workflow that contained unresolved merge markers.
- API and all-in-one containers now run as a non-root user and expose liveness health checks.

Security
- Authentication uses bcrypt cost 12 and stricter JWT algorithm, issued-at, and expiration validation.
- HTTP transport applies request-size limits, endpoint-class rate limits, forwarding-header sanitization, structured request logging, browser security headers, liveness, and database readiness probes.
- The season reset endpoint requires both a signed administrator identity and the separately configured administration key.
- Production startup rejects short signing material, short bootstrap administrator values, and database connections without transport verification.
- Dependabot monitoring covers Go modules, Dockerfiles, and GitHub Actions.

Reliability and performance
- Port and planet jobs use transaction-scoped PostgreSQL advisory locks so only one replica performs each scheduled update.
- Event generation uses a session-level leader lock across replicas.
- HTTP read, write, header, and idle timeouts are defined.
- Limiter benchmark coverage and browser contract regression tests were added.

Maintenance
- Version metadata is aligned at 01.06.01.
- Root and API Docker build stages use Go 1.26.5 and Alpine 3.22.
- CI runs JavaScript syntax checks, formatting verification, unit and regression tests, race detection, Go vet, and Compose validation.
- Release notes were added under `docs/`.

Breaking behavior
- Production deployments using weak defaults or database connections without transport verification no longer start.
- The destructive season reset endpoint now requires an authenticated administrator session in addition to the administration key.

Known limitation
- The repository connector rejected automated regeneration of `server/go.sum` and `server/vendor`. Dependency versions remain unchanged in this branch and must not be represented as upgraded until a trusted local checkout regenerates and validates the module graph.

Refs
- SC-DB-004, SC-WEB-005, SC-AUTH-004, SC-SEC-004, SC-RUNTIME-003, SC-CI-011
- Audit base: 7cec9eaf47e835e6555fdd3f8284c7b73b0c20c3
