# Validation and security review for 01.06.10

Base: `1b1b03d0eb8bb72efd4d6079292798eb2bd718d3` (v01.06.09, PR #29).
Requested work: validate four secrets at API startup with descriptive container
logs and correct the bcrypt length mismatch. No separate issue was opened.

## Validation status

Local Go unit/regression tests passed across all packages using Go 1.27.1.
All 149 release-safeguard JavaScript tests and 35 Python Compose, container-policy
and database-recovery tests passed. JavaScript and shell syntax checks and
`git diff --check` passed. The full race suite, Go vet, module verification,
Gosec and a static CGO-disabled API build also passed locally. Govulncheck found
zero reachable or imported-package vulnerabilities; its module-only advisory
GO-2026-5932 concerns the unused `golang.org/x/crypto/openpgp` package. No
dependency update is needed for the bcrypt package used here. Hosted
container/security workflows must also pass before publication.
The repository includes unit, subprocess and integration coverage for:

- Empty and boundary credentials in development, production and unset APP_ENV.
- Optional empty reset key, whitespace handling, 72/73 bytes and Unicode bytes.
- Effective URL/environment database credentials and redacted parser failures.
- Startup exit status, aggregate diagnostics and rejection before database access.
- Registration/password-change 400 errors before database work.
- Valid 8/72-byte and Unicode password round trips against fresh PostgreSQL.
- Rejected changes preserving old credentials and successful boundary changes.
- Both scratch images, native probes, Compose contracts and release safeguards.

## Security and compatibility review

Authentication: bcrypt cost and hashes, JWT signing algorithm and login behavior
are unchanged. The minimum account password remains 8 bytes; bootstrap requires
16. Shared password validation uses the actual 72-byte bcrypt maximum.
Authorization: administrator session/key checks and route protections are intact.
An empty reset key still disables the destructive endpoint.

Input/logging: fixed diagnostics identify variables and ranges without secrets,
hashes, URLs, or raw driver parse errors. Length validation occurs before database
initialization and HTTP serving. Credentials use pgx's effective precedence.
Configuration fallback credentials are removed. Neither production TLS checks
nor the healthcheck's configuration-independent path is weakened.

Dependencies: no manifests, lock files or vendor packages change. CI runs module
verification, Go vet, gosec, govulncheck, CodeQL and image vulnerability scans.
Release gates include unfixed vulnerabilities and preserve existing policies.

Database/rollback: no migration or stored-data change is introduced. The prior
source tag and published 01.06.09 release were verified before implementation.
Use the prior verified image and retained environment files for rollback without
deleting data. Code rollback does not reverse intentionally rotated credentials.

Performance/observability: startup adds one local connection-configuration parse
and bounded secret length checks, without a database connection. Request length
checks occur before hashing; hashing cost and database queries are unchanged.
Existing health/readiness endpoints and metrics are retained. Container smoke
tests cover startup and auth requests; no gameplay/load-path change needs a new
load benchmark. Invalid startup exits 1 and keeps the existing log prefix.

## Hosted validation evidence

Validated source: `1147a87f33ce28d4d8e184bef4c3260984f710b8`, PR #30.
All repository validation gates passed on this source before the final
documentation-only evidence update. Main publication reruns the same gates.
The API/runtime/configuration implementation is unchanged by the subsequent
validation-record and integration-fixture update. The first container run passed
startup validation, builds, inventory checks, scans and CodeQL, then exceeded the
existing authentication rate limit during repeated password tests. The fixture
now uses exactly nine register/login requests plus the original smoke registration,
within the existing 10-request window. Rate limiting is unchanged and enabled.

| Gate | Evidence |
|---|---|
| CI | Passed: [Run 36366629648](https://github.com/paulkakell/sovereign-conquest/actions/runs/36366629648). Unit/race tests, formatting, module verification, vet, Gosec, Govulncheck and Compose checks. |
| Build Validation | Passed: [Run 36366629647](https://github.com/paulkakell/sovereign-conquest/actions/runs/36366629647). Three image builds, both API inventories/scans/SBOMs, both startup/password integration suites and CodeQL. |
| Database Startup | Passed: [Run 36366629650](https://github.com/paulkakell/sovereign-conquest/actions/runs/36366629650). Shell lint and all 15 PostgreSQL startup/recovery regressions. |

GitHub's additional AI findings job failed before analysis because its configured
model was unsupported (`CAPIError: 400 The requested model is not supported`).
This is separate from CodeQL, Gosec, dependency/image scans and the repository's
required release gates. No workflow or approval policy was disabled or relaxed.
[Service failure log](https://github.com/paulkakell/sovereign-conquest/actions/runs/36366314726).
