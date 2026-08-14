# Commit Notes 01.06.03

```text
fix(compose): deploy the combined GHCR image

- bump Sovereign Conquest to 01.06.03
- replace Compose build definitions with ghcr.io/paulkakell/sovereign-conquest:main
- keep SC_IMAGE and SC_PULL_POLICY configurable
- remove the redundant standalone web service
- preserve browser port 3000 and direct API port 8080
- retain loopback binding, internal PostgreSQL, read-only filesystem, dropped capabilities, and no-new-privileges
- remove obsolete Compose build variables from .env.example
- add Compose contract regression coverage
- update active web badges, README, configuration guidance, changelog, validation, security review, and release notes

Breaking deployment behavior:
- docker compose build is no longer used
- the web service name is removed; use the api service for logs and lifecycle operations

Dependencies: unchanged
Database migrations: none
Rollback: restore the prior Compose file and pin SC_IMAGE to the previously verified image digest
References: SC-DEPLOY-003, SC-CI-012
```
