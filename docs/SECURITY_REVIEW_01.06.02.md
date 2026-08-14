# Security Review 01.06.02

- Authentication uses bcrypt cost 12 and restricted JWT validation.
- Season reset requires a current administrator identity plus a separate administration secret.
- Request limits, endpoint throttles, attachment authorization, and server-side command validation remain enabled.
- HTTP timeouts, security headers, proxy sanitization, liveness, readiness, and structured request logs remain enabled.
- The module graph and vendor tree were regenerated and verified.
- Unit tests, regression tests, race testing, `go vet`, and reachable-code vulnerability analysis passed before preparation.
- The release gate adds non-root container validation, high and critical vulnerability scanning, an SPDX SBOM, and provenance attestation.

Remaining material work is the fully atomic and durably audited season-reset transaction.
