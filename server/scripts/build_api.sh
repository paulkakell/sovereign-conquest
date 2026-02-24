#!/bin/sh
# Build helper for the sovereign-api Docker image.
#
# This script exists so the Dockerfile can stay small enough that build UIs
# (Portainer/remote builders) don't truncate the actionable Go error output.

set -eu

log() {
  # Log to stderr so it remains visible even when stdout is buffered.
  printf '%s\n' "$*" >&2
}

redact_url_creds() {
  # Redact basic-auth credentials if a proxy URL is provided as:
  #   https://user:pass@proxy.example.com
  # Keep the scheme + host for diagnostics, but avoid leaking secrets in logs.
  # Note: this is best-effort and intentionally simple.
  printf '%s' "${1:-}" | sed -E 's#(://)[^/@]+@#\1***@#g'
}

sanitize_dns_list() {
  # Accept space-separated or comma-separated lists.
  # Also tolerate surrounding quotes that some .env parsers include.
  # Example accepted inputs:
  #   1.1.1.1 8.8.8.8
  #   "1.1.1.1 8.8.8.8"
  #   1.1.1.1,8.8.8.8
  printf '%s' "${1:-}" | tr ',' ' ' | tr -d '"' | tr -d "'"
}

apply_build_dns() {
  dns_list="$(sanitize_dns_list "${1:-}")"
  if [ -z "${dns_list}" ]; then
    return 0
  fi

  log "Overriding build-time DNS: ${dns_list}"
  rm -f /etc/resolv.conf || true
  for ns in ${dns_list}; do
    # Only write tokens that look like IPv4/IPv6 literals.
    case "${ns}" in
      *.*|*:*)
        echo "nameserver ${ns}" >> /etc/resolv.conf
        ;;
    esac
  done
  log "Build-time DNS (/etc/resolv.conf):"
  cat /etc/resolv.conf >&2 || true
}

is_sumdb_error() {
  # Match common checksum DB failure modes.
  grep -qE '(sum\.golang\.org|checksum database)' "${1}" 2>/dev/null
}

is_dns_error() {
  # Match common DNS failure modes across BusyBox, Go, and libc.
  grep -qiE 'no such host|temporary failure in name resolution|name resolution|lookup .*:|dial tcp: lookup|connection refused.*:53|i/o timeout.*:53|resolving timed out' "${1}" 2>/dev/null
}

try_go_mod_download() {
  err_file="$1"
  : > "${err_file}"
  if go mod download 2>"${err_file}"; then
    return 0
  fi
  return 1
}

try_go_build() {
  modflag="$1"
  err_file="$2"
  : > "${err_file}"
  if CGO_ENABLED=0 GOOS=linux go build "${modflag}" -trimpath -buildvcs=false -o /out/sovereign-api ./cmd/api 2>"${err_file}"; then
    return 0
  fi
  return 1
}

log "Go module settings: GOPROXY=$(redact_url_creds "${GOPROXY:-}") GOSUMDB=${GOSUMDB:-} GOPRIVATE=${GOPRIVATE:-} GONOSUMDB=${GONOSUMDB:-} SC_USE_VENDOR=${SC_USE_VENDOR:-0} SC_BUILD_DNS=${SC_BUILD_DNS:-}"

# Optional: override build-time DNS inside the build container.
if [ -n "${SC_BUILD_DNS:-}" ]; then
  apply_build_dns "${SC_BUILD_DNS}"
fi

use_vendor=0
if [ "${SC_USE_VENDOR:-0}" = "1" ] || [ -f vendor/modules.txt ]; then
  use_vendor=1
fi

if [ "${use_vendor}" = "1" ] && [ ! -f vendor/modules.txt ]; then
  log "ERROR: Vendored build requested but vendor/modules.txt is missing."
  log "Run 'cd ./server && go mod vendor' (with network) and rebuild."
  exit 1
fi

log "USE_VENDOR=${use_vendor}"

