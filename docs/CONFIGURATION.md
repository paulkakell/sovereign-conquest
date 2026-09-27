# Sovereign Conquest Configuration

This guide describes the configuration contract for version 01.06.07.

## Deployment profiles

`docker-compose.yml` is a local-development profile. It pulls the combined API and web image from GHCR and does not contain Docker build definitions. PostgreSQL remains a separate service on the internal Compose network.

The default image is `ghcr.io/paulkakell/sovereign-conquest:main`. The `main` tag is moving. Pin `SC_IMAGE` to `ghcr.io/paulkakell/sovereign-conquest:01.06.07` or a verified digest for controlled deployments.

The Compose profile uses local database transport and publishes loopback-only development ports. Do not expose it directly to the public Internet.

## Published image tags

The active GHCR workflow publishes:

| Tag | Purpose |
|---|---|
| `main` | Moving image for the current default branch |
| `01.06.07` | Release image for this version |
| `sha-<source-sha>` | Source-specific traceability |

The candidate image is scanned and smoke-tested against fresh PostgreSQL before public tags move. The same validated image is pushed and attested. Scans include unfixed vulnerabilities and explicitly reject every CVE listed in the 01.06.06 security review, regardless of severity.

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

## Database startup settings

`POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DB` are required by Compose
and checked on every `db` startup, including existing volumes. Keep `docker/db/`
beside the Compose file. Missing databases/logins and mismatched passwords are
repaired before the database becomes healthy. Existing rows and ownership remain.
See [database startup](DATABASE_STARTUP.md) for examples, restrictions, failures,
and credential rollback.

The API uses `DATABASE_URL=postgres://db:5432/?sslmode=disable` plus `PGUSER`,
`PGPASSWORD`, and `PGDATABASE` populated from the same Compose values. Passwords
are passed literally; no URL encoding is needed. Standalone deployments can
continue to supply their existing full `DATABASE_URL`.

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

Port, planet, and event jobs use PostgreSQL advisory locks to prevent overlapping transactions. Those locks do not prevent staggered replicas from applying the same scheduled interval more than once. Run one API replica until durable scheduling is implemented; see [CR-02](CODE_REVIEW_01.06.07.md#cr-02-per-invocation-locks-do-not-prevent-duplicate-scheduled-intervals).

## Health endpoints

- `GET /api/livez` confirms the process can serve HTTP.
- `GET /api/readyz` confirms the process can reach PostgreSQL.
- `GET /api/version` returns the current application version.

Use `/api/readyz` for load-balancer readiness. Use `/api/livez` for container liveness.

## Shell-free runtime and native health check

Both API Dockerfiles use a `scratch` final stage. Runtime files are limited to the
static Go executable, merged CA certificate bundle, timezone database, temporary
directory, and web assets where applicable. The application runs as `10001:10001`.
Compose keeps its read-only root, `/tmp` tmpfs, dropped capabilities, and
`no-new-privileges` setting.

```bash
docker compose exec api /app/sovereign-api healthcheck
```

The command probes `/api/livez` using `HTTP_ADDR` (default `:8080`). Wildcard
listeners map to loopback. For example, an API configured with `HTTP_ADDR=:9090`
checks port 9090 automatically. Only HTTP 200 succeeds. Proxies and redirects are
disabled, with a three-second deadline. It does not load database credentials or
initialize the application. Use a matching host port mapping if changing ports.

There is no `sh`, `wget`, `apk`, or BusyBox in the final image. Existing external
healthcheck overrides using `CMD-SHELL` must use the executable command above.
Use host-side diagnostics and `docker compose logs api` for troubleshooting.
Place custom `.crt` roots in `server/certs/` before building; the merged bundle is
copied to `/etc/ssl/certs/ca-certificates.crt`. Timezone files remain under
`/usr/share/zoneinfo`; for example, `TZ=America/Denver` remains supported.

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
