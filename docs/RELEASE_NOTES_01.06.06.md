# Sovereign Conquest v01.06.06

Security bugfix release, 2026-09-27. Base: `d5bf909aef936ae0f619c3d053b57f3d89c212da`
(`v01.06.05`, PR #24).

## Fixes

- Replace the Alpine final stage in both API Dockerfiles with `scratch`. The
  reported zlib, wget, BusyBox, busybox-binsh, and ssl_client packages are absent
  from the resulting runtime. Build tools remain isolated in build stages.
- Replace the wget shell health check with `/app/sovereign-api healthcheck`.
  It honors `HTTP_ADDR`, accepts only HTTP 200, disables proxies and redirects,
  and has a three-second deadline. It runs before database/config initialization.
- Scan and smoke-test the image before updating public GHCR tags. Reject high
  and critical findings, including unfixed ones, and all six reported CVE IDs
  regardless of severity. Verify runtime contents independently of scanner data.
- Preserve UID/GID 10001, custom CA certificates, timezone data, static web
  assets, Compose hardening, health endpoints, and application configuration.

## Additions

- Native healthcheck regression tests, container inventory and scan-policy
  verification, integration coverage for both API image layouts, and release
  finalization safeguards for `v01.06.06`.
- Updated README, configuration, security review, validation, and commit notes.

## Compatibility

No gameplay, authentication, API response, database schema, dependency manifest,
or persistent data-format changes. No migration or database rollback is needed.

Operational change: there is no shell, package manager, or wget in the runtime.
Custom scripts using `docker exec ... sh`, `apk`, or a `CMD-SHELL` healthcheck
must use host-side tools or the native healthcheck. This removes incidental
debugging utilities; it is not an application API break.

## Upgrade and verify

After the release image is published:

```bash
SC_IMAGE=ghcr.io/paulkakell/sovereign-conquest:01.06.06 docker compose pull api
SC_IMAGE=ghcr.io/paulkakell/sovereign-conquest:01.06.06 docker compose up -d api
docker compose exec api /app/sovereign-api healthcheck
docker compose logs --tail=100 api
```

Persist the selected tag or verified digest in `.env`. Rescan the running image
digest with the scanner that produced the original report. Existing containers
retain the old packages until they are recreated from the new image.

## Artifacts and validation

The finalization workflow creates `v01.06.06` and the GitHub release only after
CI, Build Validation, and GHCR publication succeed on the same current main SHA.
The combined image has version, source-SHA, and `main` tags plus provenance.
Container validation retains scan evidence. See [validation](VALIDATION_01.06.06.md)
and [security review](SECURITY_REVIEW_01.06.06.md) for evidence and limits.

## Rollback

Record the deployed digest before upgrading. If the new container cannot start,
restore that known working digest with `SC_IMAGE` and recreate only `api`.
The previous source is preserved at `v01.06.05`, commit `d5bf909aef936ae0f619c3d053b57f3d89c212da`.
The prior image tag is `ghcr.io/paulkakell/sovereign-conquest:01.06.05`.
Its runtime has the reported vulnerabilities, so rollback is temporary while
the failure is investigated. Do not remove database volumes. To revert source,
revert the release merge commit and publish a new version; never move old tags.

Copyable [commit notes](COMMIT_NOTES_01.06.06.md).
