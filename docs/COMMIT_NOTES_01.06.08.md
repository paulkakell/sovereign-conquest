# Commit notes

```text
fix(db): recover existing-volume credentials and integrate startup checks (01.06.08)

- Combine unpublished 01.06.07 startup work with the 01.06.06 API security release.
- Reconcile the configured database/login before authenticated DB readiness.
- Add live password recovery with quoted SQL and redacted failures.
- Preserve the conquest data bind mount and networks; publish port 5000 once.
- Pass credentials literally through PGUSER, PGPASSWORD, and PGDATABASE.
- Add recovery regression tests and gate image publication on database tests.
- Update version, changelog, deployment instructions, release notes, and rollback.

Fix/additive; no schema, API, gameplay, or dependency format change.
Container integration/build/security gates remain pending where Docker is unavailable.
Base: c3de81af4d471e7cf327e6f43df96ff321a6b04a (v01.06.06, PR #25).
Reuses local work a44c249 and 9b0d400. No separate issue reference.
```
