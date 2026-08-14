# Sovereign Conquest Configuration

This guide describes the configuration contract for version 01.06.03.

## Deployment profiles

`docker-compose.yml` is a local-development profile. It pulls the combined API and web image from GHCR and does not contain Docker build definitions. PostgreSQL remains a separate service on the internal Compose network.

The default image is `ghcr.io/paulkakell/sovereign-conquest:main`. The `main` tag is moving. Pin `SC_IMAGE` to `ghcr.io/paulkakell/sovereign-conquest:01.06.03` or a verified digest for controlled deployments.

The Compose profile uses local database transport and publishes loopback-only development ports. Do not expose it directly to the public Internet.

## Published image tags

The active GHCR workflow publishes:

| Tag | Purpose |
|---|---|
| `main` | Moving image for the current default branch |
| `01.06.03` | Release image for this version |
| `sha-<source-sha>` | Source-specific traceability |

The version image is attested, scanned for high and critical vulnerabilities, and smoke-tested against a fresh PostgreSQL database before publication completes.

## Compose image settings

| Setting | Default | Purpose |
|---|---|---|
| `SC_IMAGE` | `ghcr.io/paulkakell/sovereign-conquest:main` | Combined API and web image |
| `SC_PULL_POLICY` | `always` | Compose image refresh policy |
| `WEB_PORT` | `3000` | Browser-facing host port |
| `API_PORT` | `8080` | Compatibility host port for direct API access |

Both host ports reach port 8080 in the same combined container. The API remains available below `/api` on either port.

Update a Compose deployment with:

```bash
docker compose pull
docker compose up -d --remove-orphans
```

The old standalone `web` service no longer exists. Operational commands should target the `api` service.

## Required production settings

Set `APP_ENV=production` to enable startup validation. The repository Compose profile deliberately defaults to `development` because its internal PostgreSQL connection uses `sslmode=disable`.

| Setting | Purpose | Production requirement |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection | Must use `sslmode=require`, `verify-ca`, or `verify-full` |
| `JWT_SECRET` | Signs user sessions | At least 32 characters from a cryptographic random source |
| `INITIAL_ADMIN_USERNAME` | Bootstrap administrator name | Set to the intended operator account |
| `INITIAL_ADMIN_PASSWORD` | Initial bootstrap value | At least 16 characters and unique to this deployment |
| `ADMIN_SECRET` | Secondary authorization for season reset | Optional; at least 32 characters when enabled |
| `HTTP_ADDR` | API bind address | Defaults to `:8080` |
| `WEB_ROOT` | Static web directory | Combined image uses `/app/web` |

The service refuses production startup when these requirements are not met.

## Proxy configuration

`TRUST_PROXY_HEADERS=false` is the safe default. The API removes incoming forwarding headers before they reach the router.

Set `TRUST_PROXY_HEADERS=true` only when every request reaches the API through a controlled reverse proxy that replaces `X-Real-IP` or `CF-Connecting-IP`. Do not enable it when clients can connect directly to the API port.

## Game and scheduler settings

| Setting | Default | Behavior |
|---|---:|---|
| `UNIVERSE_SEED` | `2002` | Reproducible universe generation seed |
| `UNIVERSE_SECTORS` | `200` | Number of canonical sectors |
| `TURN_REGEN_SECONDS` | `120` | Seconds required to regenerate one turn |
| `PORT_TICK_SECONDS` | `60` | Port regeneration interval; `0` disables it |
| `PLANET_TICK_SECONDS` | `60` | Planet production interval; `0` disables it |
| `EVENT_TICK_SECONDS` | `60` | Event-generation interval; `0` disables it |
| `PROTECTORATE_TICK_SECONDS` | `60` | Protectorate update interval; `0` disables it |

Port, planet, and event jobs use PostgreSQL advisory locks to prevent duplicate execution when multiple API replicas are running.

## Health endpoints

- `GET /api/livez` confirms the process can serve HTTP.
- `GET /api/readyz` confirms the process can reach PostgreSQL.
- `GET /api/version` returns the current application version.

Use `/api/readyz` for load-balancer readiness. Use `/api/livez` for container liveness.

## Manual source-build settings

Compose no longer consumes build arguments. Manual `docker build` operations continue to support the committed vendor tree and these settings:

| Setting | Purpose |
|---|---|
| `SC_USE_VENDOR` | Set to `1` to require vendored dependencies |
| `GOPROXY` | Go module proxy chain for intentional networked builds |
| `GOSUMDB` | Go checksum database setting |
| `GOPRIVATE` | Private module patterns |
| `GONOSUMDB` | Checksum exclusions for private modules |
| `SC_BUILD_DNS` | Optional build-time resolver override |

## Rollback

Restore the prior Compose file and set `SC_IMAGE` to the previously verified image digest. The Compose change does not alter the PostgreSQL schema or stored game data, so database rollback is not required for this deployment change.
