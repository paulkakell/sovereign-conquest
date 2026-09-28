# Sovereign Conquest Configuration

This guide describes the configuration contract for version 01.06.10.

## Deployment profiles

`docker-compose.yml` is a local-development profile. It pulls the combined API and web image from GHCR and does not contain Docker build definitions. PostgreSQL remains a separate service on the internal Compose network.

Both profiles default to `ghcr.io/paulkakell/sovereign-conquest:01.06.10`. Use that tag only after publication gates pass. A verified digest gives immutable image identity; the publication workflow can refresh a version tag when rebuilding its base images. `main` remains an explicit opt-in moving tag.

The Compose profile uses local database transport and publishes loopback-only development ports. Do not expose it directly to the public Internet.

## Published image tags

The active GHCR workflow publishes:

| Tag | Purpose |
|---|---|
| `main` | Moving image for the current default branch |
| `01.06.10` | Release image; published after successful release gates |
| `sha-<source-sha>` | Source-specific traceability |

The candidate image is scanned and smoke-tested against fresh PostgreSQL before public tags move. The same validated image is pushed and attested. Scans include unfixed vulnerabilities and explicitly reject every CVE listed in the 01.06.06 security review, regardless of severity.

## Compose image settings

| Setting | Default | Purpose |
|---|---|---|
| `SC_IMAGE` | `ghcr.io/paulkakell/sovereign-conquest:01.06.10` | Combined API and web image |
| `SC_PULL_POLICY` | `always` | Pull on startup; use `missing` for cached releases or `never` for an already loaded local image |
| `WEB_PORT` | `3000` | Browser-facing host port |
| `API_PORT` | `8080` | Compatibility host port for direct API access |

Keep WEB_PORT and API_PORT distinct in the root profile. The tailored
`deploy/docker-compose.bind.yml` profile uses WEB_PORT only, defaulting to 5000,
and keeps the supplied conquest data path and networks. Because `.env.example` explicitly sets `WEB_PORT=3000`, loading that file overrides the bind-profile fallback. Set `WEB_PORT=5000` for the conquest installation. See
[recovery and installation](DB_RECOVERY_01.06.08.md).

Both root-profile host ports reach port 8080 in the same combined container. The API remains available below `/api` on either port. Use free ports in 1-65535. Both are bound to `127.0.0.1`, so another computer cannot access them directly. A proxy on the Docker host can target `http://127.0.0.1:3000`; a container proxy on the tailored profile's `containers-external` network can target `http://api:8080` if that alias is unique on the shared network. Otherwise assign a stack-specific alias in an override and use it as the upstream. `127.0.0.1` inside a proxy container refers to the proxy itself.

```bash
# Standalone bind profile with the shared example copied to .env:
WEB_PORT=5000 docker compose --env-file .env -f deploy/docker-compose.bind.yml config --quiet
WEB_PORT=5000 docker compose --env-file .env -f deploy/docker-compose.bind.yml up -d
```

The tailored profile requires the existing `containers-external` network and the
scripts installed at `/dockershare/containers/conquest/config/db`. Keep its
`conquest` project name and existing data path when updating that deployment.

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

## Secrets and account settings

The example deliberately leaves secrets blank. Fill the three required values
before `docker compose config --quiet` or startup. Generate a separate value for
each variable, for example by running `openssl rand -hex 32` for each. Do not
reuse the database password for session signing or administrator authentication.

| Setting | Example/default | Verified behavior |
|---|---|---|
| `POSTGRES_USER` | `sovereign` | Required; also passed to the API as `PGUSER` |
| `POSTGRES_PASSWORD` | Empty, required | Passed literally as `PGPASSWORD`; applied during database startup |
| `POSTGRES_DB` | `sovereign_conquest` | Required; also passed as `PGDATABASE`; cannot be `template0` or `template1` |
| `JWT_SECRET` | Empty, required | Use at least 32 random characters; rotating it invalidates signed sessions |
| `ADMIN_SECRET` | Empty, optional | Empty disables `/api/admin/soft_wipe`; when set, use at least 32 random characters; the endpoint also requires an authenticated administrator |
| `INITIAL_ADMIN_USERNAME` | `admin` | Bootstrap account name; changing it can create another administrator |
| `INITIAL_ADMIN_PASSWORD` | Empty, required | Use 16-72 random ASCII characters; bcrypt rejects values above 72 bytes; existing administrator passwords are not reset at startup |

Role and database names must be 1-63 bytes without tabs or newlines. The database
wrapper preserves existing privileges and ownership. A fresh official PostgreSQL
cluster grants its bootstrap `POSTGRES_USER` superuser privileges; this local
profile is not a separate least-privilege application-role deployment.

