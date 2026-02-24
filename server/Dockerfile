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

RUN echo "Go module settings: GOPROXY=${GOPROXY} GOSUMDB=${GOSUMDB} GOPRIVATE=${GOPRIVATE} GONOSUMDB=${GONOSUMDB} SC_USE_VENDOR=${SC_USE_VENDOR}"

COPY go.mod ./
# NOTE: `go mod download` does not accept `-mod=...` (that's a flag for build/test).
# If you vendor dependencies, set SC_USE_VENDOR=1 to skip module downloads during image build.
RUN if [ "${SC_USE_VENDOR}" != "1" ]; then go mod download; fi

COPY . .

# If the build is configured to use vendoring, fail fast with a clear message
# when the vendor directory is missing.
RUN if [ "${SC_USE_VENDOR}" = "1" ] && [ ! -d vendor ]; then \
      echo "ERROR: SC_USE_VENDOR=1 but ./server/vendor is missing. Run 'go mod vendor' in ./server (with network access) and rebuild." >&2; \
      exit 1; \
    fi

RUN mkdir -p /out
RUN MODFLAG="-mod=mod"; \
    if [ "${SC_USE_VENDOR}" = "1" ]; then MODFLAG="-mod=vendor"; fi; \
    CGO_ENABLED=0 GOOS=linux go build ${MODFLAG} -trimpath -buildvcs=false -o /out/sovereign-api ./cmd/api

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
