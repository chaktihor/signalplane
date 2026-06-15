# SignalPlane Installation Guide

This guide explains how to install and run the Silver stack of SignalPlane.

## What You Are Installing

SignalPlane currently ships as a single Go binary that includes:

- HTTP API server.
- Embedded web dashboard.
- Metrics, logs, traces, host, service, alert, incident, uptime, and token APIs.
- JSON and OTLP HTTP JSON/protobuf ingestion.
- JSON snapshot persistence for quick local runs and PostgreSQL-backed runtime persistence in the platform stack.
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

For Kubernetes production installs:

- Kubernetes 1.29+ or OpenShift 4.14+.
- Helm 3.
- HA PostgreSQL 16+.
- HA ClickHouse 24.8+.
- Private registry and image pull secret.
- Ingress controller and TLS certificate.
- Storage class for ReadWriteOnce replay PVCs.

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

For source runs, set an ingest token and either a bootstrap owner user or admin token:

```bash
SIGNALPLANE_INGEST_TOKEN=dev-token \
SIGNALPLANE_BOOTSTRAP_ADMIN_TOKEN=dev-admin-token \
SIGNALPLANE_BOOTSTRAP_USER_EMAIL=admin@signalplane.local \
SIGNALPLANE_BOOTSTRAP_USER_PASSWORD=admin-password \
make run
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

## Option 4: Install On Kubernetes With Helm

The Helm chart is in:

```text
deploy/helm/signalplane
```

Production installs should use external HA PostgreSQL and ClickHouse. See [On-Prem Deployment](ON_PREM_DEPLOYMENT.md) for the full secret, values, and verification workflow.

Minimal install shape:

```bash
kubectl create namespace signalplane
kubectl -n signalplane create secret generic signalplane-runtime \
  --from-literal=ingest-token='<replace-me>' \
  --from-literal=bootstrap-user-email='admin@customer.local' \
  --from-literal=bootstrap-user-password='<replace-me>' \
  --from-literal=postgres-url='postgres://signalplane:<password>@postgres.customer.local:5432/signalplane?sslmode=require' \
  --from-literal=postgres-password='<postgres-password>' \
  --from-literal=clickhouse-password='<clickhouse-password>'
helm upgrade --install signalplane deploy/helm/signalplane \
  --namespace signalplane \
  --values deploy/helm/signalplane/examples/values-onprem-production.yaml
```

## Configuration

SignalPlane is configured with environment variables.

| Variable | Default | Purpose |
|---|---|---|
| `SIGNALPLANE_ADDR` | `127.0.0.1:4318` | Address and port for the HTTP server |
| `SIGNALPLANE_INGEST_TOKEN` | empty | Ingest-only token accepted by telemetry endpoints. Local Podman Compose sets `dev-token` |
| `SIGNALPLANE_BOOTSTRAP_ADMIN_TOKEN` | empty | Optional admin-scoped token created at startup for local automation. Local Podman Compose sets `dev-admin-token` |
| `SIGNALPLANE_BOOTSTRAP_USER_EMAIL` | empty | Optional local owner account email created at startup |
| `SIGNALPLANE_BOOTSTRAP_USER_PASSWORD` | empty | Optional local owner account password created at startup |
| `SIGNALPLANE_REQUIRE_READ_AUTH` | `true` | Require read/admin token or login session for telemetry read APIs |
| `SIGNALPLANE_DATA_PATH` | `data/signalplane.json` | File-backed persistence path |
| `SIGNALPLANE_SEED_DEMO_DATA` | `true` | Seed demo services, metrics, logs, traces, and uptime monitor |
| `SIGNALPLANE_STORE_BACKEND` | `json` | Runtime store backend. Use `json` for a local snapshot or `postgres` for PostgreSQL-backed runtime state |
| `SIGNALPLANE_TELEMETRY_BACKEND` | `json` | Telemetry archival backend. Use `clickhouse` with the local platform stack or production ClickHouse |
| `SIGNALPLANE_TELEMETRY_REPLAY_PATH` | empty | Optional JSONL replay queue for failed telemetry archive writes |
| `SIGNALPLANE_POSTGRES_ADDR` | empty | Optional dependency health check target |
| `SIGNALPLANE_POSTGRES_URL` | empty | PostgreSQL connection URL used when `SIGNALPLANE_STORE_BACKEND=postgres` |
| `SIGNALPLANE_POSTGRES_USER` | `signalplane` | PostgreSQL user used to build a connection URL when `SIGNALPLANE_POSTGRES_URL` is empty |
| `SIGNALPLANE_POSTGRES_PASSWORD` | `signalplane` | PostgreSQL password used to build a connection URL when `SIGNALPLANE_POSTGRES_URL` is empty |
| `SIGNALPLANE_POSTGRES_DATABASE` | `signalplane` | PostgreSQL database used to build a connection URL when `SIGNALPLANE_POSTGRES_URL` is empty |
| `SIGNALPLANE_POSTGRES_SSLMODE` | `disable` | PostgreSQL SSL mode used to build a connection URL when `SIGNALPLANE_POSTGRES_URL` is empty |
| `SIGNALPLANE_POSTGRES_TIMEOUT_SECONDS` | `5` | PostgreSQL startup/load/save timeout |
| `SIGNALPLANE_CLICKHOUSE_URL` | empty | ClickHouse HTTP endpoint used when `SIGNALPLANE_TELEMETRY_BACKEND=clickhouse` |
| `SIGNALPLANE_CLICKHOUSE_DATABASE` | `signalplane` | ClickHouse database for telemetry archival |
| `SIGNALPLANE_CLICKHOUSE_USER` | empty | Optional ClickHouse HTTP user |
| `SIGNALPLANE_CLICKHOUSE_PASSWORD` | empty | Optional ClickHouse HTTP password |
| `SIGNALPLANE_CLICKHOUSE_TIMEOUT_SECONDS` | `3` | ClickHouse write timeout |
| `SIGNALPLANE_CLICKHOUSE_HTTP_URL` | empty | Optional ClickHouse health check URL |
| `SIGNALPLANE_OTEL_GRPC_ADDR` | empty | Optional OTLP gRPC dependency health check target |
| `SIGNALPLANE_OTEL_HTTP_ADDR` | empty | Optional OTLP HTTP dependency health check target |
| `SIGNALPLANE_SMTP_ADDR` | empty | Optional SMTP dependency health check target |
| `SIGNALPLANE_NOTIFICATION_FROM` | `signalplane@localhost` | Sender used for email notification channels |
| `SIGNALPLANE_NOTIFICATION_TIMEOUT_SECONDS` | `5` | Notification delivery timeout |
| `SIGNALPLANE_MAILPIT_URL` | empty | Optional Mailpit web health check URL |
| `SIGNALPLANE_SECURE_COOKIES` | `false` | Set `true` behind HTTPS ingress so browser session cookies require TLS |
| `SIGNALPLANE_COOKIE_DOMAIN` | empty | Optional domain attribute for browser session cookies |
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

Source runs default to an atomic JSON snapshot file.

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

The full local stack provisions PostgreSQL and ClickHouse schemas. Podman Compose runs SignalPlane with `SIGNALPLANE_STORE_BACKEND=postgres`, which stores the API/dashboard runtime snapshot in PostgreSQL. When `SIGNALPLANE_TELEMETRY_BACKEND=clickhouse`, SignalPlane also archives incoming metrics, logs, traces, spans, and uptime results into ClickHouse over HTTP.

The current PostgreSQL runtime path persists the product state snapshot. A later Silver-hardening step should split that snapshot into normalized entity repositories for users, roles, dashboards, alert rules, notification channels, and richer metadata queries.

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
