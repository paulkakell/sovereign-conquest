# Validation Record 01.06.02

Completed before release preparation:
- module verification;
- unit and regression tests;
- race detector;
- `go vet`;
- zero reachable vulnerabilities from `govulncheck` under Go 1.26.6;
- JavaScript syntax validation;
- Docker Compose validation;
- combined, API, and web image builds.

The publication workflow must additionally pass container scanning, SBOM generation, local smoke testing, GHCR publication, provenance attestation, anonymous pulling, and smoke testing of the publicly pulled image. It emits a validation report and SBOM as workflow and prerelease artifacts.
