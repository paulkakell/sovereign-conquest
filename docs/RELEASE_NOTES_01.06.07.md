# Sovereign Conquest 01.06.07

Patch release fixing database startup with existing PostgreSQL volumes.

The `db` container now checks the database and password specified by Compose on
every start. Missing databases/logins and mismatched or expired credentials are
reconciled before the service becomes healthy. Existing data and ownership are
preserved. The API receives literal credentials through PostgreSQL environment
variables so reserved URL characters cannot break its connection.

Health checks require SCRAM on IPv4 loopback even when an existing HBA file trusts
localhost. The original HBA is included without being rewritten. A disabled
bootstrap administrator is enabled only when it is the configured login being
repaired; unrelated disabled administrators remain disabled.

Update the repository checkout, including `docker/db/`, and recreate both services:

```bash
docker compose config --quiet
docker compose pull
docker compose up -d --force-recreate --wait db api
```

The expected source tag is `v01.06.07`; the application image tag is `01.06.07`.
Release finalization waits for CI, Build Validation, Database Startup, and Publish
GHCR Image on the same main commit. Do not treat these expected tags as published
until those gates succeed.

No application schema migration, gameplay change, API break, or dependency update.
Deployment requirements: include the mounted script directory, use nonempty
credentials, and use a nontemplate database. Environment values now take effect
on every database startup. If changing usernames on an existing populated
database, review schema/table ownership separately. Password rotation needs no
ownership change.

Rollback: restore the previous environment and recreate `db` and `api` with these
scripts to restore the previous password, then restore the prior Compose and
application version. Preserve `db_data`. Newly created roles/databases are kept.
See [startup and rollback examples](DATABASE_STARTUP.md),
[validation](VALIDATION_01.06.07.md), and [commit notes](COMMIT_NOTES_01.06.07.md).
