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

No Go, JavaScript, operating-system, or GitHub Actions dependency versions change in 01.06.03. The existing module graph, checksums, and vendor tree remain unchanged.

## Database review

No schema migration or data-format change is introduced. Forward and rollback database compatibility are unchanged.

## Remaining considerations

The default `main` image tag is moving. Production deployments should override `SC_IMAGE` with an immutable tag or digest and should verify the corresponding provenance attestation before deployment.
