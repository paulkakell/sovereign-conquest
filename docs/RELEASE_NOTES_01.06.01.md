# Sovereign Conquest 01.06.01 Release Notes

01.06.01 is a corrective security and reliability release.

## Corrected behavior

- Fresh PostgreSQL databases now create season columns before season indexes.
- The web interface loads assets from the paths used by both deployment images.
- Forced bootstrap-administrator password changes are reachable from the browser.
- Planet transfer, citadel upgrade, corporation bank, market, and route commands serialize correctly.
- Activity and event panels use the API's actual JSON fields without duplicating the recent log set.
- The bug-report page shares the active session token and live version metadata.
- The conflicted secondary Docker workflow is removed.
- Soft wipe resets the documented progression fields, checks every database operation, restores starting discovery, and writes an administrator audit event.

## Security and operations

- Direct dependencies move to patched compatible releases and the vendor tree is rebuilt.
- Legacy `RealIP` handling is replaced with deployment-specific client-IP middleware.
- Request-size controls, route-class rate limits, security headers, complete HTTP timeouts, readiness checks, and structured logging are added.
- Background jobs use PostgreSQL advisory locks so replicas do not multiply scheduled work.
- Production mode rejects default secrets and insecure database transport unless explicitly overridden.
- CI now includes unit, integration, regression, race, static-analysis, dependency, web, Compose, and container-build gates.
- Release publishing is tag-gated and includes provenance and SBOM generation.

## Compatibility

Existing API field names remain unchanged. The browser was corrected to match them. Existing databases are migrated additively. Active bearer tokens issued before an administrator password change should be treated as untrusted and deployments should rotate `JWT_SECRET` during this release.
