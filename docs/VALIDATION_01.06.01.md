# Validation Record 01.06.01

Audit base: `7cec9eaf47e835e6555fdd3f8284c7b73b0c20c3`

Branch: `release-01.06.01-fixes`

## Automated gates

The permanent CI workflow performs:

- JavaScript syntax checks for the primary client, compatibility layer, and bug-report client;
- Go formatting verification;
- complete Go unit and regression tests;
- race detection across all Go packages;
- `go vet` static analysis;
- Docker Compose configuration validation.

The permanent Build Validation workflow builds:

- the combined API and web image from the repository root;
- the split API image;
- the split web image.

The core repair workflow also ran the complete Go test suite before committing the schema and authentication changes. Browser compatibility and container regression tests were added to the repository.

## Configuration and database checks

- Production startup validation rejects short signing material, short bootstrap administrator values, unsupported universe settings, and database transport without verification.
- Fresh database initialization prepares the season dependency before the legacy schema creates dependent indexes.
- Liveness and database-readiness endpoints are available for deployment probes.
- Port and planet schedulers use transaction-scoped PostgreSQL advisory locks.
- Event generation uses a session-level PostgreSQL leader lock.

## Security review status

Implemented controls include stronger password hashing, restricted JWT algorithms, expiration and issued-at validation, request-size limits, endpoint-class throttling, forwarding-header sanitization, browser security headers, non-root API containers, and current administrator verification for the season-reset endpoint.

## Dependency status

Dependency remediation is not complete. `server/go.mod`, `server/go.sum`, and `server/vendor` were not regenerated because the repository connector rejected the required operation. Dependabot monitoring is configured, but version 01.06.01 must not be tagged or released until a trusted checkout performs:

```text
go mod tidy
go mod vendor
go mod verify
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
govulncheck ./...
```

Commit the resulting module and vendor changes only after all commands pass and the diff contains no unexpected dependency additions.

## Remaining limitations

- The existing season-reset transaction still needs a full atomic-error-handling rewrite and durable audit insertion.
- Session-version claims are not yet compared with a persistent revocation counter.
- The Protectorate scheduler remains on its legacy execution path.
- Several older JSON handlers still accept unknown fields.
- Startup DDL should eventually be replaced with numbered migrations.
- Repository-level code scanning and secret scanning could not be enabled or verified through the connector.

## Release decision

This branch is a release candidate, not a completed release. Promotion remains blocked until dependency regeneration and vulnerability validation are complete and the final CI and Build Validation runs are successful.
