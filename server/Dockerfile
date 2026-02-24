# Build stage
FROM golang:1.22-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates git

# Optional: trust additional CAs during build (for corporate proxies / private module proxies).
#
# Add one or more PEM-encoded `.crt` files under `./server/certs/`.
COPY certs/ /usr/local/share/ca-certificates/
RUN update-ca-certificates

# Build-time configuration for restricted environments.
#
# Examples:
#   docker build --build-arg GOPROXY=direct \
#     --build-arg GOSUMDB=off \
#     -t sovereign-api ./server
#
# To build fully offline (after vendoring deps):
#   go mod vendor
#   docker build --build-arg SC_USE_VENDOR=1 -t sovereign-api ./server
# Default uses the "|" separator so Go falls back to direct VCS fetches if the
# proxy is unreachable (comma only falls back on 404/410).
ARG GOPROXY=https://proxy.golang.org|direct
ARG GOSUMDB=sum.golang.org
ARG GOPRIVATE=
ARG GONOSUMDB=
# Optional: override /etc/resolv.conf inside the build stage to work around
# builder sandboxes that inject a broken/isolated DNS server.
# Example: SC_BUILD_DNS="1.1.1.1 8.8.8.8"
ARG SC_BUILD_DNS=
# Standard Docker proxy args (optional) for restricted networks.
ARG HTTP_PROXY=
ARG HTTPS_PROXY=
ARG NO_PROXY=
ARG http_proxy=
ARG https_proxy=
ARG no_proxy=
ARG SC_USE_VENDOR=0
ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=${GOSUMDB}
ENV GOPRIVATE=${GOPRIVATE}
ENV GONOSUMDB=${GONOSUMDB}
ENV HTTP_PROXY=${HTTP_PROXY}
ENV HTTPS_PROXY=${HTTPS_PROXY}
ENV NO_PROXY=${NO_PROXY}
ENV http_proxy=${http_proxy}
ENV https_proxy=${https_proxy}
ENV no_proxy=${no_proxy}

RUN echo "Go module settings: GOPROXY=${GOPROXY} GOSUMDB=${GOSUMDB} GOPRIVATE=${GOPRIVATE} GONOSUMDB=${GONOSUMDB} SC_USE_VENDOR=${SC_USE_VENDOR} SC_BUILD_DNS=${SC_BUILD_DNS}"

# Optional: override build-time DNS inside the build container.
RUN if [ -n "${SC_BUILD_DNS}" ]; then \
      echo "Overriding build-time DNS: ${SC_BUILD_DNS}"; \
      rm -f /etc/resolv.conf || true; \
      for ns in ${SC_BUILD_DNS}; do echo "nameserver ${ns}" >> /etc/resolv.conf; done; \
      echo "Build-time DNS (/etc/resolv.conf):"; \
      cat /etc/resolv.conf || true; \
    fi

COPY . .