if [ "${use_vendor}" != "1" ]; then
  log "Downloading Go modules (requires outbound DNS + HTTPS to GOPROXY/GOSUMDB)..."

  if ! try_go_mod_download /tmp/go-mod-download.err; then
    log "ERROR: go mod download failed."
    cat /tmp/go-mod-download.err >&2 || true

    # Auto-retry when the checksum DB is blocked.
    if [ "${GOSUMDB:-}" != "off" ] && is_sumdb_error /tmp/go-mod-download.err; then
      log "Retrying module download with GOSUMDB=off (checksum DB disabled)..."
      if ! env GOSUMDB=off go mod download 2>/tmp/go-mod-download.err2; then
        log "ERROR: retry (GOSUMDB=off) failed."
        cat /tmp/go-mod-download.err2 >&2 || true
        log "Build-time DNS (/etc/resolv.conf):"
        cat /etc/resolv.conf >&2 || true
        exit 1
      fi
      # Persist the environment override for subsequent build steps.
      echo 'export GOSUMDB=off' > /tmp/sc-go-env
    # Auto-retry once with a public DNS fallback if the error looks DNS-related
    # and the user did not explicitly provide SC_BUILD_DNS.
    elif [ -z "${SC_BUILD_DNS:-}" ] && is_dns_error /tmp/go-mod-download.err; then
      log "Detected DNS-related failure and SC_BUILD_DNS is unset. Retrying with a public DNS fallback (1.1.1.1 8.8.8.8)..."
      apply_build_dns "1.1.1.1 8.8.8.8"
      if ! try_go_mod_download /tmp/go-mod-download.err3; then
        log "ERROR: retry (public DNS fallback) failed."
        cat /tmp/go-mod-download.err3 >&2 || true
        log "Fix options:"
        log "  - Set SC_BUILD_NETWORK=host (compose build.network)"
        log "  - Set SC_BUILD_DNS to a reachable DNS server list (space or comma separated)"
        log "  - Set GOPROXY=direct and/or GOSUMDB=off"
        log "  - Fully-offline: run 'cd ./server && go mod vendor' and rebuild (ensure vendor/modules.txt is in the build context)"
        log "Build-time DNS (/etc/resolv.conf):"
        cat /etc/resolv.conf >&2 || true
        exit 1
      fi
    else
      log "Fix options:"
      log "  - Set SC_BUILD_NETWORK=host (compose build.network)"
      log "  - Set SC_BUILD_DNS to a reachable DNS server list (space or comma separated)"
      log "  - Set GOPROXY=direct and/or GOSUMDB=off"
      log "  - Fully-offline: run 'cd ./server && go mod vendor' and rebuild (ensure vendor/modules.txt is in the build context)"
      log "Build-time DNS (/etc/resolv.conf):"
      cat /etc/resolv.conf >&2 || true
      exit 1
    fi
  fi
fi

mkdir -p /out

modflag="-mod=mod"
if [ "${use_vendor}" = "1" ] || [ -f vendor/modules.txt ]; then
  modflag="-mod=vendor"
fi

if [ -f /tmp/sc-go-env ]; then
  # shellcheck disable=SC1091
  . /tmp/sc-go-env
fi

log "Building sovereign-api (${modflag})..."

if ! try_go_build "${modflag}" /tmp/go-build.err; then
  log "ERROR: go build failed."
  cat /tmp/go-build.err >&2 || true

  # Retry with checksum DB disabled if this looks like a sumdb problem.
  if [ "${GOSUMDB:-}" != "off" ] && is_sumdb_error /tmp/go-build.err; then
    log "Retrying go build with GOSUMDB=off (checksum DB disabled)..."
    if CGO_ENABLED=0 GOOS=linux env GOSUMDB=off go build "${modflag}" -trimpath -buildvcs=false -o /out/sovereign-api ./cmd/api; then
      exit 0
    fi
  fi

  # Retry once with DNS fallback if it looks DNS-related and the user didn't set SC_BUILD_DNS.
  if [ -z "${SC_BUILD_DNS:-}" ] && is_dns_error /tmp/go-build.err; then
    log "Detected DNS-related build failure and SC_BUILD_DNS is unset. Retrying with a public DNS fallback (1.1.1.1 8.8.8.8)..."
    apply_build_dns "1.1.1.1 8.8.8.8"
    if CGO_ENABLED=0 GOOS=linux go build "${modflag}" -trimpath -buildvcs=false -o /out/sovereign-api ./cmd/api; then
      exit 0
    fi
  fi

  log "If the error mentions downloading modules, fix build-time network/DNS or vendor deps."
  log "Vendoring workflow:"
  log "  1) (with network) cd ./server && go mod vendor"
  log "  2) rebuild (vendoring is auto-detected when vendor/modules.txt is present)"
  log "Build-time DNS (/etc/resolv.conf):"
  cat /etc/resolv.conf >&2 || true
  exit 1
fi
