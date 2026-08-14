## 01.06.03 (2026-08-13)

Fix
- Replace Docker Compose build definitions with the combined GHCR image `ghcr.io/paulkakell/sovereign-conquest:main`.
- Remove the redundant standalone `web` service while preserving host ports 3000 and 8080 through the combined `api` service.
- Add `SC_IMAGE` and `SC_PULL_POLICY` overrides for immutable tags, digests, and controlled refresh behavior.
- Align application metadata and active web badges at 01.06.03.

Security
- Retain loopback-only application bindings, internal-only PostgreSQL, read-only application storage, dropped capabilities, and `no-new-privileges`.
- Document digest pinning for controlled deployments.

Maintenance
- Remove obsolete Compose build variables from `.env.example`.
- Add regression tests for the pull-based Compose contract.
- Update README, configuration, release notes, security review, validation record, and commit notes.

Breaking deployment behavior
- `docker compose build` is no longer part of the deployment workflow.
- The standalone `web` service is removed; operators should use the `api` service.

Dependencies
- No dependency changes.

Database
- No schema or data-format changes.

Refs
- SC-DEPLOY-003, SC-CI-012