# Download deps (or require vendor).
# NOTE: vendoring is auto-detected if vendor/modules.txt is present (this helps
# stack deploy UIs where env vars don't propagate to compose build args).
RUN USE_VENDOR=0; \
    if [ "${SC_USE_VENDOR}" = "1" ] || [ -f vendor/modules.txt ]; then USE_VENDOR=1; fi; \
    if [ "${USE_VENDOR}" = "1" ] && [ ! -f vendor/modules.txt ]; then \
      echo >&2 "ERROR: Vendored build requested but ./server/vendor/modules.txt is missing."; \
      echo >&2 "Run 'go mod vendor' under ./server and rebuild."; \
      exit 1; \
    fi; \
    echo "USE_VENDOR=${USE_VENDOR}"; \
    if [ "${USE_VENDOR}" != "1" ]; then \
      echo "Downloading Go modules (requires outbound DNS + HTTPS to GOPROXY/GOSUMDB)..."; \
      if ! go mod download 2> /tmp/go-mod-download.err; then \
        echo >&2 "ERROR: go mod download failed."; \
        cat /tmp/go-mod-download.err >&2 || true; \
        echo >&2 "Most common causes in stack/CI builders:"; \
        echo >&2 "  - No outbound internet/DNS from the build sandbox"; \
        echo >&2 "  - proxy.golang.org and/or sum.golang.org blocked"; \
        echo >&2 "  - Corporate proxy TLS interception missing CA (add .crt files under server/certs/)"; \
        echo >&2 "Fix options:"; \
        echo >&2 "  - Set SC_BUILD_NETWORK=host (compose build.network)"; \
        echo >&2 "  - Set GOPROXY=direct and/or GOSUMDB=off"; \
        echo >&2 "  - Fully-offline: run 'go mod vendor' under ./server and rebuild (SC_USE_VENDOR=1 or keep vendor/modules.txt)"; \
        # Auto-retry when the checksum DB is blocked.
        if [ "${GOSUMDB}" != "off" ] && grep -qE '(sum\\.golang\\.org|checksum database)' /tmp/go-mod-download.err; then \
          echo >&2 "Retrying module download with GOSUMDB=off (checksum DB disabled)..."; \
          if ! env GOSUMDB=off go mod download 2> /tmp/go-mod-download.err2; then \
            echo >&2 "ERROR: retry (GOSUMDB=off) failed."; \
            cat /tmp/go-mod-download.err2 >&2 || true; \
            echo >&2 "Build-time DNS (/etc/resolv.conf):"; \
            cat /etc/resolv.conf >&2 || true; \
            exit 1; \
          fi; \
          echo 'export GOSUMDB=off' > /tmp/sc-go-env; \
        else \
          echo >&2 "Build-time DNS (/etc/resolv.conf):"; \
          cat /etc/resolv.conf >&2 || true; \
          exit 1; \
        fi; \
      fi; \
    fi

RUN mkdir -p /out
RUN MODFLAG="-mod=mod"; \
    if [ "${SC_USE_VENDOR}" = "1" ] || [ -f vendor/modules.txt ]; then MODFLAG="-mod=vendor"; fi; \
    if [ -f /tmp/sc-go-env ]; then . /tmp/sc-go-env; fi; export GOSUMDB || true; \
    echo "Building sovereign-api (${MODFLAG})..."; \
    CGO_ENABLED=0 GOOS=linux go build ${MODFLAG} -trimpath -buildvcs=false -o /out/sovereign-api ./cmd/api 2> /tmp/go-build.err || { \
      echo >&2 "ERROR: go build failed."; \
      cat /tmp/go-build.err >&2 || true; \
      if [ "${GOSUMDB}" != "off" ] && grep -qE '(sum\\.golang\\.org|checksum database)' /tmp/go-build.err; then \
        echo >&2 "Retrying go build with GOSUMDB=off (checksum DB disabled)..."; \
        CGO_ENABLED=0 GOOS=linux env GOSUMDB=off go build ${MODFLAG} -trimpath -buildvcs=false -o /out/sovereign-api ./cmd/api || true; \
      fi; \
      echo >&2 "If the error mentions downloading modules, fix build-time network/DNS or vendor deps."; \
      echo >&2 "Vendoring workflow:"; \
      echo >&2 "  1) (with network) cd ./server && go mod vendor"; \
      echo >&2 "  2) rebuild with SC_USE_VENDOR=1 (or just keep vendor/modules.txt in the build context)"; \
      echo >&2 "Build-time DNS (/etc/resolv.conf):"; \
      cat /etc/resolv.conf >&2 || true; \
      exit 1; \
    }

# Runtime stage
FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

# Optional: trust additional CAs at runtime too (useful if the API calls internal HTTPS endpoints).
COPY certs/ /usr/local/share/ca-certificates/
RUN update-ca-certificates

COPY --from=build /out/sovereign-api /app/sovereign-api

EXPOSE 8080
CMD ["/app/sovereign-api"]
