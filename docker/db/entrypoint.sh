#!/usr/bin/env bash
set -Eeuo pipefail

# Reuse the official image's initialization and privilege-dropping helpers.
# shellcheck source=/dev/null
source /usr/local/bin/docker-entrypoint.sh

log() { printf '{"component":"db-startup","event":"%s"}\n' "$1"; }
fail() { log "$1" >&2; exit 1; }

[[ $# -gt 0 ]] || set -- postgres
[[ ${1:0:1} != '-' ]] || set -- postgres "$@"
if [[ $1 != postgres ]] || _pg_want_help "$@"; then
    exec /usr/local/bin/docker-entrypoint.sh "$@"
fi

docker_setup_env
[[ -n $POSTGRES_PASSWORD ]] || fail password_required
# Reject names PostgreSQL would silently truncate, or that cannot be represented
# safely in operational tooling. Passwords are not restricted or interpolated.
for name in "$POSTGRES_USER" "$POSTGRES_DB"; do
    [[ -n $name && $(LC_ALL=C printf '%s' "$name" | wc -c) -le 63 && $name != *[$'\r\n\t']* ]] || fail invalid_database_or_role_name
done
[[ $POSTGRES_DB != template0 && $POSTGRES_DB != template1 ]] || fail template_database_not_supported
[[ ${POSTGRES_HOST_AUTH_METHOD:-} != trust ]] || fail password_authentication_required

docker_create_db_directories
if [[ $(id -u) == 0 ]]; then
    exec gosu postgres bash "${BASH_SOURCE[0]}" "$@"
fi

ready=/var/run/postgresql/sc-db-ready
rm -f "$ready"
umask 077
work=$(mktemp -d /tmp/sc-db-startup.XXXXXXXX)
server_started=false
cleanup() {
    local status=$?
    trap - EXIT
    if [[ $server_started == true ]]; then
        pg_ctl -D "$PGDATA" -m fast -w -t 60 stop >/dev/null 2>&1 || true
    fi
    rm -rf "$work"
    [[ $status == 0 ]] || log startup_failed >&2
    exit "$status"
}
trap cleanup EXIT
trap 'exit 143' TERM
trap 'exit 130' INT

if [[ -z $DATABASE_ALREADY_EXISTS ]]; then
    log initializing_cluster
    docker_verify_minimum_env
    # Newer official images detect volumes mounted at an incompatible path.
    if declare -F docker_error_old_databases >/dev/null; then docker_error_old_databases; fi
    docker_init_database_dir
    pg_setup_hba_conf "$@"
    export PGPASSWORD=$POSTGRES_PASSWORD
    server_started=true
    docker_temp_server_start "$@"
    docker_setup_db
    docker_process_init_files /docker-entrypoint-initdb.d/*
    docker_temp_server_stop
    server_started=false
    unset PGPASSWORD
fi

# Discover the original bootstrap administrator even if POSTGRES_USER changed.
# Read its name while no server is running.
# No password, supplied identifier, or data-changing SQL enters this command.
if ! postgres --single -D "$PGDATA" -c logging_collector=off -c log_statement=none template1 \
    >"$work/discovery.log" 2>&1 <<SQL
COPY (SELECT encode(convert_to(rolname, 'UTF8'), 'hex') FROM pg_roles WHERE oid = 10 AND rolsuper) TO '$work/admin.hex';
COPY (SELECT rolcanlogin FROM pg_roles WHERE oid = 10 AND rolsuper) TO '$work/admin-login';
SQL
then
    fail administrator_discovery_failed
fi
[[ -s $work/admin.hex ]] || fail administrator_discovery_failed
hex=$(<"$work/admin.hex")
[[ $hex =~ ^([0-9a-f]{2})+$ ]] || fail administrator_discovery_failed
printf -v admin '%b' "$(printf '%s' "$hex" | sed 's/../\\x&/g')"
export SC_DB_ADMIN=$admin SC_DB_SOCKET=$work

# A NOLOGIN bootstrap role cannot connect even through a trusted private socket.
# Only enable it when it is the exact login the operator asked us to repair.
# Never enable an unrelated disabled administrator as a side effect.
if [[ $(<"$work/admin-login") == f ]]; then
    [[ $admin == "$POSTGRES_USER" ]] || fail administrator_login_disabled
    log enabling_configured_bootstrap_login
    postgres --single -D "$PGDATA" -c logging_collector=off -c log_statement=none template1 \
        >"$work/enable-admin.log" 2>&1 <<'SQL'
DO $$ DECLARE admin_name name; BEGIN SELECT rolname INTO STRICT admin_name FROM pg_roles WHERE oid = 10 AND rolsuper; EXECUTE format('ALTER ROLE %I LOGIN', admin_name); END $$;
SQL
fi

# Only the database OS user can access this socket directory. TCP is loopback
# only and always requires SCRAM, even when an old volume's HBA uses trust.
cat >"$work/pg_hba.conf" <<'HBA'
local all all trust
host all all 127.0.0.1/32 scram-sha-256
HBA
args=("${@:2}" -c listen_addresses=127.0.0.1 -p 5432
    -c "unix_socket_directories=$work" -c "hba_file=$work/pg_hba.conf"
    -c logging_collector=off -c log_statement=none -c log_min_error_statement=panic
    -c log_min_duration_statement=-1 -c log_min_duration_sample=-1
    -c log_statement_sample_rate=0 -c password_encryption=scram-sha-256)
server_started=true
pg_ctl -D "$PGDATA" -o "$(printf '%q ' "${args[@]}")" -w -t 60 start >"$work/server.log" 2>&1 || fail private_server_start_failed

log checking_credentials
if ! bash /opt/sc-db/check.sh --startup; then
    log reconciling_database_and_role
    # SQL/client errors may include supplied values. Keep them private and emit
    # a fixed diagnostic event instead of copying secrets into container logs.
    bash /opt/sc-db/reconcile.sh >"$work/reconcile.log" 2>&1 || fail reconciliation_failed
fi
bash /opt/sc-db/check.sh --startup || fail credential_verification_failed
pg_ctl -D "$PGDATA" -m fast -w -t 60 stop >/dev/null
server_started=false
# The official initdb defaults can trust localhost. Require SCRAM for health
# probes in the final server too. Include the existing HBA in place so its
# relative includes still resolve correctly; never rewrite the persistent file.
original_hba=$(postgres -D "$PGDATA" "${@:2}" -C hba_file)
[[ $original_hba == /* && $original_hba != *[$'\r\n"']* ]] || fail unsupported_hba_path
runtime_hba=/var/run/postgresql/sc-runtime-pg_hba.conf
{
    printf 'host all all 127.0.0.1/32 scram-sha-256\n'
    printf 'include "%s"\n' "$original_hba"
} >"$runtime_hba"
rm -rf "$work"
trap - EXIT INT TERM
unset SC_DB_ADMIN SC_DB_SOCKET
log credentials_verified
# Health checks cannot succeed against the temporary server above.
touch "$ready"
unset "${!POSTGRES_@}"
exec postgres "${@:2}" -c "hba_file=$runtime_hba"
