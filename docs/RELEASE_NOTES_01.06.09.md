# Sovereign Conquest 01.06.09

Patch release: Docker Compose and environment verification.

Root Compose previously allowed omitted signing/bootstrap secrets to resolve to
known development defaults. It also followed the moving `main` image while the
other deployment profile used a versioned release. The proxy option documented
for operators was fixed to false in both Compose files.

This release requires explicit authentication secrets, leaves required example
secrets blank, aligns both image defaults to 01.06.09, exposes the proxy setting
with a false default, and prevents automatic creation of the root script mount.
It adds Git ignores for local credentials and documents every setting, including
fixed container values, port/network access, storage, health checks, TLS,
bootstrap behavior, and scheduler limits.

## Upgrade

1. Preserve the existing `.env`, database backup, and verified 01.06.08 image
   reference. Do not replace an existing populated environment with the example.
2. Install the updated Compose file and its `docker/db/` directory. Existing
   database script behavior is unchanged. Keep the tailored deployment's
   project name, networks, script installation path, and data bind mount.
3. Set unique `POSTGRES_PASSWORD`, `JWT_SECRET`, and `INITIAL_ADMIN_PASSWORD`.
   For existing deployments, retain their current values unless deliberately
   rotating them. `ADMIN_SECRET` enables the authenticated HTTP soft-wipe route;
   an empty value disables that route. Use 16-72 ASCII characters for the initial
   password and at least 32 random characters for signing/reset keys.
4. After the 01.06.09 image is published, update existing `SC_IMAGE` values to the
   new tag or verified digest. The example uses port 3000; set `WEB_PORT=5000`
   explicitly for the tailored conquest deployment. Keep two different root
   host ports. Leave proxy trust false unless ingress is controlled.
5. Run `docker compose config --quiet`, `docker compose pull`, and
   `docker compose up -d --remove-orphans`. For the tailored profile, pass
   `--env-file .env -f deploy/docker-compose.bind.yml` to each Compose command.
6. Check `docker compose ps`, `/api/readyz`, and
   `docker compose exec api /app/sovereign-api healthcheck` on the chosen port.

The bundled database has no TLS configuration. Changing only
`APP_ENV=production` fails startup by design. A production override must configure
PostgreSQL TLS, certificates, and a matching database connection. Consult the
[configuration guide](CONFIGURATION.md) before changing those fixed settings.

## Compatibility and rollback

The required-value checks deliberately reject incomplete environments earlier.
Configured deployments retain their API, persistence, ports, credentials, and
scheduler behavior. No schema migration, dependency update, or data conversion
is included; migration reversal and performance/load tests are not applicable.

Restore the prior Compose and environment files and use the retained verified
01.06.08 image digest or tag. Recreate services without deleting volumes. Image
rollback does not reverse an intentional password rotation; restore the prior
credential using the documented database startup reconciliation if needed.

## Validation and publication

See [validation](VALIDATION_01.06.09.md) for executed checks and hosted evidence.
Both profiles are exercised with synthetic credentials. The finalizer creates
`v01.06.09` only after CI, Build Validation, Database Startup, and Publish GHCR
Image succeed on the same current main commit. It preserves the v01.06.08 tag.
A source branch containing these notes does not mean its image is published.
