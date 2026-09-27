# Sovereign Conquest

Sovereign Conquest is a turn-based browser game built around an authoritative Go command engine, PostgreSQL state, and a single-page web client. This source version is **v01.06.09**.

## Current maintenance release

See the [01.06.09 release notes](docs/RELEASE_NOTES_01.06.09.md),
[validation record](docs/VALIDATION_01.06.09.md), and
[existing code review findings](docs/CODE_REVIEW_01.06.04.md). This maintenance
release verifies Compose settings, requires explicit secrets, pins the default
application image to the release, and documents every environment setting. See the [database startup guide](docs/DATABASE_STARTUP.md). The roadmap implementation packets remain planned. Use one API
replica until the documented scheduler issue is repaired.

## Community and support

- [Report a bug, request a feature, or correct documentation](https://github.com/paulkakell/sovereign-conquest/issues/new/choose).
- [Ask questions, discuss ideas, or share your work](https://github.com/paulkakell/sovereign-conquest/discussions).
- [Report a vulnerability privately](SECURITY.md).
- [Sponsorship configuration and community guidance](docs/COMMUNITY.md).

The issue chooser separates actionable reports from help and discussion. Existing
Discussions categories have forms for announcements, questions, ideas, general
conversation, and community projects. The funding recipient is `paulkakell`;
receiving sponsorships requires an active GitHub Sponsors profile.

## Roadmap

The [product and implementation roadmap](docs/ROADMAP.md) connects the game design to the evaluated codebase, with seven proposed releases and fifteen implementation packets. Start with its [implementation tracker](docs/ROADMAP.md#implementation-tracker) for dependencies, status, and acceptance criteria.

The first milestone addresses database migration coverage, port demand, repeatable XP, transaction failures, session revocation, reset consistency, and release validation. Later milestones add durable actions, structured controls, shared intelligence, freight contracts, onboarding, protected conflict, relay objectives, and competitive seasons. Planned features and version numbers are proposals; the roadmap records evidence when implementation is delivered.

## GHCR image

The repository publishes one combined API and web image:

```bash
docker pull ghcr.io/paulkakell/sovereign-conquest:01.06.09
docker pull ghcr.io/paulkakell/sovereign-conquest:main
```

`01.06.09` is the release image tag, available only after all release gates pass. `v01.06.08` remains the rollback source baseline. The `main` tag moves after a validated push to the default branch. For the strongest reproducibility guarantee, set `SC_IMAGE` to a verified image digest.

The publication workflow scans and smoke-tests the built image before moving public tags. It also emits a source-SHA tag and provenance attestation. See the [container security review](docs/SECURITY_REVIEW_01.06.06.md) for the reported CVEs and verification policy.

## Architecture

- Combined Go API and bundled web client image
- PostgreSQL authoritative state
- Transactional command engine
- Server-managed turn, port, planet, event, and Protectorate jobs
- Per-player discovery, market intelligence, planets, corporations, mines, seasons, messaging, and events

Every state-changing game action is validated on the server and applied through database transactions. Client-provided outcomes are never authoritative.

## Existing database password failures

Use the [01.06.08 recovery guide](docs/DB_RECOVERY_01.06.08.md) for the conquest
bind mount, external network, port 5000, and a one-time password repair. Install
the database scripts before using the new Compose profile. Do not use a database
reset to repair credentials. Root Compose retains two distinct host port options;
use the tailored profile when the API and web use one host port.

## Docker Compose deployment

`docker-compose.yml` pulls the combined image from GHCR. It does not build the API or web containers locally.

```bash
cp .env.example .env
chmod 600 .env
# Fill POSTGRES_PASSWORD, JWT_SECRET, and INITIAL_ADMIN_PASSWORD.
# Set ADMIN_SECRET too if you need the HTTP season-reset endpoint.
docker compose config --quiet
docker compose pull
docker compose up -d --remove-orphans
```

Use the released image tag only after publication. See the [configuration guide](docs/CONFIGURATION.md) for all settings, secret generation, proxy access, production TLS, and the bind-profile port override.

Open `http://localhost:3000` on the Docker host. The same combined application is also available on `http://localhost:8080`, preserving the prior direct API port. PostgreSQL remains on the internal Compose network.

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
docker build -t sovereign-conquest:01.06.09 .
```

The combined image serves the API and web UI on port 8080. Production mode rejects weak secrets and database connections without transport verification.

## Branch policy

`main` is the only long-lived branch. Feature, release, and automated dependency branches are reviewed, merged when appropriate, and removed after disposition. Release history remains in `CHANGELOG.md` and the versioned files under `docs/`.

## Health and observability

- `GET /api/livez`: process liveness and version
- `GET /api/readyz`: database readiness
- structured request logs include method, path, status, response size, and duration; client network addresses are not persisted

The runtime contains the static Go API, certificates, timezone data, and bundled web
assets. It has no shell or package manager. To check liveness manually:

```bash
docker compose exec api /app/sovereign-api healthcheck
docker compose logs --tail=100 api
```

The native probe follows `HTTP_ADDR`, bypasses HTTP proxies, rejects redirects,
and exits within three seconds. Custom runtime scripts that call `sh`, `wget`,
or `apk` must be replaced with host-side tooling. Application APIs and database
formats remain compatible.

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
python3 -m unittest discover -s scripts -p 'test_compose_config.py' -v
docker build --pull -t sovereign-conquest:validation .
docker build --pull -t sovereign-conquest-api:validation ./server
docker build --pull -t sovereign-conquest-web:validation ./web
```

Deployment details, rollback guidance, security findings, and test cases are under `docs/`.
