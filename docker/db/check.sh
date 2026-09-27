#!/usr/bin/env bash
set -Eeuo pipefail

if [[ ${1:-} != --startup && ! -f /var/run/postgresql/sc-db-ready ]]; then exit 1; fi
[[ -n ${POSTGRES_USER:-} && -n ${POSTGRES_PASSWORD:-} && -n ${POSTGRES_DB:-} ]] || exit 1

# Environment values avoid libpq interpreting a database name containing '='
# as a connection string. Never put the password in process arguments or logs.
export PGHOST=127.0.0.1 PGHOSTADDR=127.0.0.1 PGPORT=5432
export PGUSER=$POSTGRES_USER PGPASSWORD=$POSTGRES_PASSWORD PGDATABASE=$POSTGRES_DB
export PGCONNECT_TIMEOUT=2 PGSSLMODE=disable PGPASSFILE=/dev/null
unset PGSERVICE PGSERVICEFILE PGOPTIONS
result=$(psql -X -w -qAt -v ON_ERROR_STOP=1 -c 'SELECT 1' 2>/dev/null) || exit 1
[[ $result == 1 ]]
