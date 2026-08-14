# Validation Record 01.06.03

## Required automated gates

The branch workflows must complete successfully before promotion:

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

## Dependency and database status

No dependencies or database migrations change in this release. Lock files, vendor contents, and schema rollback procedures therefore require no regeneration.

## Performance

The change removes local Compose builds and a redundant runtime service. Core game logic, database queries, scheduler behavior, and network request handling are unchanged, so a new load test is not required for this release.
