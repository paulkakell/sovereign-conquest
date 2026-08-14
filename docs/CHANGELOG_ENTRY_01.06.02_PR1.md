## v01.06.02 PR 1 remediation (2026-08-13)

Fix
- Remove the trusted client network address from structured HTTP request logs.
- Add regression coverage that fails when a proxy-derived client address appears in log output.
- Standardize active web, bug-report, README, release-note, and pull-request version badging at `v01.06.02`.
- Add regression assertions that reject stale version badges on active web pages.

Why
- GitHub Code Scanning identified a high-severity clear-text logging path from `CF-Connecting-IP` or `X-Real-IP` into the request logger.
- Client addresses are still required transiently for rate limiting and proxy normalization, but they do not need to be persisted.
- The release branch had reached version `01.06.02`, while a few release-facing labels still referenced the earlier candidate without the required `v` prefix.

Classification
- Security fix and release-metadata correction.
- Non-breaking: no API, command, database, configuration, authentication, or deployment contract changes.

References
- Pull request: `#1`
- GitHub Code Scanning alert: `#1`
- Log-privacy fix: `e9e49bce495c5f7619063b9305cda2b9caaf9b3e`
- Log-privacy regression test: `7df590cae5bd734929b43e0a481b999c36af09e3`
- README badge correction: `3d33fa9d9e692f056c068c0d092993abccd9d3c7`
- Web badge regression test: `4bd5d0546b71a1e44c1e9af747925d6fe628437f`

Rollback
- Revert the badge and documentation commits if release labeling must be revised.
- Do not restore clear-text client-address logging; replace it with an approved privacy-preserving identifier if correlation is required.
