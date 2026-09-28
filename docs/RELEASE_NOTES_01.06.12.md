# Sovereign Conquest 01.06.12

Patch release: reposition and right-align the Command pane.

## Changes

Command now appears directly below the topbar and Logout button, ahead of the
Status/Sector cards, so command entry is available before the status panels.

The previous pane applied a two-column grid to the heading, command controls,
feedback and help. This separated the heading from a cramped input and left
other elements aligned inconsistently. The pane now stacks these elements
against the right edge. Input text, placeholder text, button labels, feedback,
and help text are right-aligned. The input is capped at 640 pixels and shrinks
with the pane; long feedback/help wraps, including on mobile.

Classification: fix, with additive documentation and release safeguards.
There are no breaking API, command, configuration, dependency or schema changes.
Authentication, authorization, secrets and structured request logging are intact.

## Use and upgrade

Enter `SCAN`, `MOVE 2`, or `TRADE BUY ORE 10`, then press Enter or click Send.
Expand Help for supported command syntax. The controls and help share the same
right edge on both desktop and mobile.

Once publication completes, set
`SC_IMAGE=ghcr.io/paulkakell/sovereign-conquest:01.06.12` or its verified digest,
then run:

```bash
docker compose config --quiet
docker compose pull api
docker compose up -d --force-recreate api
docker compose exec api /app/sovereign-api healthcheck
```

Retain existing secrets, database volumes, networks and settings. For the bind
profile, include `--env-file .env -f deploy/docker-compose.bind.yml` in Compose
commands. Reload the page to load the versioned stylesheet.

## Rollback

The immutable [v01.06.11 release](https://github.com/paulkakell/sovereign-conquest/releases/tag/v01.06.11)
and source commit `7a0758852727f252d1bf1e939ff73e4a6f6f3db2` remain available.
Retain the previous deployment's verified image digest before upgrading.
Restore that image reference and recreate only the API container, then reload
the page. No database restore, migration or credential change is needed.
Reverting the release commit restores the previous alignment.

## Validation and artifacts

See [validation and security review](VALIDATION_01.06.12.md) and
[copyable commit notes](COMMIT_NOTES_01.06.12.md).
The finalizer creates `v01.06.12` only after CI, Build Validation, Database
Startup and Publish GHCR Image succeed on the same current main commit.
Publication retains the image digest, source-SHA tag, SBOM and provenance.
