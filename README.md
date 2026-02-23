Sovereign Conquest (Phase 3)

What is included
- Go API server (turn-based command engine)
- Postgres (authoritative game state)
- Nginx static web UI (single-page interface; proxies /api to the Go server)
- Phase 1 loop: register/login, scan, move, trade buy/sell, turn regeneration, port regeneration tick, activity log
- Phase 2 systems: planets + production ticker, corporations + corp bank + corp chat, mines + sweep + mine strikes, seasons + rankings, admin soft-wipe endpoint
- Phase 3 systems: player-only market intel (SCAN snapshots), market analytics + route suggestion, scheduled events (anomalies/invasions/limited-time sectors), mobile-friendly UI upgrades

Quick start
1) Unzip the archive.
2) From the project root, run:
   docker compose up --build
3) Open:
   http://localhost:3000

Build note
- The API container build downloads Go modules during image build. If your build environment restricts outbound access to the default Go module proxy, set GOPROXY accordingly (or vendor dependencies).

Default ports
- Web UI:  http://localhost:3000
- API:     http://localhost:8080
- Postgres: localhost:5432 (user sovereign, password sovereign, db sovereign_conquest)

Core commands
- SCAN
- MOVE {to}
- TRADE {BUY|SELL} {ORE|ORGANICS|EQUIPMENT} {qty}

Phase 2 commands
- PLANET
  - PLANET INFO
  - PLANET COLONIZE [name...]
  - PLANET LOAD {ORE|ORGANICS|EQUIPMENT} {qty}
  - PLANET UNLOAD {ORE|ORGANICS|EQUIPMENT} {qty}
  - PLANET UPGRADE CITADEL
- CORP
  - CORP INFO
  - CORP CREATE {name...}
  - CORP JOIN {name...}
  - CORP LEAVE
  - CORP SAY {message...}
  - CORP DEPOSIT {credits}
  - CORP WITHDRAW {credits}
- MINE
  - MINE DEPLOY {qty}      (consumes ship equipment cargo)
  - MINE SWEEP             (removes hostile mines in the sector)
- RANKINGS
- SEASON

Phase 3 commands
- MARKET [ORE|ORGANICS|EQUIPMENT]
  - Uses only your scanned intel (SCAN) to avoid omniscient pricing.
- ROUTE [ORE|ORGANICS|EQUIPMENT]
  - Suggests a trade route using scanned intel only (freshness-weighted).
- EVENTS
  - Lists active events in sectors you have discovered.

Admin: soft wipe (new season)
- Set ADMIN_SECRET in docker-compose.yml (or .env) to enable admin endpoints.
- POST /api/admin/soft_wipe with header:
  X-Admin-Secret: <ADMIN_SECRET>
- Body (optional):
  {"season_name":"Season X","reset_corps":false}

Resetting the universe (local dev)
- Stop containers, then remove the database volume:
  docker compose down -v
- Start again:
  docker compose up --build

Notes
- All state-changing actions go through a single transactional command endpoint.
- Turns regenerate on demand (each command call recalculates turns since last regen).
- Ports and planets regenerate on server ticks to keep the economy and production moving even when nobody is online.
- Events are generated/expired on an event tick (EVENT_TICK_SECONDS). Set EVENT_TICK_SECONDS=0 to disable event generation.
