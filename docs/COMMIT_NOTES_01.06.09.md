# Commit notes for 01.06.09

Local implementation commit: `547f21d`; branch: `compose-settings-01.06.09`.
GitHub publication was authorized on 2026-09-27; hosted validation follows branch publication.

```text
fix(compose): verify deployment settings and require explicit secrets (01.06.09)

Require signing and bootstrap secrets before Compose startup.
Leave required example secrets blank and ignore local environment files.
Pin both deployment profiles to the release image and protect the script mount.
Pass TRUST_PROXY_HEADERS through with a false default.
Document every setting, profile ports, TLS requirements, and bootstrap behavior.
Add real Compose rendering regressions and gated release finalization.
Update version, changelog, release notes, validation, and rollback instructions.

Configuration tightening: incomplete .env files now fail before startup.
No API, schema, dependency, database script, or gameplay changes.
Base: 2afa856 (v01.06.08, PR #28).
```
