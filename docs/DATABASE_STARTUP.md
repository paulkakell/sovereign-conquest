# Database startup and recovery

Version 01.06.07 makes the Compose database settings authoritative at each `db`
container start. PostgreSQL 16 remains the database image and existing `db_data`
volumes remain compatible. Keep the repository's `docker/db/` directory beside
`docker-compose.yml`; it is mounted read-only. No custom database image is needed.

## Startup behavior

1. Validate the database name, role name, and nonempty password.
2. Initialize a fresh volume with the official PostgreSQL helpers. Existing
   `/docker-entrypoint-initdb.d` hooks still run once on an empty volume.
3. Read the cluster's original administrator name with PostgreSQL stopped. This
   supports volumes initialized with a different or renamed `POSTGRES_USER`.
4. Start a temporary server accessible through a private Unix socket and
   loopback TCP. Require SCRAM password authentication for the check.
5. Connect to `POSTGRES_DB` using `POSTGRES_USER` and `POSTGRES_PASSWORD` and run
   `SELECT 1`. If it succeeds, leave the role and database untouched.
6. Otherwise create the missing login or database, set the configured password,
   enable login, clear password expiry, and grant database connection permission.
   Recheck the credentials, then stop the temporary server.
7. Start PostgreSQL normally. The Compose health check requires the completion
   marker and a successful authenticated query. The API waits for database health.

The scripts never drop databases, tables, or roles. New roles created on existing
volumes are ordinary logins; new databases are owned by the configured role.
The official image still creates its initial bootstrap role as a superuser.
Existing ownership and role privileges are retained. Renaming the configured
login does not transfer existing schema/table ownership; plan that separately
when changing the application identity on a populated database. Changing only
the password preserves all existing application privileges and data.

## Settings and examples

| Setting | Required | Example and behavior |
| --- | --- | --- |
| `POSTGRES_USER` | Yes | `sovereign`; missing role is created |
| `POSTGRES_PASSWORD` | Yes | A unique secret; replaces a mismatched/expired password |
| `POSTGRES_DB` | Yes | `sovereign_conquest`; missing database is created |
| `SC_IMAGE` | Existing option | Pin the application to the validated release image/digest |

Names must contain 1 to 63 UTF-8 bytes and cannot contain tabs or newlines.
`template0`, `template1`, empty passwords, and explicitly configured host `trust`
authentication are rejected. Password punctuation is supported. Single-quote
`.env` values containing `$` to prevent Compose interpolation, following Compose
quoting rules. Avoid printing `docker compose config` output because it contains
resolved secrets; use `docker compose config --quiet` to validate it.

Fresh deployment:

```bash
cp .env.example .env
# Replace the placeholder credentials and application secrets.
docker compose config --quiet
docker compose pull
docker compose up -d --wait
```

Password rotation or repair of an existing volume:

```bash
# Back up PostgreSQL and retain the previous .env securely.
# Edit POSTGRES_PASSWORD in .env, then recreate both services.
docker compose config --quiet
docker compose up -d --force-recreate --wait db api
docker compose logs --tail=50 db
```

`docker compose restart` does not reread an edited `.env`. Use `up` with recreation
after changing values. Changing `POSTGRES_DB` creates a separate empty database
and leaves the old database intact; it does not rename or copy the game universe.
To recover a missing database, use the intended database name and the same command.
Never use `docker compose down -v` for credential recovery; it deletes the volume.

## Security and failure handling

The temporary administrator socket sits in a mode-0700 directory owned by the
PostgreSQL OS user. It accepts local trust only inside that directory; loopback
TCP uses SCRAM, and no temporary listener binds the container's network interface.
Passwords enter SQL through psql environment variables and quoted literals, never
shell-generated SQL or command arguments. Role/database names use quoted SQL
identifiers. Reconciliation does not grant superuser, create-role, or create-db.

Structured events include `checking_credentials`, `reconciling_database_and_role`,
`credentials_verified`, and failure identifiers. They contain no credentials.
Temporary PostgreSQL and SQL error output is private and removed on normal exit.
Do not enable shell tracing around these scripts or secret-bearing commands.

Broken volumes, denied connections, unsupported authentication configuration,
or unsuccessful repair stop startup without marking the database healthy. Take
a backup and investigate the reported event; the wrapper does not remove lock
files, disable database protections, or delete data to force recovery. Custom
PostgreSQL ports/listener overrides are outside this Compose profile, which uses
5432. Existing persistent HBA settings are retained for the final server; legacy
trust rules should be replaced with password authentication by the operator.

## Rollback

Keep a backup and the prior environment values before changing credentials.
To undo a password rotation, restore the prior `.env` and run the recreation
command above while these scripts remain installed. This restores the credential
without replacing the data. Then restore the prior Compose file and application
image if required. Rolling back only the application image does not undo a role
password change. Newly created databases/roles remain; remove them only after
confirming they are unused. Existing database and table ownership is unchanged.

## Regression tests

```bash
python3 scripts/test-db-startup.py
shellcheck docker/db/*.sh
```

The container suite uses disposable, uniquely named volumes and no host network
ports. It covers fresh initialization, legacy volumes, idempotency, rotation and
credential rollback, missing databases/roles, a renamed bootstrap administrator,
disabled/expired logins, quoted input, legacy trust, unsuccessful repair,
invalid configuration, readiness checks, and abrupt restart. It never uses a
deployment's `db_data` volume.

Official behavior references: [PostgreSQL image initialization](https://hub.docker.com/_/postgres),
[single-user mode](https://www.postgresql.org/docs/16/app-postgres.html), and
[psql quoting and environment variables](https://www.postgresql.org/docs/16/app-psql.html).