Single-quote `.env` values containing `$`, for example
`POSTGRES_PASSWORD='example-only-$-value'`. Escape a single quote inside that
quoted value with a backslash. No URL encoding is required. Compose's host shell
values override `.env`, so clear stale exported settings before verification.
Use `config --quiet`; full `config` output includes resolved secrets. Keep `.env`
private (`chmod 600 .env`); Git ignores it. Compose's required-value checks reject
missing/empty values. The API additionally enforces the following limits at
every normal startup, including `APP_ENV=development`, before connecting to the
database, changing the schema, creating an administrator, or starting jobs.

| Value | Startup requirement | Upper limit |
|---|---|---|
| `POSTGRES_PASSWORD` | Effective database password must be nonempty | No application-defined maximum |
| `JWT_SECRET` | At least 32 bytes after trimming surrounding whitespace | No application-defined maximum |
| `ADMIN_SECRET` | Empty disables season resets; otherwise at least 32 bytes after trimming | No application-defined maximum |
| `INITIAL_ADMIN_PASSWORD` | 16-72 bytes after trimming surrounding whitespace | 72 bytes, required by bcrypt |

ASCII characters each occupy one byte; Unicode characters may occupy several.
For example, 36 copies of `é` occupy 72 UTF-8 bytes; 37 exceed the password limit.
Generate four independent values with four separate runs of `openssl rand -hex 32`.
Each result is 64 ASCII characters and fits every limit. Database passwords are
used literally; bootstrap passwords are trimmed before hashing, as before.
Signing/reset keys retain their literal value but cannot use surrounding
whitespace to meet the minimum. Avoid surrounding whitespace in all secrets.

Compose supplies `POSTGRES_PASSWORD` to the API as `PGPASSWORD`. Validation uses
the database driver's resolved password, preserving its URL, environment,
service-file, and password-file precedence. Direct deployments can continue
supplying credentials in `DATABASE_URL` or through the driver's supported
sources. No known database, JWT, or bootstrap credential is supplied as a
configuration fallback when a secret is missing.

Invalid settings cause exit status 1 and one descriptive log entry listing every
detected secret failure. For example:

```text
configuration validation failed: JWT_SECRET: must contain at least 32 bytes after trimming surrounding whitespace (32 ASCII characters); INITIAL_ADMIN_PASSWORD: must contain 16-72 bytes after trimming surrounding whitespace (16-72 ASCII characters; bcrypt limit)
```

The log names variables and requirements; it never includes their supplied
values, hashes, or connection URLs. Invalid database syntax produces a fixed
`DATABASE_URL` diagnostic rather than a driver error that might contain secrets.
`docker compose logs api` shows these errors. Fix the named values and recreate
the API. Recreate both `db` and `api` after changing database credentials.
With `restart: unless-stopped`, invalid startup will repeat until corrected.
Compose itself rejects empty required settings before an API container is
created; those earlier errors appear in the Compose/deployment log.
The separate `healthcheck` command remains independent of startup configuration.

## Required production settings

`APP_ENV=development` matches the bundled PostgreSQL service, which has no TLS configuration. Secret validation applies to every environment. `APP_ENV=production` additionally requires database transport protection. Changing only this value makes the bundled stack fail validation because `DATABASE_URL` is fixed to `sslmode=disable`.

`DATABASE_URL`, `HTTP_ADDR`, and `WEB_ROOT` are fixed container settings, not `.env` overrides in these profiles. Configure PostgreSQL TLS and certificates in a separate Compose override before enabling production, for example setting the API's `DATABASE_URL` to `postgres://db:5432/?sslmode=verify-full&sslrootcert=/run/certs/ca.crt` with the matching CA mount and server certificate. Setting that URL alone does not enable TLS on PostgreSQL. HTTPS on a reverse proxy protects browser traffic, not this database connection. Other `APP_ENV` values do not enable production checks.

| Setting | Purpose | Production requirement |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection | Must use `sslmode=require`, `verify-ca`, or `verify-full` |
| `JWT_SECRET` | Signs user sessions | At least 32 characters from a cryptographic random source |
| `INITIAL_ADMIN_USERNAME` | Bootstrap administrator name | Set to the intended operator account |
| `INITIAL_ADMIN_PASSWORD` | Initial bootstrap value | 16-72 ASCII characters and unique to this deployment |
| `ADMIN_SECRET` | Enables HTTP season reset with administrator authentication | Optional; at least 32 characters when enabled |
| `HTTP_ADDR` | API bind address | Defaults to `:8080` |
| `WEB_ROOT` | Static web directory | Combined image uses `/app/web` |

The service refuses production startup when these requirements are not met.

## Proxy configuration

`TRUST_PROXY_HEADERS=false` is the safe default in `.env.example`; both Compose profiles now pass its override to the API. The API removes incoming forwarding headers before they reach the router.

