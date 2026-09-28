# Sovereign Conquest 01.07.00

Additive administrator feature with authentication fixes. Based on v01.06.13
(commit 609a60c). No runtime dependency or environment-variable changes.

- New **User Management** page: search, status filters, pagination, account editing,
  password resets and associated player/ship/cargo corrections.
- Timed or indefinite suspension, manual bans and restoration with required reasons.
- Enforce current account status/session version across every protected endpoint.
  Password changes now return a replacement bearer token and invalidate old tokens.
- Protect against self-lockout and concurrent administrator demotions. Preserve
  renamed bootstrap accounts across restarts.
- Apply additive database migrations, revision conflict checks and transactional
  audit logging without recording passwords or hashes.
- Add real PostgreSQL integration, migration reversal, concurrency, session
  benchmark, form regression and release-safeguard coverage.

See [User Management and API](https://github.com/paulkakell/sovereign-conquest/blob/v01.07.00/docs/USER_MANAGEMENT.md),
[validation](https://github.com/paulkakell/sovereign-conquest/blob/v01.07.00/docs/VALIDATION_01.07.00.md) and [commit notes](https://github.com/paulkakell/sovereign-conquest/blob/v01.07.00/docs/COMMIT_NOTES_01.07.00.md).
IDs and system timestamps are read-only; level is derived from XP.

API paths and existing response fields remain stable. Third-party clients changing
passwords must save the replacement token or sign in again. Accounts requiring a
password change can access only their state and password-change endpoint.

Release source tag `v01.07.00` must target the validated main commit. The existing
release finalization gates require CI, container builds/scans, image publication
and database-startup checks before creating the immutable tag and GitHub release.
GHCR publishes `01.07.00`, a source-SHA tag and provenance/SBOM evidence.

Rollback: preserve v01.06.13 and its image digest. Close public ingress first:
old binaries ignore bans/suspensions/session revocation. Follow the detailed
moderation-aware rollback procedure in USER_MANAGEMENT.md before reopening.
