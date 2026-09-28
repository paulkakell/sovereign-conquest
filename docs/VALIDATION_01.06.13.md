# Validation and security review for 01.06.13

Base: `8c798ca73bcc75eb300b02f64d688a5dc1df3dac` (v01.06.12, PR #32).
Implementation branch: `fix/command-pane-left-align`.

## Scope and compatibility

Only Command pane CSS changes application behavior. Its outer/inner alignment,
text alignment and Help toggle margin now use the left edge. Position beneath
Logout and above Status, responsive widths, text wrapping, DOM order and command
handlers are preserved. Version/image/cache references advance to 01.06.13.

## Validation status

Local validation passed with Go 1.27.1:

- Full Go unit/regression and race suites, vet, gofmt and module verification.
- CGO-disabled static API build.
- Gosec and Govulncheck: zero reachable or imported-package vulnerabilities.
- All 219 JavaScript release/SARIF tests, including 20 new release safeguards.
- All 26 Python container-policy and database-recovery failure tests.
- JavaScript/shell syntax and diff whitespace checks.
- HTML inspection: one Command pane below Logout and above Status, unique IDs.

Docker is unavailable locally. Existing hosted workflows supply clean image
builds, real Compose validation, PostgreSQL startup/password integration, CodeQL
and image scans. Their results and release verification will be linked in the PR.

The previous browser preview was blocked by the browser URL policy. No rendered
browser result or screenshot is claimed for this CSS correction.

## Security and dependency review

Authentication, authorization, command validation, textContent rendering, secrets,
logging, health probes and metrics are unchanged. No new requests, permissions
or runtime dependencies are introduced. Manifests, checksums and vendored code
are unchanged. Govulncheck reports GO-2026-5932 only for the unused OpenPGP
package; the application imports bcrypt and has no affected reachable symbol.
Existing release safeguard coverage remains active and the new
release has equivalent checks for stale commits, failed gates and immutable tags.

## Database, performance and rollback

No schema, SQL, stored data, core logic, query or I/O change occurs. Migration,
database rollback and load tests are not applicable to this presentation fix.
The prior published v01.06.12 source tag is the rollback baseline. Restore its
retained image digest and settings to revert alignment without touching data.
The new tag must match the validated main commit and preserve the prior release.
