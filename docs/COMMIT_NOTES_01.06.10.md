# Commit notes for 01.06.10

Implementation commit: `6f088424a78e85a05fd59c57702b367e0bbeea22`; PR #30.

```text
fix(auth): validate startup secrets and enforce bcrypt limits (01.06.10)

Validate all four effective secrets before API database access in every environment.
Log every invalid setting with its requirement, without exposing secret values.
Keep the reset key optional and preserve database credential-source precedence.
Remove built-in database, JWT and bootstrap credential fallbacks.
Enforce 8-72 UTF-8 bytes for registration and password changes with HTTP 400 errors.
Add boundary, Unicode, redaction, process and real-container/database regressions.
Update version, changelog, configuration/API docs, release notes and rollback.

Compatibility: invalid development secrets now prevent startup.
No schema, dependency, gameplay or password-hash changes.
Base: 1b1b03d (v01.06.09, PR #29).
```
