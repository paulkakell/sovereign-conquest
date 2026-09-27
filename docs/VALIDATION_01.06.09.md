# Validation for 01.06.09

Date: 2026-09-27. Local implementation commit: `547f21d`. Base: `2afa856cef71dd6f5ea5060425baba43935828be`
(v01.06.08, PR #28). Scope: all root Compose settings and `.env.example`, checked
against the application, image definitions, database scripts, and tailored
profile. No real deployment credential or running user container was accessed.

## Local checks

| Check | Result |
|---|---|
| Full Go unit/regression suite | `go test -count=1 ./...` passed |
| Full Go race suite | `go test -race -count=1 ./...` passed |
| Formatting, static analysis | gofmt clean; `go vet ./...` passed; Gosec passed |
| Dependencies | `go mod verify` passed; manifests, checksums, and vendor unchanged |
| Govulncheck 1.7.0 | Zero reachable/imported-package findings; one finding in an unused module package, described below |
| Release safeguard tests | 129 passed, including 20 new 01.06.09 gate regressions |
| Python tests | 35 passed: nine Compose, 21 container security, five database recovery |
| Compose 5.5.1 | Both profiles rendered with every documented override, required-value failure, and synthetic punctuation-containing credentials |
| JavaScript/shell/Python syntax | All applicable files passed |
| ShellCheck 0.11.0 | Database scripts and release smoke script passed |
| YAML | 23 non-vendor YAML files parsed with duplicate-key rejection |
| Isolated source build | Go 1.27.1, empty build/module caches, network module access disabled, vendored source, CGO disabled, `-trimpath`; static binary produced |
| Whitespace | `git diff --check` passed |

The Compose suite verifies image/version alignment, environment coverage,
nonempty authentication values, literal database credential sharing, optional
reset keys, both profiles' default and overridden ports, health checks, storage,
script-mount protection, networks, restart policy, and API hardening. Tests write
only temporary synthetic environment files outside the checkout. Configuration
rendering is not evidence that a container has started or a port is available.

Docker is not installed in the local execution environment. The actual
PostgreSQL integration suite was attempted and stopped before running tests with
`FileNotFoundError: docker`. Container builds, runtime inventory/scanning, and
application smoke tests require the hosted workflows; they are not represented
as local passes. Hosted results are recorded below once available.

## Security and compatibility review

Missing JWT/bootstrap secrets no longer trigger built-in development credentials
through these Compose profiles. Empty example secrets enforce operator setup;
Compose rejects emptiness, while cryptographic quality remains the operator's
responsibility and production mode validates lengths. Proxy trust remains false
by default. Enabling it is appropriate only behind ingress that replaces client
forwarding headers. The optional reset key disables the HTTP soft-wipe route when
empty; an enabled route retains administrator-session authorization.

Database credentials still use `PGUSER`, `PGPASSWORD`, and `PGDATABASE`, avoiding
URL parsing of passwords. No SQL or migration code changed. No application
logging, input validation, authorization handler, or password-hashing behavior
changed. Existing structured logs, readiness/liveness probes, and runtime
hardening remain in place; there are no metrics/alert configuration changes.
A full Compose render exposes credentials, so operator examples use `--quiet`.
The example is tracked; real `.env` variants are ignored.

Govulncheck reports GO-2026-5932 only for the unused
`golang.org/x/crypto/openpgp` package at required-module level; no imported
package or reachable symbol is affected. Existing dependency files remain
unchanged, and container vulnerability policy remains mandatory without added
suppressions. Actual fresh container scans are a separate hosted gate.

The local profiles deliberately use plaintext database transport and default to
development. Production still requires TLS plus strong secrets. The official
PostgreSQL image creates its bootstrap role with superuser privileges on a new
cluster; that existing design is documented, not changed in this release.

No schema migration or backward-incompatible API/data change occurs. Required
configuration is tightened, and `.env.example` cannot be deployed unedited.
Existing environment overrides continue to win over new image defaults.
No core game logic, query, or I/O path changed, so load/performance and migration
rollback tests are not applicable. Prior source tag v01.06.08 resolves to the
base commit above; the retained deployment image digest remains the preferred
rollback artifact. Local Docker cannot verify old registry artifacts.

## Hosted validation

The user authorized GitHub publication on 2026-09-27 after the initial automatic
approval review blocked a push under the verification-only request. The local
Git transport lacks write credentials, so publication uses the connected GitHub
account and preserves the complete validated tree on `compose-settings-01.06.09`.
Hosted checks are pending publication of that branch; results will be recorded
once the workflows finish. No release image or source tag is yet published.

Do not infer image publication or deployment from local configuration validation.
The default 01.06.09 image is not yet published. Use a verified 01.06.08 image as
an explicit override if inspecting these compatible configuration changes before
release. Release tagging remains gated on successful CI, Build Validation,
Database Startup, and Publish GHCR Image at the same current main SHA.

## Rollback

Restore prior Compose/environment files and the verified 01.06.08 image without
removing volumes. Existing passwords and data are not rewritten by this patch.
If credentials were intentionally rotated while deploying, restore them through
the database reconciliation workflow before rolling back that configuration.
See [release notes](RELEASE_NOTES_01.06.09.md) for upgrade/rollback steps.
