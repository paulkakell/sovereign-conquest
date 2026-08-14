# Commit Notes 01.06.03

```text
release: promote Sovereign Conquest v01.06.03

- bump Sovereign Conquest to 01.06.03
- replace Compose build definitions with ghcr.io/paulkakell/sovereign-conquest:main
- keep SC_IMAGE and SC_PULL_POLICY configurable
- remove the redundant standalone web service
- preserve browser port 3000 and direct API port 8080
- retain loopback binding, internal PostgreSQL, read-only filesystem, dropped capabilities, and no-new-privileges
- remove obsolete Compose build variables from .env.example
- add Compose contract regression coverage
- align active application, documentation, and web badges at v01.06.03
- retire the obsolete 01.06.02 branch-scoped development-release workflow and manifest
- close dependency proposals not selected for this release
- remove merged, superseded, prerelease, and obsolete branches
- retain main as the only long-lived branch

Breaking deployment behavior:
- docker compose build is no longer used
- the web service name is removed; use the api service for logs and lifecycle operations

Application dependencies: unchanged
Database migrations: none
Rollback: restore the prior Compose file and pin SC_IMAGE to the previously verified image digest
References: SC-DEPLOY-003, SC-CI-012, SC-REPO-001
```
