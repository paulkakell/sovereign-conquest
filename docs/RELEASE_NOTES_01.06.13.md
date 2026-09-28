# Sovereign Conquest 01.06.13

Patch release: correct Command pane alignment to the left.

## Changes and usage

The user clarified that Command contents should align to the left. The heading,
input and placeholder, Send button, feedback, Help toggle and expanded help now
share the pane's left edge. Command stays directly below Logout and above Status.
The input retains its responsive 640-pixel cap and long feedback/help still wraps.

Enter `SCAN`, `MOVE 2`, or `TRADE BUY ORE 10`, then press Enter or click Send.
Expand Help to view syntax. This presentation fix preserves command behavior,
keyboard order, APIs, configuration options, dependencies, schema and secrets.
Documentation and guarded release finalization are additive. No breaking changes.

## Upgrade

After publication, set
`SC_IMAGE=ghcr.io/paulkakell/sovereign-conquest:01.06.13` or its verified digest:

```bash
docker compose config --quiet
docker compose pull api
docker compose up -d --force-recreate api
docker compose exec api /app/sovereign-api healthcheck
```

Reload the browser to load the versioned stylesheet. Keep current secrets,
volumes and networks. For the bind profile, add
`--env-file .env -f deploy/docker-compose.bind.yml` to the Compose commands.

## Rollback

The [v01.06.12 release](https://github.com/paulkakell/sovereign-conquest/releases/tag/v01.06.12)
and source commit `8c798ca73bcc75eb300b02f64d688a5dc1df3dac` remain the rollback
baseline. Retain the current deployment's verified image digest before upgrading.
Restore that image reference and recreate only the API container, then reload.
This restores right alignment while keeping Command above Status. No database
migration, restore, credential change or volume deletion is needed.

## Validation and artifacts

See [validation/security review](VALIDATION_01.06.13.md) and
[copyable commit notes](COMMIT_NOTES_01.06.13.md).
The finalizer creates v01.06.13 only after CI, Build Validation, Database Startup
and Publish GHCR Image pass on the same current main commit. Publication retains
source-SHA image tags, digest, SBOM and provenance evidence.
