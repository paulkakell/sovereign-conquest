#!/usr/bin/env bash
# One-time repair for an already running official PostgreSQL container.
set -Eeuo pipefail
log() { printf '{"component":"db-recovery","event":"%s"}\n' "$1"; }
fail() { log "$1" >&2; exit 1; }
[[ $# == 0 ]] || fail unexpected_arguments
[[ -n ${POSTGRES_USER:-} && -n ${POSTGRES_PASSWORD:-} && -n ${POSTGRES_DB:-} ]] || fail configuration_required

export PGPORT=5432 PGUSER=$POSTGRES_USER PGPASSWORD=$POSTGRES_PASSWORD
export PGCONNECT_TIMEOUT=3 PGSSLMODE=disable PGPASSFILE=/dev/null
unset PGHOSTADDR PGSERVICE PGSERVICEFILE PGOPTIONS
verify() {
    local result
    result=$(PGHOST=127.0.0.1 PGDATABASE=$POSTGRES_DB \
        psql -X -w -qAt -v ON_ERROR_STOP=1 -c 'SELECT 1' 2>/dev/null) || return 1
    [[ $result == 1 ]]
}
# Use the existing local administrator session, without changing HBA rules.
# Always set the requested password: legacy localhost trust can make a TCP
# query succeed even with the wrong password, so it cannot authorize skipping
# this repair. The startup wrapper separately enforces SCRAM for health probes.
# psql imports the password from the environment and quotes it as an SQL literal.
# Suppress SQL/client output because an error could include secret-bearing SQL.
if ! PGHOST=/var/run/postgresql PGDATABASE=postgres \
    psql -X -w -q -v ON_ERROR_STOP=1 >/dev/null 2>&1 <<'SQL'
SET log_statement = 'none';
SET log_min_error_statement = 'panic';
SET log_min_duration_statement = -1;
SET log_min_duration_sample = -1;
SET log_statement_sample_rate = 0;
SET password_encryption = 'scram-sha-256';
\getenv sc_user POSTGRES_USER
\getenv sc_password POSTGRES_PASSWORD
ALTER ROLE :"sc_user" LOGIN PASSWORD :'sc_password';
SQL
then
    fail local_administrator_password_repair_failed
fi
verify || fail database_login_still_fails
log password_updated
