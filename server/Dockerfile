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
COPY . .

# Build deps + compile.
#
# We run the build through a script so the Dockerfile stays small enough that
# some stack deploy/build UIs don't truncate away the actionable Go error output.
RUN chmod +x ./scripts/build_api.sh \
  && (./scripts/build_api.sh > /tmp/sc-build.log 2>&1 || { \
    echo >&2 "ERROR: build_api.sh failed; showing last 200 lines of /tmp/sc-build.log"; \
    tail -n 200 /tmp/sc-build.log >&2 || true; \
    exit 1; \
  }) \
  && cat /tmp/sc-build.log

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
