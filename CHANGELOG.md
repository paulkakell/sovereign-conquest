# Changelog


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