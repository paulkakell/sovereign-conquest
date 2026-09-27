# Validation for 01.06.08

Date: 2026-09-27. Status: local candidate, not published or deployed.
Baseline: `c3de81af4d471e7cf327e6f43df96ff321a6b04a` (v01.06.06).
Recovered local work: `a44c249`, `9b0d400`, integrated without replacing the
01.06.06 scratch API runtime or its security/build/publication gates.

## Completed checks

| Check | Result |
| --- | --- |
| Full Go unit/regression suite | `go test -count=1 ./...` passed |
| Go race suite | `go test -race -count=1 ./...` passed |
| Go formatting and vet | Passed |
| Module graph | `go mod verify` passed; manifests/vendor unchanged |
| Gosec | Passed with no findings |
| Govulncheck | Zero reachable or imported-package vulnerabilities; unused module-level finding below |
| Release safeguard tests | 109 passed |
| Python security/recovery tests | 26 passed, including five live-recovery failure/redaction checks with a stub client |
| JavaScript syntax | All three browser scripts passed |
| Shell syntax and ShellCheck 0.11.0 | All DB scripts and release smoke script passed |
| Python integration syntax | Both recovery test files compiled |
| YAML | 22 non-vendor files parsed with duplicate-key rejection |
| Compose 5.5.1 | Both profiles rendered; literal credential mapping verified using synthetic punctuation-containing credentials |
| Tailored profile | Data bind path, networks, project name, and one published port 5000 verified |
| Fresh Go build | Go 1.27.1, empty build cache, `CGO_ENABLED=0`, `-mod=vendor`, `-trimpath` passed |
| Whitespace | `git diff --check` passed |

No supplied real credential was written into source, tests, or documentation.
Compose escapes dollar signs when serializing a reusable config; validation
accounted for that representation without changing literal credential inputs.

## Unavailable gates

The PostgreSQL container integration suite was invoked and stopped before any
tests because the Docker executable is unavailable. No database integration
success is claimed. Docker image builds, runtime inventory, image vulnerability
scans, API/database smoke tests, and measured startup/rollback timings therefore
remain pending. Hosted CodeQL and GitHub workflow gates have not run for this
candidate. No production host or actual conquest data was accessed.

Run `python3 scripts/test-db-startup.py` in a Docker environment before deploying
the startup wrapper. Its 15 tests use uniquely named disposable volumes and cover
fresh/existing databases, missing roles/databases, password rotation and rollback,
quoted values, legacy HBA trust, idempotency, expired/disabled logins, retained
rows, failed repair, readiness, abrupt restart, and the two live-repair scenarios.
The script enforces a 90-second readiness bound per startup; measured performance
is unavailable here. The full suite is required in the publication workflow.

Full image builds and application smoke tests are retained in Build Validation.
The release finalizer requires CI, Build Validation, Publish GHCR Image, and
Database Startup for the exact main commit, preserves previous tags, and creates
`v01.06.08` only after those gates succeed. The source candidate is traceable by
its local branch `release-01.06.08-db-recovery` and Git commits. The known published
baseline remains `v01.06.06`; no 01.06.08 tag or image is claimed.

## Security, compatibility, and operations review

Authentication: repair requires an existing local administrator session; errors
are redacted and failed TCP verification exits nonzero. Passwords are read from
the container environment, quoted by psql, and excluded from process arguments.
The live-repair command always sets the configured password because localhost
trust can make a password check succeed incorrectly. It does not change HBA
or grant additional role privileges. A successful TCP query also confirms DB
access. The startup wrapper enforces SCRAM for the final loopback health probe. The startup
wrapper uses a private administrator socket and SCRAM on loopback; other retained HBA
settings govern non-loopback connections in the final server. Existing data and ownership are retained.

Authorization, API validation, user password hashing, game queries, and request
logging are unchanged. The API retains its scratch runtime, non-root user,
read-only filesystem, dropped capabilities, and native health check. DB logs
use fixed structured events without user-supplied credentials. API readiness
and liveness endpoints are unchanged. The supplied external network is retained;
this does not add TLS or turn the existing development profile into production.

Dependency manifests and vendored sources are unchanged. Govulncheck reports
GO-2026-5932 only for the unused `golang.org/x/crypto/openpgp` package within an
existing required module; it reports zero findings in imported packages or
reachable symbols. No new dependency or vulnerability suppression was added.
Container scanning remains pending and is not represented by the Go scan.

No schema migration, data-format conversion, or gameplay API change occurs.
Root Compose retains both original distinct host-port options. The tailored
profile uses only WEB_PORT and defaults to the existing 01.06.06 application
image. The pgx environment credential behavior is covered by its Go regression.
Startup scripts are mounted separately, and a missing script directory fails
configuration startup. Ongoing DB health performs one `SELECT 1` per five seconds.
No core gameplay query or request-I/O performance path was altered.

The recovery guide defines deployment checks and rollback. Password changes
persist in PostgreSQL and are not undone by changing the API image. Retain the
previous environment, image digest, and database backup; restore the previous
credential through the wrapper before removing it. Prior source tags remain
available. Prior container artifacts could not be pulled without Docker.
