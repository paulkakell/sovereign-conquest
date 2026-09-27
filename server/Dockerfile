# syntax=docker/dockerfile:1.7

FROM golang:1.27.1-alpine3.24 AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates git tzdata \
  && mkdir -p /out/runtime-tmp
COPY certs/ /usr/local/share/ca-certificates/
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

ENV GOPROXY=${GOPROXY} GOSUMDB=${GOSUMDB} GOPRIVATE=${GOPRIVATE} GONOSUMDB=${GONOSUMDB}
ENV HTTP_PROXY=${HTTP_PROXY} HTTPS_PROXY=${HTTPS_PROXY} NO_PROXY=${NO_PROXY}
ENV http_proxy=${http_proxy} https_proxy=${https_proxy} no_proxy=${no_proxy}
ENV SC_BUILD_DNS=${SC_BUILD_DNS} SC_USE_VENDOR=${SC_USE_VENDOR}

COPY . .
RUN chmod +x ./scripts/build_api.sh \
  && (./scripts/build_api.sh > /tmp/sc-build.log 2>&1 || { tail -n 200 /tmp/sc-build.log >&2 || true; exit 1; }) \
  && cat /tmp/sc-build.log

# Keep OS packages and build tools out of the statically linked API runtime.
FROM scratch
WORKDIR /app
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /usr/share/zoneinfo/ /usr/share/zoneinfo/
COPY --from=build --chmod=1777 /out/runtime-tmp/ /tmp/
COPY --from=build --chown=10001:10001 /out/sovereign-api /app/sovereign-api

ENV APP_ENV=production HTTP_ADDR=:8080
USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/app/sovereign-api", "healthcheck"]
CMD ["/app/sovereign-api"]
