#!/usr/bin/env bash
set -Eeuo pipefail

required_variables=(
  SC_IMAGE
  SC_EXPECTED_VERSION
  SC_HOST_PORT
  SC_NAME_PREFIX
  SC_DB_PASSWORD
  SC_JWT_SECRET
  SC_INITIAL_ADMIN_PASSWORD
)

for variable in "${required_variables[@]}"; do
  if [[ -z "${!variable:-}" ]]; then
    printf 'Required environment variable is missing: %s\n' "$variable" >&2
    exit 64
  fi
done

run_id="${GITHUB_RUN_ID:-$$}"
suffix="${run_id: -6}"
prefix="$(printf '%s' "$SC_NAME_PREFIX" | tr -c '[:alnum:]_.-' '-')"
prefix="${prefix:0:10}"
network="scnet-${prefix}-${suffix}"
db="scdb-${prefix}-${suffix}"
app="scapp-${prefix}-${suffix}"
username="${prefix}${suffix}"
username="${username:0:28}"
app_started=false

show_diagnostics() {
  local exit_code="$1"
  if [[ "$exit_code" -ne 0 ]]; then
    printf '\nRelease smoke test failed with exit code %s.\n' "$exit_code" >&2
    printf '\nDocker containers:\n' >&2
    docker ps -a --filter "name=${prefix}-${suffix}" >&2 || true
    if docker inspect "$db" >/dev/null 2>&1; then
      printf '\nPostgreSQL state:\n' >&2
      docker inspect --format '{{json .State}}' "$db" >&2 || true
      printf '\nPostgreSQL logs:\n' >&2
      docker logs "$db" >&2 || true
    fi
    if docker inspect "$app" >/dev/null 2>&1; then
      printf '\nApplication state:\n' >&2
      docker inspect --format '{{json .State}}' "$app" >&2 || true
      printf '\nApplication logs:\n' >&2
      docker logs "$app" >&2 || true
    fi
  fi
}

cleanup() {
  local exit_code=$?
  show_diagnostics "$exit_code"
  docker rm -f "$app" "$db" >/dev/null 2>&1 || true
  docker network rm "$network" >/dev/null 2>&1 || true
  exit "$exit_code"
}
trap cleanup EXIT

wait_for_database() {
  local health state
  for _ in $(seq 1 120); do
    state="$(docker inspect --format '{{.State.Status}}' "$db" 2>/dev/null || printf 'missing')"
    if [[ "$state" == "exited" || "$state" == "dead" || "$state" == "missing" ]]; then
      printf 'PostgreSQL container entered state: %s\n' "$state" >&2
      return 1
    fi
    health="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$db" 2>/dev/null || printf 'missing')"
    if [[ "$health" == "healthy" ]]; then
      return 0
    fi
    sleep 1
  done
  printf 'PostgreSQL did not become healthy within the validation window.\n' >&2
  return 1
}

wait_for_application() {
  local state
  for _ in $(seq 1 120); do
    state="$(docker inspect --format '{{.State.Status}}' "$app" 2>/dev/null || printf 'missing')"
    if [[ "$state" == "exited" || "$state" == "dead" || "$state" == "missing" ]]; then
      printf 'Application container entered state: %s\n' "$state" >&2
      return 1
    fi
    if curl --fail --silent --show-error "http://127.0.0.1:${SC_HOST_PORT}/api/readyz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  printf 'Application did not become ready within the validation window.\n' >&2
  return 1
}

docker network create "$network" >/dev/null

docker run --detach \
  --name "$db" \
  --network "$network" \
  --health-cmd='pg_isready -U sovereign -d sovereign_conquest' \
  --health-interval=1s \
  --health-timeout=3s \
  --health-start-period=1s \
  --health-retries=90 \
  --env POSTGRES_USER=sovereign \
  --env "POSTGRES_PASSWORD=$SC_DB_PASSWORD" \
  --env POSTGRES_DB=sovereign_conquest \
  postgres:16-alpine >/dev/null

wait_for_database

database_url="postgres://sovereign:${SC_DB_PASSWORD}@${db}:5432/sovereign_conquest?sslmode=disable"

docker run --detach \
  --name "$app" \
  --network "$network" \
  --publish "127.0.0.1:${SC_HOST_PORT}:8080" \
  --env APP_ENV=development \
  --env "DATABASE_URL=$database_url" \
  --env "JWT_SECRET=$SC_JWT_SECRET" \
  --env "ADMIN_SECRET=${SC_ADMIN_SECRET:-}" \
  --env INITIAL_ADMIN_USERNAME=admin \
  --env "INITIAL_ADMIN_PASSWORD=$SC_INITIAL_ADMIN_PASSWORD" \
  --env UNIVERSE_SECTORS=40 \
  --env PORT_TICK_SECONDS=0 \
  --env PLANET_TICK_SECONDS=0 \
  --env EVENT_TICK_SECONDS=0 \
  --env PROTECTORATE_TICK_SECONDS=0 \
  "$SC_IMAGE" >/dev/null
app_started=true

wait_for_application

base_url="http://127.0.0.1:${SC_HOST_PORT}"
curl --fail --silent --show-error "$base_url/api/readyz" | jq --exit-status '.ok == true' >/dev/null
curl --fail --silent --show-error "$base_url/api/livez" | jq --exit-status --arg version "$SC_EXPECTED_VERSION" '.ok == true and .version == $version' >/dev/null
curl --fail --silent --show-error "$base_url/api/version" | jq --exit-status --arg version "$SC_EXPECTED_VERSION" '.version == $version' >/dev/null
curl --fail --silent --show-error "$base_url/" | grep --fixed-strings "$SC_EXPECTED_VERSION" >/dev/null

registration="$(curl --fail --silent --show-error \
  --header 'Content-Type: application/json' \
  --data "{\"username\":\"$username\",\"password\":\"Development-Only-Player-Password-2026!\"}" \
  "$base_url/api/register")"
token="$(jq --exit-status --raw-output '.token' <<<"$registration")"

curl --fail --silent --show-error \
  --header "Authorization: Bearer $token" \
  "$base_url/api/state" | jq --exit-status '.state != null' >/dev/null

curl --fail --silent --show-error \
  --header "Authorization: Bearer $token" \
  --header 'Content-Type: application/json' \
  --data '{"type":"SCAN"}' \
  "$base_url/api/command" | jq --exit-status '.ok == true or .message != null' >/dev/null

printf 'Smoke test passed for %s on port %s.\n' "$SC_IMAGE" "$SC_HOST_PORT"
