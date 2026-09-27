# Sovereign Conquest 01.06.08

Publication requires successful validation, container security, and database
startup workflows for the release commit.

## Fixes

Integrate the separately developed 01.06.07 database startup reconciliation with the
01.06.06 security release. Existing PostgreSQL data remains in place while
configured databases, logins, and passwords are checked on every startup.
API startup waits for an authenticated database health check.

Add a tailored Compose profile for the supplied bind-mounted deployment,
preserving its networks and data path and publishing port 5000 once. Supply
database credentials literally through libpq environment settings.

## Additive

Add a one-time password repair command for a running PostgreSQL container,
failure/redaction regression tests, and disposable-container tests for live
repair, idempotency, missing local administrator, and retained rows. Require
database regression tests before publishing image candidates. Add deployment,
credential rotation, and rollback instructions in the [recovery guide](DB_RECOVERY_01.06.08.md).

## Compatibility and release gates

No breaking gameplay/API/schema/dependency change. Root Compose retains its
previous two distinct port options; the tailored profile uses only WEB_PORT.
Startup scripts must be installed for the new Compose configuration to work.
New roles created during startup receive ordinary login privileges; existing
object ownership is retained. Changing application identity may need separate
schema grants. The earlier removal of API shell/package utilities remains.

The recovery profile can use the existing 01.06.06 application image; updating
the API alone does not install DB startup scripts. The source/application
version is 01.06.08. Its Git tag is `v01.06.08`, and its image
tag is `01.06.08`, gated on successful workflows for the exact commit.
Previous tags remain unchanged.

All 15 PostgreSQL integration tests, CI, container builds and scans, application
smoke tests, and CodeQL passed on the code commit recorded in validation.
GitHub's separate AI review could not start because its configured model was
unsupported. See [validation](VALIDATION_01.06.08.md) for exact workflow evidence,
[commit notes](COMMIT_NOTES_01.06.08.md), and the recovery guide for rollback.
Baseline: `c3de81af4d471e7cf327e6f43df96ff321a6b04a` (v01.06.06, PR #26).
Prior local work: `a44c249` and `9b0d400`. Release pull request: #28. Includes the startup hardening from PR #27.
No separate issue was opened.
