# Racó Bot

[![Go](https://github.com/zry98/RacoBot/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/zry98/RacoBot/actions/workflows/go.yml)
[![CodeQL](https://github.com/zry98/RacoBot/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/zry98/RacoBot/actions/workflows/codeql.yml)

A Telegram Bot for forwarding notices from [*El Racó de la FIB*](https://raco.fib.upc.edu/).

(The old single-user version for running on Cloudflare Workers® has been moved to the branch [worker](https://github.com/zry98/RacoBot/tree/worker))

## Disclaimer

This project and the deployed Telegram Bot [@FIBRacoBot](https://t.me/FIBRacoBot) are **unofficial** and not associated with *El Racó* or *La FIB (Facultat d'Informàtica de Barcelona)*.

## Prerequisites

- **Go 1.26+**.
- **A C toolchain with CGO enabled**: The notice HTML is rewritten with [`go-lolhtml`](https://github.com/coolspring8/go-lolhtml), which binds to Cloudflare's native [lol-html](https://github.com/cloudflare/lol-html) library, so `CGO_ENABLED=1` and a C compiler are required. The native library targets **Linux**, see [Building](#building) for cross-compiling from macOS.
- **A Redis-compatible server as the database**: [Redis](https://redis.io/) 6+ or [Valkey](https://valkey.io/). It stores users, OAuth tokens, login sessions, and cached subject codes.
- **A Telegram bot**: create one via [@BotFather](https://t.me/BotFather) to get its token.
- **FIB API OAuth credentials**: register an application at the [FIB API dashboard](https://api.fib.upc.edu/v2/) to get the client ID/secret and set the redirect URI.

## Configuration

The bot reads a [TOML](https://toml.io/) config file (default `./config.toml`, override with `-config`). Copy the example and fill it in:

```bash
cp example.config.toml config.toml
```

Key settings:

| Section        | Key                                             | Description                                                                                     |
|----------------|-------------------------------------------------|-------------------------------------------------------------------------------------------------|
| _(top level)_  | `host`, `port`                                  | Address the HTTP server listens on.                                                             |
| `[tls]`        | `certificate_path`, `private_key_path`          | Optional. Omit the whole section to serve plain HTTP (e.g. behind a reverse proxy that terminates TLS). |
| `[redis]`      | `address`                                       | A `host:port` or a Unix socket path (e.g. `/var/run/redis/redis-server.sock`). Also `username`, `password`, `db`. |
| `[fib_api]`    | `oauth_client_id`, `oauth_client_secret`, `oauth_redirect_uri`, `public_client_id` | FIB API OAuth application credentials.                       |
| `[telegram_bot]` | `token`                                       | Bot token from BotFather.                                                                       |
| `[telegram_bot]` | `webhook_url`                                 | If set, updates are received via this webhook; if empty, the bot falls back to long polling.    |
| `[telegram_bot]` | `admin_uids`                                  | Telegram user IDs allowed to run admin commands (e.g. `/announce`).                             |
| `[jobs]`       | `push_new_notices_cron`, `cache_subject_codes_cron` | Cron expressions for the scheduled jobs (Europe/Madrid timezone).                           |

## Building

On **Linux** (native), with a C compiler installed:

```bash
CGO_ENABLED=1 go build -o RacoBot
```

Cross-compiling from **macOS** to `linux/amd64` requires a Linux cross toolchain (e.g. install one with `brew install messense/macos-cross-toolchains/x86_64-unknown-linux-gnu`), then:

```bash
CGO_ENABLED=1 \
CC=x86_64-unknown-linux-gnu-gcc \
CXX=x86_64-unknown-linux-gnu-g++ \
GOOS=linux GOARCH=amd64 \
go build -o RacoBot
```

## Running

### Manually

1. Start a Redis or Valkey server and make sure it's reachable at the `[redis]` `address` in your config.

2. Run the bot, pointing it at your config file:

```bash
./RacoBot -config ./config.toml
```

### With Docker Compose

The provided [`Dockerfile`](Dockerfile) and [`docker-compose.yml`](docker-compose.yml) run the bot together with a persistent Valkey instance. The image is **`linux/amd64` only** (see [Prerequisites](#prerequisites)); on other architectures Docker will build/run it under QEMU emulation.

Prepare a `config.toml` suited to the container network:

```toml
host = "0.0.0.0"   # must bind all interfaces so the mapped port is reachable
port = 8080        # plain HTTP; terminate TLS at a reverse proxy / Cloudflare and omit the [tls] section

[redis]
address = "db:6379"

[log]
level = "info"     # leave `path` unset so logs go to stdout
```

Then build and start everything:

```bash
docker compose up -d --build
docker compose logs -f racobot
```

Database is persisted in the `db-data` volume, so user tokens survive restarts.

Stop with `docker compose down` (add `-v` to also wipe the datastore).

### Health check

The server exposes a `GET /healthz` endpoint that reports Redis connectivity and basic runtime metrics (returns HTTP 503 if Redis is unreachable).

The Docker image also ships a built-in `HEALTHCHECK` that probes this endpoint (assuming the config's `port = 8080`), so `docker compose ps` reflects the container's health.
