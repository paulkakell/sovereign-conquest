# Security Review 01.06.03

## Authentication and authorization

No authentication, authorization, JWT, password, or administrator-control behavior changes in this release.

## Input validation and logging

No request parsing, command validation, attachment handling, or logging behavior changes. Existing request limits, endpoint throttles, proxy sanitization, security headers, and client-address log protections remain intact.

## Container and secrets handling

- Compose now pulls the combined GHCR image rather than building local containers.
- Runtime secrets remain environment variables and are not embedded in the Compose file.
- The application container retains a read-only filesystem, dropped capabilities, `no-new-privileges`, and a temporary `/tmp` filesystem.
- PostgreSQL remains internal and is not published on a host port.
- Application ports remain loopback-only by default.
- `SC_IMAGE` supports digest pinning to reduce moving-tag and supply-chain risk.

## Dependency validation

Application dependencies, Go module checksums, and the committed vendor tree do not change in 01.06.03. Automated dependency branches were reviewed during repository consolidation:

- Go 1.27 release-candidate proposals were rejected because prerelease compilers are not appropriate for the production image.
- Unrelated major GitHub Actions and Nginx proposals were closed for separate review rather than silently entering this deployment release.
- The obsolete development-release workflow that referenced older action versions was removed.

The full source, race, static-analysis, reachable-vulnerability, CodeQL, image-build, Compose, and fresh-database gates must remain green before promotion.

## Database review

No schema migration or data-format change is introduced. Forward and rollback database compatibility are unchanged.

## Branch and workflow review

`main` is the only long-lived branch after promotion. A one-time, least-privilege cleanup workflow removes disposed branches and is deleted after successful execution. Historical version records remain as documentation rather than executable branch-scoped automation.

## Remaining considerations

The default `main` image tag is moving. Production deployments should override `SC_IMAGE` with an immutable tag or digest and verify the corresponding provenance attestation before deployment.
