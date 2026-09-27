# Commit notes: 01.06.06

```text
fix(security): remove vulnerable API runtime packages in 01.06.06

Replace Alpine runtime stages with scratch for combined and API-only images.
Keep the static API, custom CA trust, timezone data, non-root UID, and web assets.
Replace wget with a native healthcheck that honors HTTP_ADDR and rejects
redirects, proxies, failed status codes, and requests exceeding three seconds.

Check runtime inventory and block all six reported CVEs, including medium
and unfixed findings. Scan and smoke-test before updating GHCR public tags.
Add regression, integration, scan-policy, and release-finalization coverage.
Advance version metadata and document deployment, validation, and rollback.

Fixes: CVE-2026-85091, CVE-2026-58469, CVE-2026-58470,
       CVE-2026-58471, CVE-2026-58472, CVE-2025-60876
Base: d5bf909aef936ae0f619c3d053b57f3d89c212da (v01.06.05, PR #24)
Compatibility: no API, schema, or persistent data change; runtime shell removed.
Validation: see docs/VALIDATION_01.06.06.md and exact-commit GitHub checks.
```
