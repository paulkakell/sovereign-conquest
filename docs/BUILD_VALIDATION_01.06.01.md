# Build Validation 01.06.01

On August 13, 2026, the branch-scoped correction gate completed successfully before committing the supported Docker base tags.

Validated artifacts:

- combined API and web image from the repository root;
- split API image from `server/Dockerfile`;
- split web image from `web/Dockerfile`.

The validated build base is `golang:1.26.5-alpine3.24`. The API runtime base is `alpine:3.24`, and both API images use the unprivileged runtime identity defined in their Dockerfiles.

This document records build evidence only. Dependency regeneration and vulnerability validation remain separate release blockers documented in `docs/VALIDATION_01.06.01.md`.
