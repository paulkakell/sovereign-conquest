# Validation Record 01.06.03

## Required automated gates

The release workflows must complete successfully before promotion:

- JavaScript syntax checks
- Go formatting verification
- module graph verification
- complete unit and regression tests
- race detection
- `go vet`
- gosec static analysis
- `govulncheck` reachable-code analysis
- Docker Compose configuration validation
- combined, API, and web source image builds
- CodeQL analysis
- fresh-PostgreSQL runtime integration test

## Compose regression coverage

`server/internal/build/compose_contract_test.go` verifies that:

- the combined GHCR image is the default;
- `SC_IMAGE` and `SC_PULL_POLICY` remain operator-configurable;
- no Compose build sections remain;
- the standalone `web` service is absent;
- host ports 3000 and 8080 remain available;
- read-only filesystem, capability dropping, and `no-new-privileges` controls remain enabled.

## Configuration validation

CI copies `.env.example` to `.env` and runs `docker compose config --quiet`. Build Validation also confirms that the rendered image list contains `ghcr.io/paulkakell/sovereign-conquest:main`.

## Documentation and version validation

- Root `VERSION`, backend `config.Version`, active web badges, cache-busting values, README, release notes, and current configuration guidance identify 01.06.03 or v01.06.03 as appropriate.
- Historical release documents retain their original version numbers.
- The obsolete 01.06.02 branch-scoped development-release workflow and manifest are removed.

## Publication validation

The active `Publish GHCR Image` workflow:

- reads and validates the root version;
- publishes `main`, `01.06.03`, and source-SHA tags;
- applies OCI version, source, and revision labels;
- publishes provenance attestation;
- scans the pushed version image for high and critical vulnerabilities;
- starts a fresh PostgreSQL database and smoke-tests the pushed image, including the reported application version.

## Dependency and database status

Application dependencies and database migrations do not change in this release. Automated dependency proposals were reviewed separately; prerelease compiler branches and unrelated major updates were not folded into v01.06.03.

## Branch consolidation validation

After the release merge:

- all open automated dependency pull requests not selected for this release are closed;
- merged, superseded, prerelease, and obsolete branches are deleted;
- a branch listing returns `main` as the only remaining branch;
- the one-time cleanup workflow is removed after successful execution.

## Performance

The change removes local Compose builds and a redundant runtime service. Core game logic, database queries, scheduler behavior, and network request handling are unchanged, so a new load test is not required for this release.
