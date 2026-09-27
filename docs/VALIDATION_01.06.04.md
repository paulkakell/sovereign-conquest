# Validation for 01.06.04

Review date: 2026-09-27. Baseline: `03625c68a141f649aa87480b13d61527febceff1`.

These results cover the locally integrated release working tree after the nine
branch merges, dependency reconciliation, version update, and build-contract test
updates. Release edits were still uncommitted during local validation.
Hosted checks on the final pull-request head remain required. This document does
not claim that local checks are evidence for a later, changed source revision.

## Toolchain provenance

The official `go1.27.1.linux-amd64.tar.gz` distribution was downloaded from
`https://go.dev/dl/go1.27.1.linux-amd64.tar.gz`. Its SHA-256 matched the entry from
`https://go.dev/dl/?mode=json&include=all` before extraction:

```text
63d339f0da5ab53635a56f2490a7984dfe12dfcff22ad749f63edaf590168445
```

`go version` reported `go1.27.1 linux/amd64`. Both Docker builders and CI use Go
1.27.1. The module's minimum language/toolchain requirement is Go 1.26.0.
Gosec v2.28.0 and govulncheck v1.7.0 were installed at the versions pinned by CI.

## Completed local checks

Commands below ran from `server/` unless indicated. The toolchain directory was
prepended to `PATH`; scanner binaries were called by absolute path.

| Check | Command | Result |
|---|---|---|
| Unit and regression suite | `go test -count=1 -coverprofile=<scratch>/coverage.out ./...` | Passed all packages |
| Race detector | `go test -race -count=1 ./...` | Passed all packages |
| Static analysis | `go vet ./...` | Passed, no diagnostics |
| Security static analysis | `gosec -quiet ./...` | Passed, no reported findings |
| Dependency checksums | `go mod verify` | All modules verified |
| Reachable vulnerabilities | `govulncheck ./...` and `govulncheck -show verbose ./...` | Zero reachable or imported-package vulnerabilities; one module-only advisory described below |
| Go formatting, repository root | `gofmt -l server` | No unformatted files |
| Browser syntax, repository root | `node --check web/static/app.js`, `node --check web/static/compat-010601.js`, `node --check web/static/bug.js` | All passed |
| Release-finalization safeguards, repository root | `node --test scripts/finalize-release.test.cjs` | All 24 tests passed |
| Release-finalization syntax, repository root | `node --check scripts/finalize-release.cjs`, `node --check scripts/finalize-release.test.cjs` | Both passed |
| Shell syntax, repository root | `bash -n scripts/release-smoke.sh`, `sh -n server/scripts/build_api.sh` | Both passed |
| Whitespace, repository root | `git diff --check` | Passed |
| Existing performance benchmark | `go test -run '^$' -bench BenchmarkRequestLimiterAllow -benchmem ./internal/api` | Passed; 87.00 ns/op, 0 B/op, 0 allocs/op |

The production binary also built with an initially empty compiler cache, the
network module proxy disabled, and the committed vendor tree:

```bash
CGO_ENABLED=0 \
GOCACHE=<new-empty-scratch-directory> \
GOPROXY=off \
GOTOOLCHAIN=local \
go build -mod=vendor -trimpath -buildvcs=false \
  -o <scratch>/sovereign-api ./cmd/api
```

`go version -m <scratch>/sovereign-api` confirmed Go 1.27.1, Linux amd64, CGO
disabled, and the expected dependency versions. This validates the standalone
binary build without relying on previously compiled local packages. It does not
replace the container build and runtime checks.

To check vendored content, a separate source copy without `vendor/` was created
outside the repository. Running `go mod vendor` there, followed by
`diff -qr <repository>/server/vendor <copy>/vendor`, produced no differences.
The reconciled vendored files therefore match the resolved module graph.

## Coverage and security limits

The suite measured 9.8% total statement coverage. Package coverage was 20.1% for
the API, 76.7% for authentication, 5.5% for game logic, and 95.5% for rules.
Configuration, database, schema, utilities, and the executable entry point had
0.0% statement coverage. Build and SQL source-contract tests contain no measured
application statements. Passing this suite is not comprehensive gameplay or
database correctness evidence.

The release-finalization tests separately cover exact revision and successful
workflow requirements, changed or protected branches, unmerged and open pull
requests, missing ancestry, immutable tags, repeated execution, moved main,
fork-origin triggers, and conflicting draft/prerelease releases. These are mocked
API tests; real GitHub finalization still depends on the hosted gates.

Govulncheck used the vulnerability database updated on 2026-09-24. It reported
[GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932) for the unused
`golang.org/x/crypto/openpgp` package within required module
`golang.org/x/crypto v0.57.0`. That package is unmaintained and has no fixed
version. The application does not import the affected package or call its
vulnerable symbols. Do not describe the dependency graph as having no known
advisories; the supported result is zero reachable and zero imported-package
vulnerabilities.

Manual review findings, including existing session-revocation, reset, scheduler,
and audit gaps, are recorded in `CODE_REVIEW_01.06.04.md`. No application security
correction is claimed for this branch-consolidation release.

## Configuration, data, performance, and compatibility

Compared with the baseline, this release changes no schema, migration, database
access code, gameplay rules, authentication code, transport logging code,
environment variable definitions, or Compose configuration. The application
configuration change is its reported version. API and stored-data compatibility
are intended to remain unchanged. Dependency updates, especially pgx, still need
the hosted database integration gate to establish runtime compatibility.

No schema rollback script is needed because this release contains no schema
change. Existing migration and startup behavior must still pass the fresh
database smoke test. A live upgrade/restore rehearsal was not performed locally.

The limiter benchmark above is a single measurement of an existing benchmark,
not a before/after performance comparison. No load test, database-query benchmark,
or multi-replica test ran locally. The dependency updates touch I/O paths, so no
latency or throughput regression claim is made.

The existing tests for structured request logging and client-address exclusion
passed as part of the API suite. Metrics, alert delivery, and deployment log
collection were not exercised against a running installation.

## Hosted gates

There is no local Docker daemon, container runtime, or PostgreSQL server. The
following remain hosted gates on the final release revision:

- Compose configuration validation.
- Combined, API-only, and web-only container builds.
- Fresh PostgreSQL initialization, application readiness, registration, state
  retrieval, and command execution through `scripts/release-smoke.sh`.
- CodeQL analysis.
- Published-image vulnerability scan, provenance attestation, and smoke test.
- Final release workflow execution and exact-head workflow/check verification.

Use the release pull request and its linked Actions runs for the final revision's
results. Do not interpret this local report as evidence that those gates passed.

## Rollback artifact

The public GHCR manifest for
`ghcr.io/paulkakell/sovereign-conquest:sha-03625c68a141` returned HTTP 200 during
this validation. Its OCI image-index digest is:

```text
sha256:da629d8ffa417656e470a1fa12af717372d8d8e57cd89ed0d78f7f46155e60e1
```

This establishes that the prior source-specific artifact remains available;
it does not claim a local image pull or rollback execution. For application
rollback, set `SC_IMAGE` to
`ghcr.io/paulkakell/sovereign-conquest@sha256:da629d8ffa417656e470a1fa12af717372d8d8e57cd89ed0d78f7f46155e60e1`
and run `docker compose pull` followed by `docker compose up -d --remove-orphans`.
Verify readiness, version, sign-in, and game state. Preserve a database backup
before deployment; no data-format reversal is required by this release.
