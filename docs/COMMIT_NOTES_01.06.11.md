# Commit notes

```text
fix(security): resolve four CodeQL path-injection alerts (01.06.11)

Confine web assets, directory indexes and SPA fallback to os.Root.
Serve checked open files and reject traversal and symlink escapes.
Preserve normal web navigation, caching, HEAD, Range and conditional requests.
Add exploit/race regressions and a static-serving performance benchmark.
Fail Build Validation on high/critical CodeQL findings and retain SARIF evidence.
Verify main's alert resolution and add guarded 01.06.11 release finalization.
Update versions, changelog, configuration, release notes and rollback guidance.

Resolves code scanning alerts #2, #3, #4 and #5 (go/path-injection, CWE-22).
Security compatibility change: web symlinks cannot escape WEB_ROOT.
No API schema, database, dependency, gameplay or secret changes.
```
