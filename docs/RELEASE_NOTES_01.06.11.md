# Sovereign Conquest 01.06.11

Patch release fixing all four reported high-severity CodeQL path-injection
alerts (#2, #3, #4, #5). They affected two file metadata lookups and two file
serving calls in the SPA handler. The old lexical path cleanup did not prevent
symlinks beneath `WEB_ROOT` from exposing files outside that directory.

## Changes

- **Fix:** Resolve web assets, directory indexes and SPA fallback pages through
  `os.Root`. Serve the checked open file descriptor and reject escapes with 404.
- **Fix:** Reject traversal components and malformed paths without logging or
  returning filesystem errors. Retain HTTP behavior for legitimate web requests.
- **Additive:** Exploit regressions, a concurrent symlink replacement test,
  a static-serving benchmark, a high/critical SARIF build gate, stored scan
  evidence and post-analysis verification of default-branch alert resolution.
- **Compatibility:** Absolute or escaping symlinks in custom web layouts are
  intentionally unsupported. The bundled assets and safe relative symlinks work
  unchanged. No API schema, database, dependency, gameplay or secret change.

## Upgrade

1. Retain the current Compose/environment files, a database backup and the
   verified previous image digest. Keep all existing database mounts.
2. If using a custom `WEB_ROOT`, place its public assets inside that directory.
   Replace absolute or escaping symlinks with actual files or safe relative links.
3. After release gates publish the image, set
   `SC_IMAGE=ghcr.io/paulkakell/sovereign-conquest:01.06.11` or its verified digest.
4. Run `docker compose config --quiet`, `docker compose pull api`, then
   `docker compose up -d --force-recreate api`.
5. Check `/api/readyz`, the main page, static assets and
   `docker compose exec api /app/sovereign-api healthcheck`.

For the bind profile, add `--env-file .env -f deploy/docker-compose.bind.yml`
to Compose commands and retain the existing networks, scripts and bind mount.
No credential rotation or database recreation is required.

## Rollback

Restore the retained Compose/environment files and verified 01.06.10 image
reference, then recreate the API without deleting volumes. The source baseline
is `v01.06.10` at `d4654ff3234ce03382a51a9269352acd2a006ea3`.
The prior publication's verified image is:

```text
ghcr.io/paulkakell/sovereign-conquest@sha256:bd1f34c3c6f22b2e224f575a025a6308eeb7e472330add4b136d9e85b32e6af0
```

It is recorded in publication run 36368207125. There is no schema rollback. Reverting restores the vulnerable web handler;
remove unsafe web symlinks before using the prior release.

## Validation and artifacts

See [validation and security review](VALIDATION_01.06.11.md),
[web file security](WEB_FILE_SECURITY.md), [configuration](CONFIGURATION.md),
and [copyable commit notes](COMMIT_NOTES_01.06.11.md).
The guarded finalizer creates `v01.06.11` only when CI, Build Validation,
Database Startup and Publish GHCR Image succeed on the same current main commit.
Build Validation now fails on high/critical CodeQL findings and verifies main's
alert state. Source-SHA image tags, SBOMs, image scans, digest and provenance
evidence remain part of publication. SARIF artifacts are retained for 30 days.
