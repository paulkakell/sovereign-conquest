# Sovereign Conquest Configuration

This guide describes the configuration contract for version 01.06.01.

## Deployment profiles

`docker-compose.yml` is a local-development profile. It uses local database transport and publishes development ports. Do not expose it directly to the public Internet.

The repository-root image is the preferred production artifact. It serves the API and bundled web client on port 8080. Production deployments must provide their own PostgreSQL service, TLS termination, secrets, backup policy, and persistent database storage.

## Required production settings

Set `APP_ENV=production` to enable startup validation.

| Setting | Purpose | Production requirement |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection | Must use `sslmode=require`, `verify-ca`, or `verify-full` |
| `JWT_SECRET` | Signs user sessions | At least 32 characters from a cryptographic random source |
| `INITIAL_ADMIN_USERNAME` | Bootstrap administrator name | Set to the intended operator account |
| `INITIAL_ADMIN_PASSWORD` | Initial bootstrap value | At least 16 characters and unique to this deployment |
| `ADMIN_SECRET` | Secondary authorization for season reset | Optional; at least 32 characters when enabled |
| `HTTP_ADDR` | API bind address | Defaults to `:8080` |
| `WEB_ROOT` | Static web directory | Root image defaults to `/app/web` |

The service refuses production startup when these requirements are not met.

## Proxy configuration

`TRUST_PROXY_HEADERS=false` is the safe default. The API removes incoming forwarding headers before they reach the router.

Set `TRUST_PROXY_HEADERS=true` only when every request reaches the API through a controlled reverse proxy that replaces `X-Real-IP` or `CF-Connecting-IP`. Do not enable it when clients can connect directly to the API port.

The split Nginx configuration replaces `X-Real-IP` and clears `X-Forwarded-For` before proxying API requests.

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

## Build settings

The committed vendor tree remains the default build source.

| Setting | Purpose |
|---|---|
| `SC_USE_VENDOR` | Set to `1` to require vendored dependencies |
| `GOPROXY` | Go module proxy chain for intentional networked builds |
| `GOSUMDB` | Go checksum database setting |
| `GOPRIVATE` | Private module patterns |
| `GONOSUMDB` | Checksum exclusions for private modules |
| `SC_BUILD_DNS` | Optional build-time resolver override |
| `SC_BUILD_NETWORK` | Compose build network mode |

Version 01.06.01 does not update the module graph because the repository connector could not safely regenerate `go.sum` and `vendor`. Run the dependency procedure in `docs/VALIDATION_01.06.01.md` from a trusted checkout before declaring dependency remediation complete.
