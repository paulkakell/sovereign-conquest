# syntax=docker/dockerfile:1

# Root all-in-one image for GHCR and generic repository-root Docker builds.
# The resulting container serves both the Go API and the bundled web UI on :8080.

FROM golang:1.22-alpine AS build
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

ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=${GOSUMDB}
ENV GOPRIVATE=${GOPRIVATE}
ENV GONOSUMDB=${GONOSUMDB}
ENV SC_BUILD_DNS=${SC_BUILD_DNS}
ENV HTTP_PROXY=${HTTP_PROXY}
ENV HTTPS_PROXY=${HTTPS_PROXY}
ENV NO_PROXY=${NO_PROXY}
ENV http_proxy=${http_proxy}
ENV https_proxy=${https_proxy}
ENV no_proxy=${no_proxy}
ENV SC_USE_VENDOR=${SC_USE_VENDOR}

COPY server/ /src/server/
WORKDIR /src/server

RUN chmod +x ./scripts/build_api.sh       && (./scripts/build_api.sh > /tmp/sc-build.log 2>&1 || {         echo >&2 "ERROR: build_api.sh failed; showing last 200 lines of /tmp/sc-build.log";         tail -n 200 /tmp/sc-build.log >&2 || true;         exit 1;       })       && cat /tmp/sc-build.log

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

COPY server/certs/ /usr/local/share/ca-certificates/
RUN update-ca-certificates

COPY --from=build /out/sovereign-api /app/sovereign-api
COPY web/static/ /app/web/

ENV HTTP_ADDR=:8080
ENV WEB_ROOT=/app/web

EXPOSE 8080
CMD ["/app/sovereign-api"]
