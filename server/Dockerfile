# Build stage
FROM golang:1.22-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/sovereign-api ./cmd/api

# Runtime stage
FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /out/sovereign-api /app/sovereign-api

EXPOSE 8080
CMD ["/app/sovereign-api"]
