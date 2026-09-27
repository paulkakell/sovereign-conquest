# Recover the conquest deployment

The supplied September 27 logs show PostgreSQL running and rejecting the API's
password for `conquestapp`. The supplied Compose file uses the stock image with
no startup repair scripts. Its bind mount already contains a PostgreSQL cluster,
so setting `POSTGRES_PASSWORD` does not update the password stored in that cluster.
`pg_isready` does not authenticate the configured credentials. Both published
port entries also resolve to `127.0.0.1:5000:8080`; only one is needed.

The earlier 01.06.07 work was developed separately. This 01.06.08 update
integrates it with the 01.06.06 security changes. Container integration is a
required release gate; see the validation record for actual results.

## Immediate recovery using the running containers

Extract this source package, then run these commands from its root on the Docker
host. They use the container names from the uploaded logs and do not require a
Compose file in the current directory. Retain a database backup and the previous
deployment configuration before changing credentials.

```bash
docker exec -i conquest-db-1 bash < docker/db/repair-password.sh &&
docker restart conquest-api-1
docker logs --since=2m --tail=50 conquest-api-1
```

The repair reads the running DB container's `POSTGRES_USER`, `POSTGRES_PASSWORD`,
and `POSTGRES_DB`; it never prints or embeds their values in command arguments.
It connects through the local Unix socket as the existing configured
administrator and sets that role's password, then requires a successful TCP
database query. The password is always set because legacy localhost trust can
otherwise produce a false successful password check. The startup wrapper
separately enforces SCRAM for its health probes. Existing rows, databases,
ownership, and role privileges are preserved. If the local administrator login
fails, it exits with `local_administrator_password_repair_failed`; use the
original administrator or install the startup wrapper below. Authentication
rules are not relaxed. This one-time command cannot create a missing login or
database, clear password expiry, or change the API container's environment.

If `.env` has been edited since container creation, recreate both services with
the intended settings before relying on the one-time command, or install the
startup wrapper and recreate once. A plain restart keeps the old environment.
If the script reports `password_updated` but the API still fails, compare
the API and DB environment sources without posting their secret values.

## Install repair on every database start

The tailored standalone file is `deploy/docker-compose.bind.yml`. It keeps:

- Database data at `/dockershare/containers/conquest/db`.
- The `conquest-internal` network and external `containers-external` network.
- The combined web/API service on `127.0.0.1:5000` by default.
- The API's read-only filesystem, dropped capabilities, and non-root image user.

It mounts the startup scripts from `/dockershare/containers/conquest/config/db`.
That directory must exist; missing script directories cause deployment to fail
instead of silently mounting an empty directory. The new profile passes database
credentials through `PGUSER`, `PGPASSWORD`, and `PGDATABASE`, which the existing
pgx driver already supports, so URL punctuation does not break passwords.

From the extracted source root, install the reviewed configuration:

```bash
install -d -m 0755 /dockershare/containers/conquest/config/db
install -m 0644 docker/db/*.sh /dockershare/containers/conquest/config/db/
install -m 0644 deploy/docker-compose.bind.yml /dockershare/containers/conquest/config/compose.yml
```

Keep the environment values in the stack manager if it supplies them. For a CLI
deployment, save them in `/dockershare/containers/conquest/config/.env` and protect
that file with mode `0600`. Set `WEB_PORT=5000`; `API_PORT` is unused in this
single-port profile. Preserve `POSTGRES_DB=conquest` and `POSTGRES_USER=conquestapp`
to retain the intended database and role. No secret is included in this package.

Run using explicit paths, from any directory:

```bash
docker compose -p conquest \
  --env-file /dockershare/containers/conquest/config/.env \
  -f /dockershare/containers/conquest/config/compose.yml config --quiet

docker compose -p conquest \
  --env-file /dockershare/containers/conquest/config/.env \
  -f /dockershare/containers/conquest/config/compose.yml \
  up -d --force-recreate --wait --wait-timeout 180 db api

docker logs --since=3m --tail=80 conquest-db-1
docker logs --since=3m --tail=50 conquest-api-1
curl --fail --silent --show-error http://127.0.0.1:5000/api/readyz
```

Use the replacement file as a standalone configuration, not as an override of
the old Compose file; Compose can merge the old mount and port lists. Keep the
existing project name `conquest` when updating through a stack manager. The
external network must already exist, as it did in the supplied configuration.
The default application image is the existing `01.06.06` release so recovery
does not depend on an unpublished `01.06.08` image. Override `SC_IMAGE` after a
new release passes all gates. The database wrapper itself is provided by the
read-only script mount, not by updating the API image.

Expected DB log events include `checking_credentials`, optionally
`reconciling_database_and_role`, then `credentials_verified`. The DB health check
requires that startup completes and a TCP `SELECT 1` succeeds. The API readiness
endpoint must then return HTTP 200. The health query uses the configured
database and login every five seconds. See [startup details](DATABASE_STARTUP.md)
for existing-volume behavior and restrictions.

## Rotate the shared credentials

Use different randomly generated values for the database password, JWT signing
secret, and administrator password. Recreate both services after changing the
database credential. Changing the JWT secret invalidates existing sessions.
Changing `INITIAL_ADMIN_PASSWORD` does not reset an existing administrator's
password; use the application's password-change feature for that account.

## Rollback

Retain the prior Compose file, environment, and application image digest.
To undo a password rotation, restore the previous environment and recreate the
DB once while the startup wrapper is still installed. Then restore the previous
Compose file and API image. Keep `/dockershare/containers/conquest/db` intact.
Removing a container does not roll back credentials stored in that directory.
Do not delete the database directory or use a volume-wiping reset for this issue.
Newly created roles/databases remain and require a separate, reviewed cleanup.
No schema migration or rollback migration is involved.

References: [official PostgreSQL image initialization](https://hub.docker.com/_/postgres),
[pg_isready authentication behavior](https://www.postgresql.org/docs/16/app-pg-isready.html),
and [psql variable quoting](https://www.postgresql.org/docs/16/app-psql.html).
