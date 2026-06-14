# SignalPlane Installation Guide

This guide explains how to install and run the Silver developer preview of SignalPlane.

## What You Are Installing

SignalPlane currently ships as a single Go binary that includes:

- HTTP API server.
- Embedded web dashboard.
- Metrics, logs, traces, host, service, alert, incident, uptime, and token APIs.
- File-backed persistence using an atomic JSON snapshot for quick local runs.
- Dependency health checks for the full local Silver platform stack.

The Podman Compose stack also includes:

- PostgreSQL for control-plane metadata.
- ClickHouse for telemetry-scale signal storage.
- OpenTelemetry Collector for OTLP intake ports.
- Mailpit for email notification testing.

The default local URL is:

```text
http://127.0.0.1:4318
```

## Requirements

For local source builds:

- Go 1.26 or newer.
- macOS, Linux, or another environment supported by Go.
- `make` for convenience commands.

For containerized runs:

- Podman 5 or newer.
- Podman Compose support through `podman compose`.
- On macOS, a running Podman machine.

Optional for examples:

- Node.js for the Node test app.
- Python 3 for Python test apps.
- C compiler for the C host probe.

## Option 1: Run From Source

Clone the repository:

```bash
git clone https://github.com/chaktihor/signalplane.git
cd signalplane
```

Run SignalPlane:

```bash
make run
```

Open:

```text
http://127.0.0.1:4318
```

Default local bootstrap token:

```text
dev-token
```

## Option 2: Build The Binary

```bash
make build
./bin/signalplane
```

The binary listens on:

```text
127.0.0.1:4318
```

Override the bind address:

```bash
SIGNALPLANE_ADDR=0.0.0.0:4318 ./bin/signalplane
```

## Option 3: Run The Full Local Stack With Podman

```bash
make stack-up
```

Open:

```text
http://127.0.0.1:4318
```

Podman Compose starts SignalPlane, PostgreSQL, ClickHouse, OpenTelemetry Collector, and Mailpit.

Useful local URLs:

| Service | URL |
|---|---|
| SignalPlane | `http://127.0.0.1:4318` |
| ClickHouse HTTP | `http://127.0.0.1:8123` |
| OTLP gRPC | `127.0.0.1:4317` |
| OTLP HTTP | `127.0.0.1:4319` |
| Mailpit | `http://127.0.0.1:8025` |

Podman Compose persists data in named volumes:

```text
postgres-data
clickhouse-data
signalplane-data
```

Stop the stack:

```bash
make stack-down
```

Remove persisted Compose data:

```bash
make stack-reset
```

On macOS, initialize and start the Podman VM first if needed:

```bash
podman machine init
podman machine start
```

If you need to use Docker-compatible Compose instead, override the command:

```bash
CONTAINER_COMPOSE="docker compose" make stack-up
```

## Configuration

SignalPlane is configured with environment variables.

| Variable | Default | Purpose |
|---|---|---|
| `SIGNALPLANE_ADDR` | `127.0.0.1:4318` | Address and port for the HTTP server |
| `SIGNALPLANE_INGEST_TOKEN` | `dev-token` | Local bootstrap/admin token |
| `SIGNALPLANE_DATA_PATH` | `data/signalplane.json` | File-backed persistence path |
| `SIGNALPLANE_SEED_DEMO_DATA` | `true` | Seed demo services, metrics, logs, traces, and uptime monitor |
| `SIGNALPLANE_STORE_BACKEND` | `json` | Runtime store backend; PostgreSQL implementation is the next Silver-hardening slice |
| `SIGNALPLANE_POSTGRES_ADDR` | empty | Optional dependency health check target |
| `SIGNALPLANE_CLICKHOUSE_HTTP_URL` | empty | Optional ClickHouse health check URL |
| `SIGNALPLANE_OTEL_GRPC_ADDR` | empty | Optional OTLP gRPC dependency health check target |
| `SIGNALPLANE_OTEL_HTTP_ADDR` | empty | Optional OTLP HTTP dependency health check target |
| `SIGNALPLANE_SMTP_ADDR` | empty | Optional SMTP dependency health check target |
| `SIGNALPLANE_MAILPIT_URL` | empty | Optional Mailpit web health check URL |
| `SIGNALPLANE_READ_TIMEOUT_SECONDS` | `5` | HTTP read timeout |
| `SIGNALPLANE_WRITE_TIMEOUT_SECONDS` | `10` | HTTP write timeout |
| `SIGNALPLANE_IDLE_TIMEOUT_SECONDS` | `60` | HTTP idle timeout |

Example:

```bash
SIGNALPLANE_ADDR=0.0.0.0:4318 \
SIGNALPLANE_INGEST_TOKEN=change-me \
SIGNALPLANE_DATA_PATH=/var/lib/signalplane/signalplane.json \
./bin/signalplane
```

## Data Persistence

Silver persists data to an atomic JSON snapshot file.

Default local path:

```text
data/signalplane.json
```

This file stores:

- Organization and environment metadata.
- API tokens.
- Services.
- Hosts.
- Metrics.
- Logs.
- Traces.
- Alerts.
- Incidents.
- Uptime monitors.
- Audit events.

The full local stack now provisions PostgreSQL and ClickHouse schemas. The current runtime still uses JSON persistence until the next Silver-hardening slice wires the store interface into those databases.

## Verify Installation

Health check:

```bash
curl http://127.0.0.1:4318/healthz
```

Expected response:

```json
{
  "service": "signalplane",
  "status": "ok",
  "timestamp": "..."
}
```

Bootstrap API:

```bash
curl http://127.0.0.1:4318/api/bootstrap
```

You should see counts for services, hosts, metrics, logs, traces, alerts, tokens, incidents, and uptime monitors.

Dependency health API:

```bash
curl http://127.0.0.1:4318/api/system/dependencies
```

## Next Step

After installation, read:

- [User Guide](USER_GUIDE.md)
- [Telemetry Guide](TELEMETRY_GUIDE.md)
- [API Reference](API_REFERENCE.md)
