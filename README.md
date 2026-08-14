# Sovereign Conquest

Sovereign Conquest is a turn-based browser game built around an authoritative Go command engine, PostgreSQL state, and a single-page web client. The current development candidate is **v01.06.02**.

## Development image

```bash
docker pull ghcr.io/paulkakell/sovereign-conquest:dev-01.06.02
docker pull ghcr.io/paulkakell/sovereign-conquest:dev
```

`dev-01.06.02` is immutable. `dev` moves to the newest validated development candidate. Neither tag is the stable channel.

Development publication requires unit and regression tests, race detection, `go vet`, `govulncheck`, container scanning, an SPDX SBOM, provenance attestation, a local runtime smoke test, an anonymous GHCR pull, and a second smoke test of the publicly pulled image.

## Architecture

- Go API and transactional command engine
- PostgreSQL authoritative state
- Nginx split deployment or an all-in-one image served by Go
- Server-managed turn, port, planet, event, and Protectorate jobs
- Per-player discovery, market intelligence, planets, corporations, mines, seasons, messaging, and events

Every state-changing game action is validated on the server and applied through database transactions. Client-provided outcomes are never authoritative.

## Local split deployment

```bash
cp .env.example .env
# Replace every placeholder secret in .env.
docker compose up --build
```

Open `http://localhost:3000`. The API is bound to `127.0.0.1:8080`. PostgreSQL remains on the internal Compose network.

To reset a development universe:

```bash
docker compose down -v
docker compose up --build
```

## All-in-one local build

```bash
docker build -t sovereign-conquest:01.06.02 .
```

The combined image serves the API and web UI on port 8080. Production mode rejects weak secrets and database connections without transport verification.

## Health and observability

- `GET /api/livez`: process liveness and version
- `GET /api/readyz`: database readiness
- structured request logs include method, path, status, response size, and duration; client network addresses are not persisted

## Commands

```text
SCAN
MOVE <sector>
TRADE <BUY|SELL> <ORE|ORGANICS|EQUIPMENT> <quantity>
PLANET INFO
PLANET COLONIZE [name]
PLANET LOAD <commodity> <quantity>
PLANET UNLOAD <commodity> <quantity>
PLANET UPGRADE CITADEL
CORP INFO
CORP CREATE <name>
CORP JOIN <name>
CORP LEAVE
CORP SAY <message>
CORP DEPOSIT <credits>
CORP WITHDRAW <credits>
MINE DEPLOY <quantity>
MINE SWEEP
SHIPYARD
SHIPYARD BUY <SCOUT|TRADER|FREIGHTER|INTERCEPTOR>
SHIPYARD SELL
SHIPYARD UPGRADE <CARGO|TURNS>
MARKET [commodity]
ROUTE [commodity]
EVENTS
RANKINGS
SEASON
```

## Administrator season reset

`/api/admin/soft_wipe` is enabled only when `ADMIN_SECRET` is configured. It requires an authenticated administrator bearer token and the separate `X-Admin-Secret` value.

## Validation

```bash
cd server
go mod verify
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
go install golang.org/x/vuln/cmd/govulncheck@v1.7.0
"$(go env GOPATH)/bin/govulncheck" ./...
cd ..
cp .env.example .env
docker compose config --quiet
docker build --pull -t sovereign-conquest:validation .
docker build --pull -t sovereign-conquest-api:validation ./server
docker build --pull -t sovereign-conquest-web:validation ./web
```

Deployment details, rollback guidance, security findings, and test cases are under `docs/`.
