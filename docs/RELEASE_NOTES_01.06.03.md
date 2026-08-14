# Sovereign Conquest v01.06.03 Release Notes

## Summary

Version 01.06.03 changes the default Docker Compose deployment from local image builds to the combined GHCR image at `ghcr.io/paulkakell/sovereign-conquest:main`. It also completes repository consolidation so `main` is the only long-lived branch.

## Fixes

- Replace the API build definition with an `SC_IMAGE`-controlled GHCR image.
- Remove the redundant standalone `web` service because the combined image serves both the API and bundled client.
- Preserve browser port 3000 and direct API port 8080 as loopback-only mappings to the combined container.
- Add an always-pull default through `SC_PULL_POLICY` while permitting an operator override.
- Remove obsolete build-time variables from `.env.example`.
- Align application, documentation, and active browser version badges at v01.06.03.

## Security and reliability

- Retain the unprivileged image user, read-only root filesystem, temporary `/tmp`, dropped Linux capabilities, and `no-new-privileges`.
- Keep PostgreSQL on the internal Compose network without a host-published database port.
- Keep application ports bound to `127.0.0.1` by default.
- Permit immutable image tags or digests through `SC_IMAGE` for controlled deployments.
- Exclude unvalidated Go 1.27 release-candidate updates from this release.

## Repository maintenance

- Retire the obsolete branch-scoped 01.06.02 development-release workflow and manifest.
- Close automated dependency proposals that were not selected for v01.06.03.
- Remove merged, superseded, prerelease, and obsolete branches after promotion.
- Preserve historical release records under `CHANGELOG.md` and `docs/`.

## Compatibility

Application APIs, command formats, authentication, and database schemas are unchanged.

Deployment behavior changes:

- `docker compose build` is no longer part of the Compose workflow.
- The standalone `web` service is removed.
- Use `docker compose logs api` for the combined service.
- Use `docker compose pull` before `docker compose up -d` to refresh the image.

## Upgrade

```bash
cp .env.example .env
docker compose pull
docker compose up -d --remove-orphans
```

Existing `.env` files should add `SC_IMAGE`, `SC_PULL_POLICY`, and `API_PORT` when operator-controlled values are required.

## Rollback

Restore the previous Compose file and set `SC_IMAGE` to the prior verified image digest. No database rollback is required because this release introduces no schema or data-format changes.

## References

- SC-DEPLOY-003
- SC-CI-012
- SC-REPO-001
