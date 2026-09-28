# Changelog

## 01.06.10 - 2026-09-27

### Fixes

- Validate all four secrets at API startup in every environment, before database
  access. Inspect the effective PostgreSQL password, require 32-byte signing/reset
  keys and a 16-72-byte bootstrap password, and retain the optional empty reset key.
- Report all invalid settings in one descriptive container-log entry without
  values or connection URLs. Exit 1 and remove built-in credential fallbacks.
- Correct registration/password-change validation to bcrypt's actual 8-72-byte
  range, returning HTTP 400 before hashing or database access instead of a 500.

### Additive

- Add boundary, Unicode, redaction, startup-process and actual-container tests.
  Exercise valid passwords and rejected changes against fresh PostgreSQL for
  both API images. Add guarded release finalization and password API docs.

### Compatibility

- Configuration tightening: weak development signing/bootstrap secrets now stop
  API startup. This is an intentional startup compatibility change. Existing
  compliant deployments, optional reset-key behavior and login hashes remain valid.
- No schema, dependency, gameplay, API shape or persistence-format changes.
  Existing overlength account-password requests now receive a descriptive 400.

Base: `1b1b03d0eb8bb72efd4d6079292798eb2bd718d3` (v01.06.09, PR #29).
Reference: user-requested startup validation and password-limit correction;
no separate issue. See [release notes](docs/RELEASE_NOTES_01.06.10.md),
[validation/security review](docs/VALIDATION_01.06.10.md), and
[commit notes](docs/COMMIT_NOTES_01.06.10.md).

## 01.06.09 - 2026-09-27

### Fixes

- Require nonempty JWT and bootstrap administrator secrets in root Compose so
  omitted settings cannot fall through to development credentials. Leave the
  three required example secrets blank to require operator input.
- Align both Compose image defaults and `.env.example` with the 01.06.09 release
  instead of silently following `main`. Verified digests remain supported.
- Make the root database script mount read-only with `create_host_path: false`.
- Pass the documented `TRUST_PROXY_HEADERS` setting through both Compose
  profiles, retaining `false` by default. Ignore local environment files in Git.
- Correct documentation for disabled season resets, bootstrap password limits,
  development/TLS requirements, both profile ports, database privileges, and
  actual scheduler bounds and creation-time universe behavior.

### Additive

- Exercise real Compose rendering with nine regression tests covering every
  example setting, missing/empty required values, literal credentials, overrides,
  storage, networks, health, and hardening; run them in CI/build/publication gates.
  Accept equivalent omitted/false JSON options across Compose versions while
  continuing to reject automatic creation of the script mount directory.
- Add guarded 01.06.09 release finalization, 20 release-safeguard regressions,
  full configuration documentation, validation, release notes, and rollback.

### Compatibility

- Configuration tightening: copied examples must be completed before Compose can
  render/start. Existing deployments with explicit secrets remain compatible.
  `ADMIN_SECRET` is optional; leaving it empty disables HTTP season resets.
- No API, schema, database script, dependency, gameplay, or data-format changes.
  Root ports remain 3000/8080; the tailored default remains 5000 when unset.
  Existing `.env` image overrides still take precedence and must be updated
  explicitly when adopting the release.

Base: `2afa856cef71dd6f5ea5060425baba43935828be` (v01.06.08, PR #28).
Release pull request: #29; requested Compose and environment verification, no separate issue.
Hosted CI, all 15 PostgreSQL regressions, container builds/scans, application smoke
tests, and CodeQL passed on `09c5c7220e306aabc5888d2eb1b6e1dd6c701775`.
See [configuration](docs/CONFIGURATION.md),
[release notes](docs/RELEASE_NOTES_01.06.09.md),
[validation](docs/VALIDATION_01.06.09.md), and
[copyable commit notes](docs/COMMIT_NOTES_01.06.09.md).

## 01.06.08 - 2026-09-27

### Fixes

- Combine the separately developed 01.06.07 database startup work with the published
  01.06.06 API security fixes. Check and repair credentials on existing volumes
  before marking PostgreSQL healthy, resolving the reported SQLSTATE 28P01 loop.
- Add a bind-mount deployment profile preserving `/dockershare/containers/conquest/db`
  and the supplied networks, with one port-5000 publication for the combined API/UI.
- Incorporate PR #27 hardening for legacy localhost trust and disabled bootstrap
  logins. Always reset the password during explicit live recovery so trust cannot
  hide a stored-password mismatch.
- Add one-time live password repair using environment-backed, quoted SQL and
  authenticated verification without printing secrets or changing HBA rules.

### Additive

- Add failure/redaction and real-PostgreSQL recovery regressions; require the
  database regression suite before publishing release image candidates.
- Document recovery, deployment, secret rotation, validation, and rollback.
  Align application/web metadata and gated release tagging at 01.06.08.

### Compatibility

- No breaking API, gameplay, schema, or dependency changes. Preserve the prior
  scratch API runtime, native probes, vulnerability gates, and existing rows.
- The tailored standalone profile uses WEB_PORT only; root Compose retains both
  existing port options. Database scripts must be installed with the new profile.

Base: `c3de81af4d471e7cf327e6f43df96ff321a6b04a` (v01.06.06, PR #26).
Release pull request: #28; incorporates startup hardening from #27.
GitHub validated all 15 database integration tests, CI, container builds/scans,
application smoke tests, and CodeQL on `981dba92de46a2ea40239987a0e75e653cca5cd7`.
Prior local implementation: `a44c249`, `9b0d400`; no separate issue opened.
See [release notes](docs/RELEASE_NOTES_01.06.08.md),
[validation](docs/VALIDATION_01.06.08.md),
[recovery](docs/DB_RECOVERY_01.06.08.md), and
[copyable commit notes](docs/COMMIT_NOTES_01.06.08.md).

## 01.06.07 - 2026-09-27

### Fixes

- Reconcile the configured PostgreSQL database and login on every container
  startup, including existing volumes, missing roles/databases, password changes,
  and disabled or expired logins. No databases or tables are deleted.
- Replace `pg_isready` with a password-authenticated query and block readiness
  until private startup reconciliation finishes.
- Pass Compose credentials through PGUSER, PGPASSWORD, and PGDATABASE so literal
  punctuation cannot corrupt the API connection URL.

### Additive

- Add PostgreSQL container regression tests, required database release gates,
  structured startup events, deployment examples, and credential rollback steps.

### Compatibility

- Patch release fixing deployment startup. No API, schema, dependency manifest,
  or gameplay changes. Existing role privileges and object ownership are retained.
- Compose deployments must include `docker/db/`. Empty credentials and template
  databases are rejected. Environment credentials become authoritative on restart.

Base: `d5bf909aef936ae0f619c3d053b57f3d89c212da` (v01.06.05, PR #24).
01.06.06 is reserved for the separate API security change already in progress.
Reference: user-requested startup repair; no separate issue was opened.
See [release notes](docs/RELEASE_NOTES_01.06.07.md),
[validation](docs/VALIDATION_01.06.07.md), and
[copyable commit notes](docs/COMMIT_NOTES_01.06.07.md).

## 01.06.06 - 2026-09-27

### Fixes

- Remove vulnerable zlib, wget, busybox, busybox-binsh, and ssl_client packages
  from both API runtime images by using static Go executables in scratch stages.
  Addresses CVE-2026-85091, CVE-2026-58469, CVE-2026-58470, CVE-2026-58471,
  CVE-2026-58472, and CVE-2025-60876 without advisory suppressions.
- Replace shell-based wget health checks with a native, bounded probe that
  respects HTTP_ADDR, disables proxies/redirects, and requires HTTP 200.
- Validate runtime contents, scan vulnerabilities including unfixed reports,
  and smoke-test before moving GHCR public tags. Retrieve the pushed manifest
  digest from structured registry metadata and verify the validated image identity.

### Additive

- Add healthcheck regression tests, runtime inventory and scan-policy gates,
  coverage for both API images, and protected 01.06.06 release finalization.
- Update README, configuration, release notes, security review, and rollback.

### Compatibility

- No API, gameplay, schema, dependency manifest, or persistent format changes.
  Runtime shell/package utilities are removed; custom operator scripts must use
  host-side tools or the native healthcheck. Non-root UID, certificates, timezone
  data, web assets, and Compose hardening remain supported.

Base: `d5bf909aef936ae0f619c3d053b57f3d89c212da` (v01.06.05, PR #24).
Release pull request: #25. References: the six public CVEs above; no separate issue was opened.
See [release notes](docs/RELEASE_NOTES_01.06.06.md),
[validation](docs/VALIDATION_01.06.06.md),
[security review](docs/SECURITY_REVIEW_01.06.06.md), and
[copyable commit notes](docs/COMMIT_NOTES_01.06.06.md).

## 01.06.05 - 2026-09-27

### Additive

- Add issue forms for bugs, feature requests, and documentation, with links to
  Q&A, Ideas, and private vulnerability reporting so reports reach the right channel.
- Add category-matched discussion forms, a support guide, and community examples.
- Configure the owner's GitHub Sponsors destination in `.github/FUNDING.yml`.
  Account enrollment and payment acceptance are separate prerequisites.
- Add `SECURITY.md` with supported-version, private-reporting, and disclosure
  guidance. Existing Discussions and private reporting were verified enabled.

### Fixes and maintenance

- Close missing repository community-configuration gaps and align release
  metadata at 01.06.05. No breaking gameplay, API, schema, or configuration change.
- Add 20 release-publication safeguard tests and require all release tests in CI.
  Finalization creates the new tag and release only after exact-commit CI, build,
  and publication gates pass; previous tags remain unchanged.
- Preserve dependency manifests and vendored sources; no dependency updates.

Base: `d3a8e23c9e7975895d55d21cb8d513821cacc9d9` (v01.06.04, PR #23).
Release pull request: #24. No separate issue was opened for this repository setup.
See [release notes](docs/RELEASE_NOTES_01.06.05.md),
[validation](docs/VALIDATION_01.06.05.md),
[community guide](docs/COMMUNITY.md), and
[copyable commit notes](docs/COMMIT_NOTES_01.06.05.md).

## 01.06.04 - 2026-09-27

### Additive

- Integrate roadmap revision 00.02.01 from PR #22 without marking planned gameplay
  complete. Add a source-backed code review and safe, bounded release finalization.

### Fixes and maintenance

- Consolidate chi 5.3.2 (#14), pgx 5.11.0 (#21), x/crypto 0.57.0 (#20), synchronized
  vendoring, both Go 1.27.1 builders (#18, #19), Buildx v4 (#11), attestation v4
  (#13), and Trivy v0.36.0 (#12).
- Repair shared module conflicts and align compiler, action, UI, and smoke-test
  version contracts so the combined proposals can pass the complete release gates.
- Increment application version 01.06.03 to 01.06.04. Source builds require Go 1.26
  or later; no API, schema, or persistent data-format changes are introduced.

See [release notes](docs/RELEASE_NOTES_01.06.04.md),
[validation](docs/VALIDATION_01.06.04.md),
[review findings](docs/CODE_REVIEW_01.06.04.md), and
[copyable commit notes](docs/COMMIT_NOTES_01.06.04.md).


## Roadmap 00.02.01 (2026-09-27)

Documentation
- Add the complete [product and implementation roadmap](docs/ROADMAP.md), including the accepted game design, source assessment, implementation packets, acceptance criteria, migration and rollback requirements, and original planning brief.
- Add a linked implementation tracker for all fifteen packets across seven proposed releases; implementation status begins at Planned.
- Link the roadmap from the README and distinguish historical audit evidence from future release gates.
- Keep the roadmap document version separate from application version `01.06.03`; this integration changes no runtime code, database schema, dependencies, or deployment configuration.

## 01.06.03 (2026-08-13)

Fix
- Replace Docker Compose build definitions with the combined GHCR image `ghcr.io/paulkakell/sovereign-conquest:main`.
- Remove the redundant standalone `web` service while preserving host ports 3000 and 8080 through the combined `api` service.
- Add `SC_IMAGE` and `SC_PULL_POLICY` overrides for immutable tags, digests, and controlled refresh behavior.
- Align application metadata and active web badges at 01.06.03.

Security
- Retain loopback-only application bindings, internal-only PostgreSQL, read-only application storage, dropped capabilities, and `no-new-privileges`.
- Document digest pinning for controlled deployments.

Maintenance
- Remove obsolete Compose build variables from `.env.example`.
- Add regression tests for the pull-based Compose contract.
- Update README, configuration, release notes, security review, validation record, and commit notes.

Breaking deployment behavior
- `docker compose build` is no longer part of the deployment workflow.
- The standalone `web` service is removed; operators should use the `api` service.

Dependencies
- No dependency changes.

Database
- No schema or data-format changes.

Refs
- SC-DEPLOY-003, SC-CI-012

## 01.06.02 (2026-08-13)

Additive
- Add a branch-scoped development release pipeline with immutable and moving GHCR development tags.
- Add CodeQL, provenance attestation, SPDX SBOM generation, anonymous package-pull verification, and runtime smoke tests.
- Add deployment, validation, security-review, release-note, and commit-note documentation.

Fix
- Align application, browser, test, and documentation version metadata at 01.06.02.
- Correct the standalone bug-report page paths and version.
- Move compiled images and security validation to Go 1.26.6.

Security
- Upgrade chi to 5.3.1, jwt/v5 to 5.3.1, pgx/v5 to 5.10.0, and x/crypto to 0.55.0.
- Regenerate and verify `go.sum` and `server/vendor`.
- Require clean reachable-code and container vulnerability gates before development publication.

Compatibility
- Additive for the development channel; no stable tag or `main` publication occurs.

Known limitations
- Season-reset atomicity and durable audit insertion remain planned follow-up work.
- Startup DDL remains transitional pending numbered migrations.

Refs
- SC-REL-012, SC-SEC-005, SC-DEP-003
- Dependency commit: 6fe80acb730006ba834bc322b192ebd776933239
- Audit base: 7cec9eaf47e835e6555fdd3f8284c7b73b0c20c3

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
- Root and API Docker build stages use Go 1.26.5 and Alpine 3.24.
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



## 01.06.00 (2026-03-13)

Additive
- Add a repository-root Dockerfile that builds an all-in-one image for GHCR and generic `docker build .` workflows. The container now serves both the API and the bundled web UI on port 8080.
- Add `.github/workflows/docker-image.yml` to publish `ghcr.io/<owner>/sovereign-conquest` from the repository root with Buildx cache and provenance attestation.
- Server can optionally serve the bundled web UI when `WEB_ROOT` is set.
- API now exposes `/api/version` and `/api/help` for direct UI compatibility in single-container deployments.

Fix
- CI no longer fails with `failed to read dockerfile: open Dockerfile: no such file or directory` when using a repository-root Docker publish workflow.
- Single-image deployments no longer depend on a separate nginx container to serve the UI.
- Fix pre-existing compile issues in the admin map and mine deployment paths so the full Go source tree compiles cleanly again.

Maintenance
- Add regression tests for the root Dockerfile, the root GHCR workflow, and optional static web serving through the Go server.
- Update docs and environment examples for root-image deployment.

Refs
- SC-CI-010, SC-DEPLOY-002 | Base commit: 2bd576e



## 01.05.09 (2026-03-13)

Fix
- GitHub Actions container publishing: replace the single root-image workflow with a matrix build that publishes the two real service images (`./server/Dockerfile` and `./web/Dockerfile`) so CI no longer fails on `open Dockerfile: no such file or directory`.
- Cache isolation: scope the Buildx cache per service image so API and web layer caches do not collide in registry builds.
- Version metadata: realign `server/internal/config.Version` with the root `VERSION` file.

Maintenance
- Build regression tests: assert the publish workflow targets `./server` and `./web` instead of the repository root.
- Documentation: clarify the published GHCR image names and explain that local development still uses `docker compose up --build`.

Refs
- SC-BUILD-011 | Root-cause commit observed in CI logs: f296352


## 01.05.08 (2026-03-13)

Fix
- Docker/container build: commit `server/go.sum` and a fully populated `server/vendor/` tree so the API image can build from pinned dependencies without reaching `proxy.golang.org` or `sum.golang.org` during the application dependency step.
- Build reproducibility: run `go mod tidy` so the module manifest matches the actual compile graph and vendored/non-vendored builds resolve the same dependency set.
- Version metadata: realign `server/internal/config.Version` with the root `VERSION` file so the shipped version is internally consistent again.

Maintenance
- Build regression tests: require `server/go.sum` and `server/vendor/modules.txt` to stay committed and verify `.dockerignore` does not exclude vendored dependencies.
- Documentation: update `README.md` and `.env.example` to make vendoring the default/offline-safe build path while retaining the older DNS/proxy overrides as fallback guidance.

Refs
- SC-BUILD-010 | Base commit: 2bd576e

## 01.05.07 (2026-02-24)

Fix
- Docker build: make SC_BUILD_DNS override best-effort (do not fail the build if /etc/resolv.conf is not writable in the builder sandbox).
- Docker build: on build script failure, emit a compact tail of the captured build log at the end of the Docker RUN step so Portainer/BuildKit error summaries are more likely to include the actionable Go error output.

Maintenance
- Build tests: assert Dockerfile captures build_api.sh output and prints a tail on failure.

Refs
- SC-BUILD-009 | Commit: N/A (no git metadata in provided artifact)

## 01.05.06 (2026-02-24)

Fix
- Docker build: move module download/build logic into `server/scripts/build_api.sh` so Portainer/remote builder UIs are less likely to truncate away the actionable Go error output.
- Docker build: sanitize `SC_BUILD_DNS` values (accept comma-separated lists and tolerate surrounding quotes) and auto-retry module download/build once with a public DNS fallback when failures look DNS-related and `SC_BUILD_DNS` is unset.

Security
- Build logs: redact basic-auth credentials in GOPROXY values when printing module settings.

Maintenance
- Build tests: assert the Dockerfile delegates to the build script and that the script retains the expected restricted-network build features.

Refs
- SC-BUILD-008 | Commit: N/A (no git metadata in provided artifact)

## 01.05.05 (2026-02-24)

Fix
- Docker build: run `go mod download` after copying the full server source so the Go tool can resolve the full module graph (avoids `go build` triggering new downloads unexpectedly in restricted builders).
- Docker build: add `SC_BUILD_DNS` build arg to optionally override /etc/resolv.conf inside the build stage (workaround for builder sandboxes that inject a broken DNS server).
- Docker build: auto-detect vendoring when `vendor/modules.txt` is present, so vendored builds work even if build args can't be propagated by the stack deploy UI.
- Docker build: automatically retry module download/build with `GOSUMDB=off` when failure is checksum-db related (sum.golang.org blocked).

Maintenance
- Compose: pass `SC_BUILD_DNS` through to the API image build.
- Docs: document `SC_BUILD_DNS` and clarify vendoring auto-detection.

Refs
- SC-BUILD-007 | Commit: N/A (no git metadata in provided artifact)

## 01.05.04 (2026-02-24)

Fix
- Compose build: add `SC_BUILD_NETWORK` to control the build-time network mode (`build.network`) for environments where module downloads fail due to broken/restricted DNS in the build sandbox (common in some Portainer/remote builder setups).
- Docker build: print actionable diagnostics and remediation hints when `go mod download` or `go build` fails (network/DNS, GOPROXY/GOSUMDB, custom CAs, vendoring).

Maintenance
- Docs: document `SC_BUILD_NETWORK` in README and `.env.example`.
- Build tests: assert compose supports `SC_BUILD_NETWORK`.

Refs
- SC-BUILD-006 | Commit: N/A (no git metadata in provided artifact)

## 01.05.03 (2026-02-24)

Fix
- Docker build: trust additional CA certificates placed under `server/certs/` during both build and runtime (helps corporate proxies / private module proxies).
- Compose build: pass standard proxy args (HTTP_PROXY/HTTPS_PROXY/NO_PROXY + lowercase variants) into the API image build.

Maintenance
- Docs: add troubleshooting note for TLS/x509 failures and the `server/certs/` workflow.

Refs
- SC-BUILD-005 | Commit: N/A (no git metadata in provided artifact)


## 01.05.02 (2026-02-24)

Fix
- Docker/compose build: default GOPROXY now uses the pipe form (`https://proxy.golang.org|direct`) so Go can fall back to direct VCS fetches when the proxy is unreachable (the previous comma form only falls back on 404/410).

Maintenance
- Build tests: assert Dockerfile + compose defaults include the proxy fallback behavior.
- Docs: clarify recommended GOPROXY settings for restricted build environments.

Refs
- SC-BUILD-004 | Commit: N/A (no git metadata in provided artifact)


## 01.05.01 (2026-02-23)

Fix
- Docker/compose build: allow overriding Go module download settings (GOPROXY/GOSUMDB/GOPRIVATE/GONOSUMDB) via compose build args to support restricted networks.
- Docker build: add an opt-in vendored build mode (SC_USE_VENDOR=1) that skips `go mod download` and uses `-mod=vendor` for fully-offline image builds.

Maintenance
- Docs: expand build troubleshooting in README and .env.example.

Refs
- SC-BUILD-003 | Commit: N/A (no git metadata in provided artifact)


## 01.05.00 (2026-02-23)

Additive
- Progression: introduce XP, levels, and rank titles; all successful commands grant XP and rank-ups are announced in the activity log.
- Galactic Protectorate: mark ~10% of sectors as Protectorate space with fluctuating fighter patrol counts, a major port (all resources), and shipyards (buy/sell/upgrade).
- Admin: add an ANSI/ASCII universe map (sectors, ports, planets, ownership, player locations) accessible via an admin-only UI link.
- Shipyard: add SHIPYARD command for purchasing ships and upgrading cargo/turns capacity (Protectorate sectors only).

Fix
- Messaging UI: move messaging to a dedicated Messages page; add topbar notification bell + unread count badge; add reply + per-user delete actions.
- Messaging backend: add read tracking, unread count endpoint, and per-user soft delete for inbox/sent views and attachment access.

Maintenance
- Schema: add player progression and ship fields to players; add Protectorate fields to sectors; add message metadata (read/deleted flags) to direct_messages.

Refs
- SC-MSG-003, SC-UI-004, SC-RANK-001, SC-PROT-001, SC-SHIP-001, SC-ADMIN-003 | Commit: N/A (no git metadata in provided artifact)

## 01.04.00 (2026-02-23)

Additive
- Messaging: add direct in-game messaging between users (send by username) with an inbox panel in the UI.
- Abuse handling: add a per-message "Report spam/abuse" action that forwards the reported message (and any attachments) to the Admin account via in-game messaging.
- Bug reporting: add a "Report A Bug" link next to the version badge that opens a dedicated bug report window with optional file attachments; submissions are delivered to the Admin account via in-game messaging.

Fix
- UI/gameplay: command failures now display the server-provided explanation (for example "No warp to that sector") instead of generic error codes.

Maintenance
- Schema: add `direct_messages` and `direct_message_attachments` tables plus attachment download endpoint.

Refs
- SC-UI-003, SC-MSG-001, SC-MSG-002 | Commit: N/A (no git metadata in provided artifact)

## 01.03.02 (2026-02-23)

Fix
- Auth/state: fix PostgreSQL error on login/register/state load by locking only the players row (`FOR UPDATE OF p`) when loading a player (the query includes LEFT JOINs).
- UI: add cache-busting query params for app.js/style.css and use the build version as the badge fallback so the login page shows a version even if /api/healthz is temporarily unreachable.
- API: prevent caching of /api/healthz (Cache-Control: no-store) so the UI always sees the current backend version.

Refs
- SC-DB-003, SC-UI-002 | Commit: N/A (no git metadata in provided artifact)

## 01.03.01 (2026-02-23)

Fix
- UI: version badge now retries /api/healthz on load (with backoff) so it populates once the API becomes reachable.
- Auth/bootstrap: if no admin users exist and the configured initial admin username already exists as a non-admin, the server promotes that user to admin and resets the password to the configured INITIAL_ADMIN_PASSWORD (password change still required on first login).

Refs
- SC-UI-001, SC-ADMIN-002 | Commit: N/A (no git metadata in provided artifact)

## 01.03.00 (2026-02-23)

Additive
- Auth/bootstrap: seed an initial admin account (configurable username/password) on server startup.
- Admin "god mode": admin players have zero turn costs and can MOVE to any existing sector (teleport).
- Auth: add a password change endpoint and UI flow; the seeded admin account requires password change on first login.
- UI: replace "Phase 3" header badge with a live version badge sourced from /api/healthz.

Fix
- Registration: choose a valid starting sector and ensure an active season exists, preventing "db error" on account creation in edge-case databases.

Refs
- SC-ADMIN-001 | Commit: N/A (no git metadata in provided artifact)

## 01.02.02 (2026-02-23)

Fix
- Docker build: remove invalid `-mod=mod` flag from `go mod download` step (this flag is only supported by build/test commands, not `go mod download`).

Maintenance
- Add a lightweight unit test to keep `server/internal/config.Version` in sync with the root VERSION file.

Refs
- SC-BUILD-002 | Commit: N/A (no git metadata in provided artifact)

## 01.02.01 (2026-02-23)

Fix
- Docker build: make the Go build stage more deployment-friendly by ensuring the output directory exists, allowing module sums to be generated during build, and disabling VCS stamping for artifact-only builds.

Refs
- SC-BUILD-001 | Commit: N/A (no git metadata in provided artifact)

## 01.02.00 (2026-02-23)

Additive
- Phase 3: market intel captured on SCAN (player-only port snapshots) plus MARKET analytics command.
- Phase 3: ROUTE command suggests best trade route using scanned intel only (freshness-weighted).
- Phase 3: scheduled event system (ANOMALY, INVASION, LIMITED) with sector overlays, UI display, and invasion penalties on entry.
- UI: quick buttons for Market/Route/Events and improved mobile layout.

Maintenance
- Schema: add events + player_sector_intel tables; soft wipe clears both.

Refs
- SC-PH3 | Commit: N/A (no git metadata in provided artifact)

## 01.01.00 (2026-02-23)

Additive
- Rebrand: rename project and UI to "Sovereign Conquest".
- Phase 2: planets (colonize, load/unload storage, citadel upgrade) with production ticker.
- Phase 2: corporations (create/join/leave), corp chat, and corp bank (deposit/withdraw).
- Phase 2: mines (deploy/sweep) and automatic mine strikes on sector entry.
- Phase 2: seasons + rankings commands.
- Admin: optional soft wipe endpoint to start a new season and reset player progression.

Fix
- Health endpoint now returns name/version metadata.

Breaking / behavior changes
- Local dev defaults changed (database name/user/password, localStorage token key).

## 01.00.00

- Initial Phase 1 loop: register/login, scan, move, trade buy/sell, turn regeneration, port regeneration tick, activity log.
