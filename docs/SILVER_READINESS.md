# SignalPlane Silver Readiness

Silver is the first self-hosted SignalPlane product tier. It should be useful for a small team running real services, not just a demo.

## Current Status

SignalPlane is Silver-demo ready and has the start of the Silver product foundation:

- Single Go service with embedded dashboard and HTTP API.
- JSON telemetry ingestion for metrics, logs, traces, and hosts.
- Service and host inference from resource metadata.
- Metric/log/trace/uptime alert creation.
- Incident records.
- Local uptime checks.
- Scoped API tokens.
- Demo checkout application.
- Podman Compose platform stack with PostgreSQL, ClickHouse, OpenTelemetry Collector, and Mailpit.
- Postgres schema for Silver control-plane metadata.
- PostgreSQL-backed runtime snapshot persistence in the platform stack.
- ClickHouse schema for telemetry-scale signal data.
- ClickHouse HTTP telemetry archival for metrics, logs, traces, spans, and uptime results in the platform stack.
- Platform dependency health checks in the UI.

## Silver-Ready Acceptance Criteria

SignalPlane should not be called fully Silver-ready until these are complete:

- Real email/password login and session UI.
- Normalized organization, user, environment, role, token, dashboard, alert-rule, notification-channel, and audit repositories in PostgreSQL.
- OTLP HTTP/gRPC ingestion through the collector or native gateway.
- Query APIs backed by ClickHouse rather than in-memory JSON snapshots.
- Durable telemetry write failure handling and replay/backfill from the local JSON snapshot into ClickHouse.
- Configurable alert rules for metrics, logs, traces, and uptime checks.
- Email, generic webhook, and Slack-compatible webhook notification channels.
- Dashboard create/edit/clone/delete and JSON import/export.
- Dedicated explorer pages for metrics, logs, traces, services, hosts, alerts, incidents, and uptime.
- Retention settings and bounded local data growth.
- CI, release packaging, and documented upgrade/reset paths.

## Local Platform Stack

Run the full local stack:

```bash
make stack-up
```

Services:

| Service | Purpose | Local URL |
|---|---|---|
| SignalPlane | Product API and UI | `http://127.0.0.1:4318` |
| PostgreSQL | Metadata/control-plane store | `127.0.0.1:5432` |
| ClickHouse | Telemetry store | `http://127.0.0.1:8123` |
| OpenTelemetry Collector | OTLP receiver | `127.0.0.1:4317`, `127.0.0.1:4319` |
| Mailpit | Email notification sink | `http://127.0.0.1:8025` |

Reset local stack state:

```bash
make stack-reset
```

## Next Implementation Slice

The next Silver-hardening slice should make the durable stores query-native:

- Normalized PostgreSQL repositories for organizations, users, tokens, services, hosts, alert rules, alerts, incidents, dashboards, uptime monitors, notification channels, and audit events.
- ClickHouse query layer for metrics, logs, traces, spans, uptime results, and events.
- Migration/version tracking.
- Backfill/import path from JSON or PostgreSQL runtime snapshots for local developer continuity.

After that, implement OTLP ingestion and alert rule evaluation against persisted data.
