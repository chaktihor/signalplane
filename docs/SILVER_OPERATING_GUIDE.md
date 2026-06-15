# SignalPlane Silver Operating Guide

This guide is for teams installing, demoing, operating, and extending the Silver tier.

## What Silver Includes

Silver is a self-hosted observability product for small teams and early adopters. It includes:

- SignalPlane API and web dashboard.
- JSON telemetry ingestion for metrics, logs, traces, hosts, and uptime.
- OTLP HTTP JSON and protobuf ingestion at `/v1/metrics`, `/v1/logs`, and `/v1/traces`.
- PostgreSQL-backed runtime persistence in the Podman stack.
- ClickHouse telemetry archival and ClickHouse-backed telemetry query APIs.
- Durable telemetry replay queue for failed ClickHouse writes.
- Service and host inference from resource metadata.
- Uptime checks.
- Built-in and configurable alert rules.
- Email, generic webhook, and Slack-compatible webhook notification channels.
- Local user login sessions, roles, API tokens, and admin APIs.
- Dependency health checks for PostgreSQL, ClickHouse, OpenTelemetry Collector, SMTP, and Mailpit.
- Helm deployment assets for on-prem and cloud Kubernetes installs.

## Recommended Hardware

### Local Demo

Use this for laptop demos and evaluation:

| Component | Minimum |
|---|---:|
| CPU | 4 cores |
| RAM | 8 GB |
| Disk | 20 GB free SSD |
| Network | Internet access for first image pull |

### Small Team Pilot

Use this for a small team running real development or staging workloads:

| Component | Recommended |
|---|---:|
| SignalPlane API | 2 vCPU, 2-4 GB RAM |
| PostgreSQL | 2 vCPU, 4 GB RAM, SSD volume |
| ClickHouse | 4-8 vCPU, 16-32 GB RAM, SSD/NVMe volume |
| OpenTelemetry Collector | 2 vCPU, 2 GB RAM |
| Disk | Size for retention target; start at 100-250 GB for ClickHouse |

### Production HA Baseline

For HA, run:

- 2+ SignalPlane API replicas behind a load balancer.
- 3 PostgreSQL nodes or a managed PostgreSQL service with automated failover.
- 3+ ClickHouse nodes using replicated tables and distributed queries.
- 2+ OpenTelemetry Collector replicas per network zone.
- External SMTP or webhook delivery provider.
- Shared secrets managed through Kubernetes Secrets, Vault, or cloud secret manager.

## Required Software

Local stack:

- Podman with Podman Compose support.
- Go 1.26 for source builds.
- Git.
- curl or wget for checks.

Containerized services:

- PostgreSQL 16.
- ClickHouse 24.8.
- OpenTelemetry Collector Contrib 0.111.
- Mailpit for local email testing.
- Helm 3 for Kubernetes installs.

## Install The Local Silver Stack

Start the stack:

```bash
make stack-up
```

Open:

```text
http://127.0.0.1:4318
```

Default local admin:

```text
email: admin@signalplane.local
password: admin-password
token: dev-token
```

Check health:

```bash
curl http://127.0.0.1:4318/healthz
curl http://127.0.0.1:4318/api/system/dependencies
```

Stop:

```bash
make stack-down
```

Reset all persisted data:

```bash
make stack-reset
```

## What Happens Under The Hood

1. Applications send telemetry to SignalPlane JSON endpoints or OTLP HTTP JSON/protobuf endpoints.
2. SignalPlane authenticates the request with an API token.
3. SignalPlane normalizes resource metadata such as service, host, environment, region, and version.
4. SignalPlane infers services and hosts.
5. SignalPlane updates the runtime model.
6. Runtime state is persisted to PostgreSQL when `SIGNALPLANE_STORE_BACKEND=postgres`.
7. Telemetry is written to ClickHouse when `SIGNALPLANE_TELEMETRY_BACKEND=clickhouse`.
8. If ClickHouse is temporarily unavailable, failed telemetry writes are appended to `SIGNALPLANE_TELEMETRY_REPLAY_PATH`.
9. A background replay loop retries spooled telemetry every 10 seconds.
10. Query APIs read telemetry from ClickHouse when configured and fall back to runtime state if ClickHouse queries fail.
11. Built-in alert logic and configured alert rules evaluate incoming telemetry.
12. Alerts are persisted and sent to enabled notification channels.

## How Logs Are Captured

Logs can enter SignalPlane through:

- `POST /api/ingest/logs`
- `POST /v1/logs` for OTLP HTTP JSON or protobuf

Each log should include:

- `timestamp`
- `severity`
- `message`
- optional `traceId` and `spanId`
- `resource` metadata
- custom fields

SignalPlane stores recent runtime state in PostgreSQL and archives logs to ClickHouse. ClickHouse stores logs in `signalplane.logs` with 30-day default TTL from the schema.

## How Metrics Are Captured

Metrics can enter through:

- `POST /api/ingest/metrics`
- `POST /v1/metrics` for OTLP HTTP JSON or protobuf

Metric samples include name, value, unit, type, labels, and resource metadata. SignalPlane uses metric samples to update service health, trigger built-in threshold alerts, evaluate configured metric rules, and archive the sample to ClickHouse.

