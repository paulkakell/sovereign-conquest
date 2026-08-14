# Commit Notes 01.06.02

```text
release(dev): publish Sovereign Conquest 01.06.02 candidate

- bump application and browser metadata to 01.06.02
- update Go build and validation toolchain to 1.26.6
- refresh chi, jwt/v5, pgx/v5, x/crypto, go.sum, and vendor
- add CodeQL, govulncheck, container scanning, SBOM, and provenance gates
- add GHCR dev, dev-01.06.02, and source-SHA tags
- verify unauthenticated GHCR pull and runtime behavior
- update README, changelog, deployment, security, validation, and release notes

Breaking behavior: none for stable tags or main.
Rollback: redeploy the prior immutable image digest or version tag and restore the database only from a verified backup when required.
References: SC-REL-012, SC-SEC-005, SC-DEP-003
```
