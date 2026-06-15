# SignalPlane Project Structure

This repository is now organized around a runnable Silver self-hosted baseline plus product and architecture documents for Gold and Platinum expansion.

```text
.
├── cmd/signalplane
│   └── main.go
├── internal
│   ├── platform
│   │   └── dependencies.go
│   ├── server
│   │   ├── server.go
│   │   └── web
│   │       ├── index.html
│   │       ├── app.js
│   │       └── styles.css
│   └── store
│       └── store.go
├── deploy
│   ├── clickhouse
│   ├── otel-collector
│   └── postgres
├── examples
│   ├── demo-shop
│   └── test-applications
├── docs
├── Containerfile
├── docker-compose.yml
├── Makefile
└── go.mod
```

## Runtime

- `cmd/signalplane`: process entry point, environment configuration, store initialization, and graceful shutdown.
- `internal/platform`: local dependency health checks for PostgreSQL, ClickHouse, OpenTelemetry Collector, SMTP, and Mailpit.
- `internal/server`: HTTP API, auth checks, embedded static web UI, and request/response handling.
- `internal/server/web`: lightweight Silver dashboard.
- `internal/store`: in-memory domain model, JSON or PostgreSQL runtime snapshot persistence, token validation, service/host inference, telemetry ingestion, alert creation, and audit events.

## Deployment

- `deploy/postgres/init`: PostgreSQL schema for Silver control-plane state.
- `deploy/clickhouse/init`: ClickHouse schema for telemetry signals.
- `deploy/otel-collector/config.yaml`: local OpenTelemetry Collector receiver config.
- `docker-compose.yml`: full local Silver stack, run through Podman Compose by default.

## Examples

`examples/demo-shop` is the main observed demo application. It sends logs, metrics, traces, host heartbeats, and uptime registration to SignalPlane.

`examples/test-applications` contains small telemetry producers for common workloads:

- Go backend API.
- Node.js microservice.
- Python web backend.
- Python worker.
- Database/dependency simulator.
- C host probe.
- Kubernetes-style workload metadata.
- Uptime monitor registration.

These examples are intentionally dependency-light. They are used to populate the local dashboard and validate ingestion behavior.

## Documentation

- `docs/PRODUCT_STRATEGY.md`: tiered product strategy and Dynatrace-equivalent intent.
- `docs/SILVER_READINESS.md`: checklist for calling the product fully Silver-ready.
- `docs/PRODUCT_REQUIREMENTS.md`: full Silver, Gold, and Platinum requirements.
- `docs/ARCHITECTURE.md`: long-term logical architecture.
- `docs/HOW_IT_WORKS.md`: current Silver runtime behavior.
- `docs/API_REFERENCE.md`: current HTTP API.
- `docs/TELEMETRY_GUIDE.md`: telemetry payload conventions.
- `docs/INSTALLATION.md`: local and Podman install instructions.
- `docs/OPERATIONS.md`: operating and troubleshooting the Silver preview.
- `docs/ROADMAP.md`: phased build plan.
- `docs/BRAND.md`: naming, positioning, and product voice.

## Archive

The `archive` directory is treated as source material and historical context. It should not be used as the live application root. New implementation work should happen in the top-level project structure.
