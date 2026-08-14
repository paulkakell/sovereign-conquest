# syntax=docker/dockerfile:1.7

FROM golang:1.26.6-alpine3.24 AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates git
COPY server/certs/ /usr/local/share/ca-certificates/
RUN update-ca-certificates

ARG GOPROXY=https://proxy.golang.org|direct
ARG GOSUMDB=sum.golang.org
ARG GOPRIVATE=
ARG GONOSUMDB=
ARG SC_BUILD_DNS=
ARG HTTP_PROXY=
ARG HTTPS_PROXY=
ARG NO_PROXY=
ARG http_proxy=
ARG https_proxy=
ARG no_proxy=
ARG SC_USE_VENDOR=1

ENV GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB} \
    GOPRIVATE=${GOPRIVATE} \
    GONOSUMDB=${GONOSUMDB} \
    SC_BUILD_DNS=${SC_BUILD_DNS} \
    HTTP_PROXY=${HTTP_PROXY} \
    HTTPS_PROXY=${HTTPS_PROXY} \
    NO_PROXY=${NO_PROXY} \
    http_proxy=${http_proxy} \
    https_proxy=${https_proxy} \
    no_proxy=${no_proxy} \
    SC_USE_VENDOR=${SC_USE_VENDOR}

COPY server/ /src/server/
WORKDIR /src/server
RUN chmod +x ./scripts/build_api.sh \
    && (./scripts/build_api.sh > /tmp/sc-build.log 2>&1 || { \
      echo >&2 "ERROR: API build failed; showing the last 200 lines"; \
      tail -n 200 /tmp/sc-build.log >&2 || true; \
      exit 1; \
    }) \
    && cat /tmp/sc-build.log

FROM alpine:3.24
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S -g 10001 sovereign \
    && adduser -S -D -H -u 10001 -G sovereign sovereign

COPY server/certs/ /usr/local/share/ca-certificates/
RUN update-ca-certificates
COPY --from=build --chown=10001:10001 /out/sovereign-api /app/sovereign-api
COPY --chown=10001:10001 web/static/ /app/web/

ENV APP_ENV=production \
    HTTP_ADDR=:8080 \
    WEB_ROOT=/app/web \
    TRUST_PROXY_HEADERS=false

USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/api/livez >/dev/null || exit 1
CMD ["/app/sovereign-api"]