## How Traces Are Captured

Traces can enter through:

- `POST /api/ingest/traces`
- `POST /v1/traces` for OTLP HTTP JSON or protobuf

SignalPlane groups spans by trace ID, infers service relationships from span resource metadata, stores recent runtime trace state, and archives traces and spans to ClickHouse tables `signalplane.traces` and `signalplane.spans`.

## Alert Rules

Create a metric threshold rule:

```bash
curl -X POST http://127.0.0.1:4318/api/alert-rules \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High checkout latency",
    "signalType": "metric",
    "metricName": "http.server.duration",
    "operator": "gte",
    "threshold": 500,
    "severity": "warning",
    "labels": {"team": "checkout"}
  }'
```

Create a log rule:

```bash
curl -X POST http://127.0.0.1:4318/api/alert-rules \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Payment timeout logs",
    "signalType": "log",
    "logSeverity": "error",
    "query": "timeout",
    "severity": "critical"
  }'
```

## Notification Channels

Create an email channel for Mailpit:

```bash
curl -X POST http://127.0.0.1:4318/api/notification-channels \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Local email",
    "type": "email",
    "target": "oncall@signalplane.local"
  }'
```

Test it:

```bash
curl -X POST http://127.0.0.1:4318/api/notification-channels/<id>/test \
  -H "Authorization: Bearer dev-token"
```

Open Mailpit:

```text
http://127.0.0.1:8025
```

Webhook channels use `type: "webhook"` and Slack-compatible channels use `type: "slack_webhook"`.

## Authentication And Roles

Local admin credentials come from:

- `SIGNALPLANE_BOOTSTRAP_USER_EMAIL`
- `SIGNALPLANE_BOOTSTRAP_USER_PASSWORD`

Login:

```bash
curl -X POST http://127.0.0.1:4318/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@signalplane.local","password":"admin-password"}'
```

Roles:

| Role | Access |
|---|---|
| `owner` | Full access |
| `admin` | Full access |
| `editor` | Read and ingest |
| `viewer` | Read only |

API tokens are still the recommended path for collectors and applications.

For HTTPS deployments, set `SIGNALPLANE_SECURE_COOKIES=true` so browser session cookies are only sent over TLS. Set `SIGNALPLANE_COOKIE_DOMAIN` when the customer requires an explicit cookie domain.

## Archival And Retention

ClickHouse tables are created under the `signalplane` database:

- `metrics`
- `logs`
- `traces`
- `spans`
- `uptime_results`
- `events`

The default schema applies a 30-day TTL to telemetry tables. Change TTL in `deploy/clickhouse/init/001_telemetry_schema.sql` before initializing a production ClickHouse cluster, or apply an `ALTER TABLE ... MODIFY TTL ...` migration later.

Backups:

- Back up PostgreSQL for runtime metadata and configuration.
- Back up ClickHouse parts or use ClickHouse-native backup tooling for telemetry.
- Preserve `/data/telemetry-replay.jsonl` during upgrades so failed telemetry writes can replay.

## HA Deployment Model

### SignalPlane API

SignalPlane is stateless apart from PostgreSQL, ClickHouse, and the optional local replay file. For HA:

- Run 2+ API replicas.
- Put a load balancer in front of port `4318`.
- Use shared PostgreSQL and ClickHouse backends.
- Put replay queues on persistent volumes or send failed writes to a shared queue in a future deployment.

### PostgreSQL

Use managed PostgreSQL or a HA PostgreSQL cluster with:

- streaming replication
- automated failover
- daily backups
- point-in-time recovery
- TLS in production

### ClickHouse

Use replicated MergeTree tables and multiple shards/replicas for HA. Production ClickHouse should use:

- replicated tables
- distributed tables for query fanout
- object storage backups
- disk monitoring
- TTL policies aligned to budget and compliance

### OpenTelemetry Collector

Run collectors close to workloads. In Kubernetes, use:

- DaemonSet collectors for node-local telemetry
- Deployment collectors for gateway aggregation
- retry and batch processors
- memory limits

### Notifications

Use external SMTP/webhook providers for production. Mailpit is only for local testing.

## Troubleshooting

Check service status:

```bash
podman compose ps
```

Check logs:

```bash
podman compose logs signalplane
podman compose logs clickhouse
podman compose logs postgres
```

Check dependency health:

```bash
curl http://127.0.0.1:4318/api/system/dependencies
```

Verify ClickHouse archival:

```bash
podman compose exec -T clickhouse wget -qO- \
  --header 'Authorization: Basic c2lnbmFscGxhbmU6c2lnbmFscGxhbmU=' \
  'http://127.0.0.1:8123/?query=SELECT%20count()%20FROM%20signalplane.logs'
```

Verify PostgreSQL runtime state:

```bash
podman compose exec -T postgres psql -U signalplane -d signalplane \
  -c "SELECT id, updated_at FROM runtime_snapshots"
```

If telemetry is not visible:

- Confirm the application sends `Authorization: Bearer <token>`.
- Confirm resource metadata includes `service` or `host`.
- Check ClickHouse health.
- Check `SIGNALPLANE_TELEMETRY_REPLAY_PATH` for queued failed writes.
- Check SignalPlane logs for query fallback warnings.
