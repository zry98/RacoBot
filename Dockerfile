# syntax=docker/dockerfile:1

ARG GO_VERSION="1.26" \
    DEBIAN_VERSION="bookworm"

# Stage 1: build
FROM --platform=linux/amd64 golang:${GO_VERSION}-${DEBIAN_VERSION} AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO is required by go-lolhtml (bundled static liblolhtml for linux/amd64).
# -tags timetzdata embeds the Europe/Madrid zoneinfo so the binary is self-contained.
# -trimpath and -ldflags "-s -w" produce a smaller, reproducible binary.
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build \
      -tags timetzdata \
      -trimpath \
      -ldflags="-s -w" \
      -o /RacoBot .

# Stage 2: production
# using debian-slim (matches the build stage's glibc) so a shell-based HEALTHCHECK can run
FROM --platform=linux/amd64 debian:${DEBIAN_VERSION}-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
         ca-certificates \
         curl \
    && rm -rf /var/lib/apt/lists/*

# run as a non-root user
RUN useradd --system --no-create-home --uid 65532 racobot

WORKDIR /app
COPY --from=build /RacoBot /app/RacoBot

USER racobot
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -fsS http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/RacoBot", "-config", "/app/config.toml"]
