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
| `node --test scripts/*.test.cjs` | 89 tests passed, including the required database publication gate |
| Container security policy tests | 21 Python tests passed |
| Browser JavaScript and Bash syntax | Passed |
| ShellCheck 0.11.0 | All database scripts and release smoke script passed |
| Python integration runner syntax | Compiled successfully |
| YAML | 20 files parsed with duplicate-key rejection |
| Docker Compose 5.5.1 | Configuration validated; required credentials, read-only script mount, credential mapping, and health dependency checked |
| Clean Go build | Go 1.27.1, empty build cache, `CGO_ENABLED=0`, `-trimpath`, `-mod=vendor` passed |
| Whitespace | `git diff --check` passed |
| Previous release | Git tag and published GitHub release `v01.06.06` verified available; prior container was not pulled locally |

## Completed hosted checks

The following PR runs passed for source commit
`42fbf5ce59c0f3be28dac9695ff3fc4f92f36edb`. The subsequent validation-record
commit changes documentation only; consult PR #27 for its own exact-commit runs.

| Workflow | Evidence and result |
| --- | --- |
| [CI](https://github.com/paulkakell/sovereign-conquest/actions/runs/36350289773) | All unit/regression and race tests, release safeguards, formatting, module verification, vet, Gosec, Govulncheck, and Compose validation passed |
| [Database Startup](https://github.com/paulkakell/sovereign-conquest/actions/runs/36350289727) | ShellCheck and all 13 PostgreSQL 16 container scenarios passed in 40.6 seconds |
| [Build Validation](https://github.com/paulkakell/sovereign-conquest/actions/runs/36350289846) | Clean combined/API/web images built; both API runtime inventories, vulnerability policies, SBOM generation, and application smoke tests passed; CodeQL passed |

The database suite includes empty and populated volumes, unchanged restart,
password rotation and reversal, missing database/login, renamed bootstrap
administrator, disabled/expired login, quoted identifiers and passwords, legacy
trust rules, invalid configuration, failed repair, readiness failures, and abrupt
restart. Existing-volume readiness took approximately 0.4 to 0.73 seconds;
fresh initialization took 1.68 to 1.70 seconds on that runner, excluding image
pull time. This is startup evidence, not an application load-test claim.

The initial container run exposed localhost trust bypass and a disabled bootstrap
login preventing repair. Commit `42fbf5c` fixes both; the successful scenarios
verify rejection of wrong/old passwords and preservation of stored game rows.

GitHub's separate [AI code-scanning run](https://github.com/paulkakell/sovereign-conquest/actions/runs/36350290648)
could not execute: its service returned HTTP 400, "The requested model is not
supported." That check remains a service failure, not a passing scan. No scan was
disabled or bypassed. CodeQL, Gosec, and the container security checks passed.

Docker and PostgreSQL are unavailable locally; the runtime evidence above comes
from GitHub Actions. No production deployment, merge, new release tag, or GHCR
publication has occurred for this branch. Publication/finalization workflows
remain gated for a future merge to main.

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
The final server also requires SCRAM on IPv4 loopback through a runtime HBA
wrapper that includes the persistent HBA without changing it. Only the exact
configured bootstrap role can have NOLOGIN repaired offline; unrelated disabled
administrator accounts remain disabled.
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
