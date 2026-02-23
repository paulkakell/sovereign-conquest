# Changelog

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
