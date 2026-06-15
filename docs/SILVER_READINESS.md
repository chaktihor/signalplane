# SignalPlane Silver Readiness

Silver is the first self-hosted SignalPlane product tier. It should be useful for a small team running real services, not just a demo.

## Current Status

SignalPlane is Silver-ready for local self-hosted pilots and demos:

- Single Go service with embedded dashboard and HTTP API.
- JSON telemetry ingestion for metrics, logs, traces, and hosts.
- OTLP HTTP JSON ingestion for metrics, logs, and traces.
- Service and host inference from resource metadata.
- Built-in metric/log/trace/uptime alert creation.
- Configurable metric and log alert rules.
- Email, generic webhook, and Slack-compatible webhook notification channels.
- Incident records.
- Local uptime checks.
- Scoped API tokens.
- Local user login sessions, roles, and user-management APIs.
- Demo checkout application.
- Podman Compose platform stack with PostgreSQL, ClickHouse, OpenTelemetry Collector, and Mailpit.
- Postgres schema for Silver control-plane metadata.
- PostgreSQL-backed runtime snapshot persistence in the platform stack.
- ClickHouse schema for telemetry-scale signal data.
- ClickHouse HTTP telemetry archival for metrics, logs, traces, spans, and uptime results in the platform stack.
- ClickHouse-backed telemetry query APIs with runtime fallback.
- Durable local telemetry replay queue for failed ClickHouse writes.
- Platform dependency health checks in the UI.
- End-user and operator documentation.

## Remaining Silver Hardening

These are not blockers for a Silver pilot, but they should be completed before a larger production rollout:

- Login form in the web UI; the API login/session path is implemented.
- Normalized organization, user, environment, role, token, dashboard, alert-rule, notification-channel, and audit repositories in PostgreSQL.
- OTLP protobuf/gRPC native decoding; OTLP HTTP JSON is implemented.
- Trace and uptime alert-rule types; metric/log configurable rules are implemented.
- Dashboard create/edit/clone/delete and JSON import/export.
- Dedicated explorer pages for metrics, logs, traces, services, hosts, alerts, incidents, and uptime.
- Runtime-configurable retention settings; ClickHouse schema TTL is currently static.
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

The next Silver-hardening slice should improve production ergonomics:

- Normalized PostgreSQL repositories for organizations, users, tokens, services, hosts, alert rules, alerts, incidents, dashboards, uptime monitors, notification channels, and audit events.
- Web UI forms for login, alert rules, notification channels, and telemetry explorers.
- Migration/version tracking.
- Backfill/import path from JSON or PostgreSQL runtime snapshots for local developer continuity.
- OTLP protobuf/gRPC compatibility.
