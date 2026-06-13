# SignalPlane Installation Guide

This guide explains how to install and run the Silver developer preview of SignalPlane.

## What You Are Installing

SignalPlane currently ships as a single Go binary that includes:

- HTTP API server.
- Embedded web dashboard.
- Metrics, logs, traces, host, service, alert, incident, uptime, and token APIs.
- File-backed persistence using an atomic JSON snapshot.

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

- Docker.
- Docker Compose.

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

## Option 3: Run With Docker Compose

```bash
docker compose up --build
```

Open:

```text
http://127.0.0.1:4318
```

Docker Compose persists data in the named volume:

```text
signalplane-data
```

Stop SignalPlane:

```bash
docker compose down
```

Remove persisted Compose data:

```bash
docker compose down -v
```

## Configuration

SignalPlane is configured with environment variables.

| Variable | Default | Purpose |
|---|---|---|
| `SIGNALPLANE_ADDR` | `127.0.0.1:4318` | Address and port for the HTTP server |
| `SIGNALPLANE_INGEST_TOKEN` | `dev-token` | Local bootstrap/admin token |
| `SIGNALPLANE_DATA_PATH` | `data/signalplane.json` | File-backed persistence path |
| `SIGNALPLANE_SEED_DEMO_DATA` | `true` | Seed demo services, metrics, logs, traces, and uptime monitor |
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

This is good enough for the Silver developer preview. Future tiers should move to a proper database and telemetry storage backend.

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

## Next Step

After installation, read:

- [User Guide](USER_GUIDE.md)
- [Telemetry Guide](TELEMETRY_GUIDE.md)
- [API Reference](API_REFERENCE.md)

