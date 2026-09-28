# Sovereign Conquest 01.06.10

Patch release: API startup secret validation and bcrypt password limits.

The API previously enforced signing/bootstrap minimum lengths only in production,
omitted the bootstrap password's bcrypt upper limit, and did not check its
effective database password before connecting. Registration and password changes
advertised 100 characters even though bcrypt rejected inputs above 72 bytes.

## Changes

- **Fix:** Validate all four secrets before database access in every environment.
  Require a nonempty effective PostgreSQL password, at least 32 bytes for JWT
  and enabled reset keys, and 16-72 bytes for the trimmed bootstrap password.
- **Fix:** Report all detected secret failures together with variable names and
  requirements, exit 1, and never print supplied values or database URLs.
- **Fix:** Remove built-in database, signing, and bootstrap credential fallbacks.
- **Fix:** Return HTTP 400 for registration/password changes outside 8-72 UTF-8
  bytes, before hashing or database access. Existing login/hash behavior remains.
- **Additive:** Boundary, Unicode, redaction, subprocess, real-container startup,
  and real-database registration/password-change regressions for both API images.

## Upgrade and compatibility

This intentionally tightens configuration: development deployments with signing
keys below 32 bytes or bootstrap passwords outside 16-72 bytes will now refuse
startup. `ADMIN_SECRET` remains optional; empty disables the HTTP soft-wipe route.
Surrounding whitespace does not satisfy key/bootstrap minimums. Database
credentials follow the existing driver's precedence and remain literal.

1. Back up PostgreSQL and retain the current Compose/environment files and a
   verified 01.06.09 image reference. Keep the existing database mount.
2. Set compliant values in `.env`. Four separately generated 64-character values
   from `openssl rand -hex 32` fit the limits. Keep existing compliant secrets.
   Rotating JWT invalidates sessions; changing the bootstrap password does not
   rotate an existing administrator's password.
3. After release gates publish the image, set
   `SC_IMAGE=ghcr.io/paulkakell/sovereign-conquest:01.06.10` or its verified digest.
4. Run `docker compose config --quiet`, `docker compose pull api`, then
   `docker compose up -d --force-recreate api`. If database credentials changed,
   recreate both `db` and `api` so database startup reconciliation applies them.
5. Check `docker compose logs api`, `/api/readyz`, and
   `docker compose exec api /app/sovereign-api healthcheck`.

For the tailored profile, add `--env-file .env -f deploy/docker-compose.bind.yml`
to Compose commands and preserve the existing networks, scripts and bind mount.
The bundled database still has no TLS configuration; production TLS requirements
are unchanged. Missing required Compose values are rejected by Compose before
container creation. Invalid nonempty values appear in API container logs.

No database migration, dependency update, game-rule change, volume change, or
password-hash conversion is included. Registration's usable maximum was already
72 bytes; longer requests now receive a descriptive 400 instead of a 500.

## Rollback

Restore the retained Compose/environment settings and verified 01.06.09 image
reference, then recreate the API without deleting volumes. Reverting code does
not undo password changes or JWT rotation. No schema rollback is necessary.
The preserved source baseline is `v01.06.09` at
`1b1b03d0eb8bb72efd4d6079292798eb2bd718d3`.

## Validation and release

See [validation and security review](VALIDATION_01.06.10.md),
[configuration](CONFIGURATION.md), [password API documentation](API_PASSWORDS.md),
and [commit notes](COMMIT_NOTES_01.06.10.md).
The guarded finalizer creates `v01.06.10` only after CI, Build Validation,
Database Startup and Publish GHCR Image pass on the same current main commit.
Image publication retains SBOM, scan, runtime inventory, digest and provenance
evidence. An unmerged source branch does not imply a published image.
