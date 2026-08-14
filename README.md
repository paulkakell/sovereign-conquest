# Sovereign Conquest

Sovereign Conquest is a turn-based browser game built around an authoritative Go command engine, PostgreSQL state, and a single-page web client. The current release is **v01.06.03**.

## GHCR image

The repository publishes one combined API and web image:

```bash
docker pull ghcr.io/paulkakell/sovereign-conquest:01.06.03
docker pull ghcr.io/paulkakell/sovereign-conquest:main
```

`01.06.03` is the release tag for this version. The `main` tag moves after a validated push to the default branch. For the strongest reproducibility guarantee, set `SC_IMAGE` to a verified image digest.

The publication workflow also emits a source-SHA tag, provenance attestation, high and critical vulnerability scan, and a runtime smoke test of the pushed image.

## Architecture

- Combined Go API and bundled web client image
- PostgreSQL authoritative state
- Transactional command engine
- Server-managed turn, port, planet, event, and Protectorate jobs
- Per-player discovery, market intelligence, planets, corporations, mines, seasons, messaging, and events

Every state-changing game action is validated on the server and applied through database transactions. Client-provided outcomes are never authoritative.

## Docker Compose deployment

`docker-compose.yml` pulls the combined image from GHCR. It does not build the API or web containers locally.

```bash
cp .env.example .env
# Replace every placeholder secret in .env.
docker compose pull
docker compose up -d --remove-orphans
```

Open `http://localhost:3000`. The same combined application is also available on `http://localhost:8080`, preserving the prior direct API port. PostgreSQL remains on the internal Compose network.

Override the image without editing Compose:

```bash
SC_IMAGE=ghcr.io/paulkakell/sovereign-conquest@sha256:<digest> docker compose up -d
```

Update an existing deployment:

```bash
docker compose pull
docker compose up -d --remove-orphans
```

Reset a development universe:

```bash
docker compose down -v
docker compose pull
docker compose up -d --remove-orphans
```

The former standalone `web` service has been removed. Use `docker compose logs api` for the combined service. `docker compose build` is no longer part of the Compose deployment path.

## Manual source build

A local source build remains available outside Compose:

```bash
docker build -t sovereign-conquest:01.06.03 .
```

The combined image serves the API and web UI on port 8080. Production mode rejects weak secrets and database connections without transport verification.

## Branch policy

`main` is the only long-lived branch. Feature, release, and automated dependency branches are reviewed, merged when appropriate, and removed after disposition. Release history remains in `CHANGELOG.md` and the versioned files under `docs/`.

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
