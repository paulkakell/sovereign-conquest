# Validation: 01.06.06

Base: `d5bf909aef936ae0f619c3d053b57f3d89c212da` (`v01.06.05`).
This record is maintained during implementation. Hosted results are tied to the
actual commit and image digest in GitHub Actions; an earlier pass does not certify
a later commit.

## Required gates

| Area | Validation |
| --- | --- |
| Unit and regression | Full Go suite, native healthprobe edge cases, existing auth/game/config/build tests |
| Concurrency | Full Go race suite |
| Static analysis | gofmt, Go vet, Gosec, CodeQL, JavaScript and shell syntax |
| Dependencies | go mod verify, Govulncheck, unchanged go.mod/go.sum/vendor comparison |
| Containers | Fresh builds for combined, API-only, and web-only images |
| Runtime security | Exported filesystem and static ELF checks; Trivy JSON policy including unfixed and reported medium CVEs |
| Integration | Fresh PostgreSQL, authentication/game smoke flow, Docker native health status for both API images |
| Configuration | Compose validation, version consistency, native probe custom bind address support |
| Release | Finalizer regression tests, exact-main workflow gates, preservation of previous tags |

## Local and hosted evidence

Local Go 1.27.1 was downloaded for this review. Local Docker and PostgreSQL are
unavailable; full container and database execution uses the hosted Build
Validation workflow. Local results are recorded below; hosted run links are
recorded on the pull request and Actions page. No container build or scan pass is implied by source inspection.

Completed local checks on the implementation:

- Go 1.27.1 full unit/regression and race suites passed. Native proxy and IPv6
  cases execute without skips.
- Go vet, Gosec v2.28.0, formatting, and module verification passed.
- Govulncheck v1.7.0 found zero reachable vulnerabilities and zero affected
  imported packages. Module-only GO-2026-5932 concerns the unused
  `golang.org/x/crypto/openpgp` package; it is not imported or vendored here.
- All 69 Node release safeguard tests and 13 Python container-policy tests passed.
- Browser JavaScript, workflow YAML, shell syntax, and whitespace checks passed.
- `CGO_ENABLED=0` vendored Linux build produced a statically linked ELF binary.
  Executable smoke checks passed for HTTP 200, HTTP 503, and a stopped listener
  with invalid database configuration and empty production secrets.
- Manifests, checksums, and vendor files match the base commit.

Executable smoke testing caught the existing `init()` configuration hook running
before the new healthcheck mode. Configuration validation was moved into normal
server startup, with regression coverage for both command paths. Normal
production validation remains required before connecting to PostgreSQL.

## Scope-specific review

No Go dependency version changes or lockfile regeneration. No database schema or
migration changes, so forward/backward migration tests are not applicable.
Existing authentication, authorization, logging, and input-validation tests
remain in the full suite. The new healthcheck has a fixed deadline and no body
processing; gameplay load tests are not required because game/query paths are
unchanged. Container startup and health behavior are covered by integration.

The security review covers package removal, TLS trust preservation, secrets,
non-root execution, and publication sequencing. External metrics and alerting
are not accessible from this repository task. Rollback uses the prior image
digest without database changes, as documented in the release notes; that prior
image retains the reported vulnerabilities.