Set `TRUST_PROXY_HEADERS=true` only when every request reaches the API through a controlled reverse proxy that replaces `X-Real-IP` or `CF-Connecting-IP`. Do not enable it when clients can connect directly to the API port.

## Game and scheduler settings

| Setting | Default | Behavior |
|---|---:|---|
| `UNIVERSE_SEED` | `2002` | Signed 64-bit generation seed; does not regenerate an existing map |
| `UNIVERSE_SECTORS` | `200` | Initial map size; validation accepts 2-1000000; generation rounds 2-19 up to 20; existing maps are retained |
| `TURN_REGEN_SECONDS` | `120` | Seconds per regenerated turn; minimum 10; applied on player actions/reads |
| `PORT_TICK_SECONDS` | `60` | Port regeneration interval; `<=0` disables; positive values below 5 become 5 |
| `PLANET_TICK_SECONDS` | `60` | Planet production interval; `<=0` disables; positive values below 5 become 5 |
| `EVENT_TICK_SECONDS` | `60` | Event-generation interval; `<=0` disables; positive values below 10 become 10 |
| `PROTECTORATE_TICK_SECONDS` | `60` | Protectorate update interval; `<=0` disables; positive values below 10 become 10 |

Numeric parse errors currently fall back to application defaults; supply whole numbers rather than unit suffixes. Recreate the API after editing these values.

Port, planet, and event jobs use PostgreSQL advisory locks to prevent overlapping transactions. Those locks do not prevent staggered replicas from applying the same scheduled interval more than once. Run one API replica until durable scheduling is implemented; see [CR-02](CODE_REVIEW_01.06.04.md#cr-02-per-invocation-locks-do-not-prevent-duplicate-scheduled-intervals).

## Compose service settings

| Setting | Verified purpose and limits |
|---|---|
| `db.image=postgres:16-alpine` | PostgreSQL 16 matches the data path and scripts; tag can receive minor/base updates; no major upgrade is included |
| `db.entrypoint` and `command` | Run `bash /opt/sc-db/entrypoint.sh postgres` for startup reconciliation |
| `db.volumes` | Root uses persistent `db_data`; tailored uses the existing conquest bind path; script mounts are read-only and request `create_host_path: false` |
| `db.expose=5432` | Documents the database port; no host publication; services on its network can reach it |
| `db.healthcheck` | Authenticated `SELECT 1` through `check.sh`; 5-second interval, 3-second timeout, 20 retries, 60-second start period |
| `api.depends_on` | Initial startup waits for healthy PostgreSQL; does not restart API after a later database failure |
| `restart=unless-stopped` | Both services restart on process exit; an unhealthy status alone does not restart a container |
| `api.init=true` | Uses Docker's init process for child reaping and signal handling |
| `api.read_only`, `tmpfs` | Read-only application root with writable 32 MiB `/tmp`, mode 1777 |
| `security_opt`, `cap_drop` | API forbids new privileges and drops all capabilities; image runs as UID/GID 10001 |
| `networks.backend` | Root uses a project-scoped bridge shared by API and DB; it is not declared `internal: true`, so it does not block outbound traffic |
| Tailored networks | DB joins `conquest-internal`; API also joins existing `containers-external`; names and storage remain compatible |
| `volumes.db_data` | Retained across recreations; `docker compose down -v` deletes the root database volume |

A successful configuration render does not prove that host ports are free,
script files exist, an image is published, a database accepts connections, or a
proxy can reach the application. Validate runtime health after deployment.

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

Restore the prior Compose file and `.env`, and set `SC_IMAGE` to the previously verified 01.06.09 image digest or release tag. Do not delete volumes. This release does not alter the PostgreSQL schema or stored game data, so database rollback is not required. Keep compliant secrets when rolling back; rotating a JWT key invalidates sessions, and changing the bootstrap setting does not reset an existing administrator password.

## Verification commands

```bash
# Operator check, after setting real secrets locally; never paste rendered config.
docker compose config --quiet
docker compose ps
docker compose exec api /app/sovereign-api healthcheck

# Repository regressions use disposable synthetic values and need no daemon.
python3 -m unittest discover -s scripts -p 'test_compose_config.py' -v
```

The tests cover every `.env.example` key, required-value failures, image and
port overrides, shared literal credentials, storage, health checks, and container
hardening in both profiles. CI and image publication run these regressions.

Compose semantics were checked against the official
[service reference](https://docs.docker.com/reference/compose-file/services/),
[interpolation reference](https://docs.docker.com/reference/compose-file/interpolation/),
and [environment precedence guide](https://docs.docker.com/compose/how-tos/environment-variables/variable-interpolation/).
