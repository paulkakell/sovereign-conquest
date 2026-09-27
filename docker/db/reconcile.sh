#!/usr/bin/env bash
set -Eeuo pipefail
: "${SC_DB_ADMIN:?Private startup administrator is required}"
: "${SC_DB_SOCKET:?Private startup socket is required}"
: "${POSTGRES_USER:?}" "${POSTGRES_PASSWORD:?}" "${POSTGRES_DB:?}"

export PGHOST=$SC_DB_SOCKET PGPORT=5432 PGUSER=$SC_DB_ADMIN PGDATABASE=postgres
export PGCONNECT_TIMEOUT=5 PGPASSFILE=/dev/null
unset PGHOSTADDR PGPASSWORD PGSERVICE PGSERVICEFILE PGOPTIONS

# psql quotes identifiers and literals; SQL is never assembled by the shell.
# New roles have no cluster-administration privileges. Existing privileges and
# object ownership stay intact. Each step is safe to repeat after interruption.
psql -X -w -q -v ON_ERROR_STOP=1 <<'SQL'
\getenv sc_user POSTGRES_USER
\getenv sc_password POSTGRES_PASSWORD
\getenv sc_db POSTGRES_DB
SET log_statement = 'none';
SET log_min_error_statement = 'panic';
SELECT format('CREATE ROLE %I LOGIN', :'sc_user')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = :'sc_user') \gexec
ALTER ROLE :"sc_user" LOGIN PASSWORD :'sc_password' VALID UNTIL 'infinity';
SELECT format('CREATE DATABASE %I OWNER %I', :'sc_db', :'sc_user')
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = :'sc_db') \gexec
GRANT CONNECT ON DATABASE :"sc_db" TO :"sc_user";
SQL
