# Changelog




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
