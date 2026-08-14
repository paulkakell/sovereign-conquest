# Commit Notes 01.06.02

```text
release(dev): publish Sovereign Conquest 01.06.02 candidate

- bump application and browser metadata to 01.06.02
- update Go build and validation toolchain to 1.26.6
- refresh chi, jwt/v5, pgx/v5, x/crypto, go.sum, and vendor
- add CodeQL, govulncheck, container scanning, SBOM, and provenance gates
- add GHCR dev, dev-01.06.02, and source-SHA tags
- verify unauthenticated GHCR pull and runtime behavior
- exclude trusted proxy addresses from persistent request logs
- retain client-address use only for rate limiting and proxy normalization
- add regression coverage preventing client-address disclosure in logs
- update README, changelog, deployment, security, validation, and release notes

Breaking behavior: none for stable tags, main, API contracts, database schemas, or configuration fields.
Rollback: redeploy the prior immutable image digest or version tag and restore the database only from a verified backup when required.
References: SC-REL-012, SC-SEC-005, SC-DEP-003, PR #1, GitHub Code Scanning alert #1
Security fix commit: e9e49bce495c5f7619063b9305cda2b9caaf9b3e
Regression test commit: 7df590cae5bd734929b43e0a481b999c36af09e3
```
