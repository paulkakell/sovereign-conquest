# Commit notes for 01.07.00

```text
feat(admin): add user management, suspensions and bans (01.07.00)

Add searchable admin account/player editing and password resets.
Add timed or indefinite suspension, bans and audited restoration.
Enforce account status and session revocation on every protected endpoint.
Prevent self-lockout, stale edits and concurrent admin demotions.
Preserve renamed bootstrap accounts and rotate password-change sessions.
Add idempotent migrations and PostgreSQL, concurrency and form tests.
Document account/API fields, release validation and safe rollback.
```

Classification: additive feature and authentication fix. Schema changes are
additive. Base: v01.06.13 / 609a60c. Implementation commit: `6f079814aee3d7570ae3d292300035ba3615d12a`.
[PR #34](https://github.com/paulkakell/sovereign-conquest/pull/34) includes the
validation follow-up and final merge commit. No hash is embedded into its own commit.
