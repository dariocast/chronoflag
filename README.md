# Chronograph

Chronograph is a zero-configuration, server-authoritative shared stopwatch and countdown timer. Opening the site shows a ready stopwatch; the first action creates an anonymous instance with separate secret control and unlisted public-view links.

## Features

- Multiple independent stopwatches and countdown timers
- Shared Start, Pause, Resume, Reset, stopwatch Lap, labels, focus, and ten-second Undo
- Server timestamps, idempotent commands, deterministic concurrent control
- Live public board over Server-Sent Events
- JSON and zipped CSV export
- Free retention: 24 hours active plus 7 days read-only
- Installable brutalist PWA with honest offline behavior
- PostgreSQL durability and one-container Go/Svelte deployment

Manual commands are timed when the server receives them. Network latency is therefore part of the measurement; this is not certified sports-timing equipment.

## Local development

Requirements: Go 1.24+, Node 22+, npm 10+.

```sh
cd web && npm install && cd ..
make verify
make build
LISTEN_ADDR=:8080 ./chronograph
```

Without `DATABASE_URL`, the service uses volatile memory and logs a warning. For PostgreSQL:

```sh
export DATABASE_URL='postgres://chronograph:secret@localhost:5432/chronograph?sslmode=disable'
./chronograph
```

Migrations are embedded and applied idempotently on startup.

## Docker deployment

```sh
cp .env.example .env
# replace POSTGRES_PASSWORD in .env
docker compose up -d --build
./scripts/smoke.sh
```

Put an HTTPS reverse proxy such as Caddy in front of port 8080. Preserve streaming responses, disable proxy buffering for `text/event-stream`, enable HTTP/2, and synchronize host time with chrony/systemd-timesyncd. Back up the `chronograph-data` volume and regularly test restore. Capability URLs are credentials: exclude them from access logs and analytics.

## Configuration

| Variable | Purpose | Default |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection URL | memory store |
| `LISTEN_ADDR` | HTTP listen address | `:8080` |
| `POSTGRES_PASSWORD` | Compose database secret | required |
| `HTTP_PORT` | Published Compose port | `8080` |

## Verification

`make verify` runs Go tests with the race detector, frontend unit tests, Svelte/TypeScript diagnostics, and production builds. `npm run test:e2e` runs Playwright against an automatically started memory-backed binary. Set `TEST_DATABASE_URL` to include PostgreSQL adapter integration tests.

The `/healthz` endpoint is suitable for liveness checks. Operational metrics should be collected from container/runtime telemetry; application logs intentionally omit capability tokens and labels.
