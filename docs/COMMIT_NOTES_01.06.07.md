# Commit notes

```text
fix(db): reconcile Compose database credentials on startup (01.06.07)

Check the configured database and password before PostgreSQL becomes healthy.
Create missing databases/logins and repair mismatched, expired, or disabled
credentials on existing volumes while preserving data and object ownership.

Use a private startup server, quoted SQL, credential-aware health checks, and
literal PostgreSQL environment variables for the API connection. Fail closed
when reconciliation cannot complete.

Add real PostgreSQL regression coverage and a required database release gate.
Update version metadata, changelog, deployment examples, release notes,
security review, and credential rollback instructions.

No application schema, gameplay, API, or dependency changes.
```
