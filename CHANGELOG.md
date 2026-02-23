# Changelog

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
