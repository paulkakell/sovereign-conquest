# Commit Notes v01.06.02

```text
release: merge Sovereign Conquest v01.06.02

- standardize active application, browser, README, release-note, and PR version badging at v01.06.02
- add regression checks that reject stale version badges on active web pages
- update Go build and validation toolchain to 1.26.6
- refresh chi, jwt/v5, pgx/v5, x/crypto, go.sum, and vendor
- add CodeQL, gosec, govulncheck, container scanning, SBOM, and provenance gates
- add fresh PostgreSQL integration and runtime smoke validation
- repair schema startup, browser command contracts, forced password changes, and replica-safe schedulers
- remove proxy-derived client network addresses from persistent request logs
- retain transient client-address use for rate limiting and trusted-proxy normalization
- update README, changelog, deployment, security, validation, and release notes

Breaking behavior:
- production rejects weak configuration and unverified database transport
- season reset requires an authenticated administrator plus the separate administration factor

Rollback:
- redeploy the prior immutable image digest or version tag
- retain additive schema objects unless a tested forward repair requires otherwise
- restore PostgreSQL only from a verified backup when necessary

References:
SC-DB-004, SC-WEB-005, SC-AUTH-004, SC-SEC-004, SC-RUNTIME-003, SC-CI-011, SC-REL-012, SC-SEC-005, SC-DEP-003, SC-DB-005
```
