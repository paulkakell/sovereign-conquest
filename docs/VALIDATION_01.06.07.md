# Validation for 01.06.07

Baseline: `c3de81af4d471e7cf327e6f43df96ff321a6b04a` (v01.06.06, PRs #25 and #26).
Rebased onto the released API security fixes before publication.
Validation date: 2026-09-27.

## Completed local checks

| Check | Result |
| --- | --- |
| `go test -count=1 ./...` | All packages passed, including literal-credential regression |
| `go test -race -count=1 ./...` | All packages passed |
| Go formatting and `go vet ./...` | Passed |
| `go mod verify` | All modules verified |
| Gosec | Passed without findings |
| Govulncheck | Zero reachable or imported-package vulnerabilities; module-only GO-2026-5932 concerns unused `x/crypto/openpgp` |
| `node --test scripts/*.test.cjs` | 64 tests passed, including the required database publication gate |
| Browser JavaScript and Bash syntax | Passed |
| ShellCheck 0.11.0 | All database scripts and release smoke script passed |
| Python integration runner syntax | Compiled successfully |
| YAML | 19 files parsed with duplicate-key rejection |
| Docker Compose 5.5.1 | Configuration validated; required credentials, read-only script mount, credential mapping, and health dependency checked |
| Clean Go build | Go 1.27.1, empty build cache, `CGO_ENABLED=0`, `-trimpath`, `-mod=vendor` passed |
| Whitespace | `git diff --check` passed |
| Previous release | Git tag and published GitHub release `v01.06.05` verified available; prior container was not pulled |

Docker and PostgreSQL are unavailable in the editing environment. Actual
PostgreSQL integration, image builds, container vulnerability scans, startup
timings, and hosted CodeQL are configured in GitHub Actions. Hosted outcomes will
be recorded when runs complete. No deployment or release-tag publication is
claimed by these local results.

## Required gates

- Full Go unit/regression suite, race detector, formatting, vet, module verification,
  Gosec, and reachable-code vulnerability scan.
- Browser JavaScript syntax and all release finalization safeguards.
- ShellCheck and actual PostgreSQL container startup/regression tests.
- Fresh combined/API/web image builds, application smoke test using the new
  database startup scripts, Compose validation, and CodeQL.
- Image vulnerability scan and published-image integration before release tagging.

## Security and compatibility review

The new path repairs only the configured database/login. SQL names and secrets
use psql identifier/literal quoting with values from the environment. The
temporary administrative socket is private to the database OS user. Temporary
TCP binds only loopback and uses SCRAM. Public readiness cannot succeed until
reconciliation finishes. Failed repair does not permit normal database startup.
No database/table/role is dropped. Existing privileges and ownership are retained;
new repair-created roles are ordinary logins. Existing bootstrap superuser
behavior is retained for compatibility with the official image.

Structured events contain no supplied names or passwords. Private temporary
diagnostics are deleted, and SQL statement logging is disabled during repair.
Environment values remain readable by Docker administrators, as before.
Application authentication, authorization, logging, rate limits, API formats,
and gameplay settings are unchanged. Literal Compose credential handling has a
pgx parsing regression test. Existing security findings remain outside this fix.

No dependency manifests or vendored code change; no lockfile regeneration is
required. No schema migration or stored data-format conversion is introduced.
The regression suite tests preservation of existing rows and reversal of password
changes. Existing schema ownership is deliberately preserved and may require
operator review when changing application usernames.

Startup adds a bounded local server phase and credential query. Ongoing database
health uses one `SELECT 1` every five seconds. Gameplay queries and request paths
are unchanged. The container suite measures completion within 90 seconds per
startup and includes abrupt restart recovery. Prior version tags and releases
remain unchanged. Rollback details are in `DATABASE_STARTUP.md`.
