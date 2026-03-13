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

Initial admin account (seeded)
- On startup, the server ensures an initial admin account exists.
- Defaults:
  - Username: admin
  - Password: ChangeMeNow!
- The seeded admin account requires a password change on first login.
- Configure via environment variables:
  - INITIAL_ADMIN_USERNAME
  - INITIAL_ADMIN_PASSWORD
- Bootstrap recovery behavior:
  - If *no* admin users exist and the configured initial admin username already exists as a non-admin (for example from an older dev database), the server will promote that user to admin and reset its password to INITIAL_ADMIN_PASSWORD so you can regain access.
- Admin "god mode": admin players have zero turn costs and can MOVE to any existing sector (teleport).

UI note
- The login header shows the build version immediately; once the API is reachable, the UI overwrites it with the live backend version from /api/healthz.
- The header includes a "Report A Bug" link which opens a separate bug report window. Bug reports (with optional attachments) are delivered to the Admin account via in-game direct messaging.
- The top bar includes a Messages bell which opens the Messages page (inbox/sent, reply, delete). Unread messages are shown as a badge count.

Build note
- This archive now includes `server/go.sum` and a committed `server/vendor/` tree. The default API image build auto-detects `vendor/modules.txt` and uses `-mod=vendor`, so `docker compose up --build` no longer needs live access to `proxy.golang.org` or `sum.golang.org` after the base image is available locally.
- If you intentionally want a networked module restore or need extra build workarounds, set build args via a local `.env` file (or Portainer stack variables):
  - Important: these values feed compose interpolation (`build.network` + `build.args`). Setting them under `services.api.environment` only affects the runtime container and will not affect the image build.
  - If your builder still has broken/restricted DNS during image builds (a common Portainer/remote-builder failure mode), set:
    - `SC_BUILD_NETWORK=host`
    This maps to compose `build.network` (equivalent to `docker build --network=host`).
    - If host networking is unavailable, you can override build-time DNS inside the build container:
      - `SC_BUILD_DNS="1.1.1.1 8.8.8.8"` (space-separated) or `SC_BUILD_DNS="1.1.1.1,8.8.8.8"` (comma-separated)
      - Note: some builder sandboxes mount `/etc/resolv.conf` read-only; in that case the build prints a warning and continues without applying `SC_BUILD_DNS`.
      - If `SC_BUILD_DNS` is unset and a networked module download/build fails with a DNS-looking error, the build auto-retries once using a public DNS fallback.
  - `GOPROXY=https://proxy.golang.org|direct` (recommended default when you are not using vendoring)
  - `GOPROXY=direct` (common when `proxy.golang.org` is blocked but GitHub is reachable)
  - `GOSUMDB=off` (common when `sum.golang.org` is blocked)
  - `SC_USE_VENDOR=1` is still supported, but vendoring is now auto-detected when `vendor/modules.txt` is present.
  - If module downloads fail with TLS/x509 errors (corporate proxy / private proxy CA), add your CA certificate(s) as `.crt` files under `./server/certs/` and rebuild.

CI/registry note
- The GitHub Actions publish workflow builds the real service images instead of the repository root:
  - `ghcr.io/<owner>/sovereign-conquest-api` from `./server/Dockerfile`
  - `ghcr.io/<owner>/sovereign-conquest-web` from `./web/Dockerfile`
- Local development remains `docker compose up --build`; the repository root intentionally does not use a combined Dockerfile.

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
- SHIPYARD (Protectorate sectors only)
  - SHIPYARD
  - SHIPYARD BUY {SCOUT|TRADER|FREIGHTER|INTERCEPTOR}
  - SHIPYARD SELL
  - SHIPYARD UPGRADE {CARGO|TURNS}
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
- Protectorate fighter counts fluctuate on a tick (PROTECTORATE_TICK_SECONDS). Set PROTECTORATE_TICK_SECONDS=0 to disable fluctuations.
