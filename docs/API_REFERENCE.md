# SignalPlane API Reference

Base URL:

```text
http://127.0.0.1:4318
```

## Authentication

Read APIs, ingestion, token-management, user-management, alert-rule, notification-channel, and state-changing configuration endpoints require either a scoped API token or a login session when `SIGNALPLANE_REQUIRE_READ_AUTH=true`.

Use:

```text
Authorization: Bearer <token>
```

or:

```text
X-SignalPlane-Token: <token>
```

Default local ingest token:

```text
dev-token
```

Default local bootstrap admin token in the Podman stack:

```text
dev-admin-token
```

Default local stack owner:

```text
admin@signalplane.local / admin-password
```

### `POST /api/auth/login`

Creates a session cookie and returns the logged-in user.

```bash
curl -X POST http://127.0.0.1:4318/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@signalplane.local","password":"admin-password"}'
```

### `GET /api/me`

Returns the current session user.

### `POST /api/auth/logout`

Revokes the current session.

## Health

### `GET /healthz`

Returns server health.

```bash
curl http://127.0.0.1:4318/healthz
```

## Bootstrap

### `GET /api/bootstrap`

Returns summary counts, health, recent alerts, top services, and top hosts.

Requires a read/admin token or login session when read auth is enabled.

```bash
curl http://127.0.0.1:4318/api/bootstrap
```

## Services

### `GET /api/services`

Returns inferred services.

Query parameters:

| Parameter | Description |
|---|---|
| `limit` | Max results, default `100`, max `500` |

```bash
curl 'http://127.0.0.1:4318/api/services?limit=50' \
  -H "Authorization: Bearer dev-admin-token"
```

## Hosts

### `GET /api/hosts`

Returns inferred hosts.

```bash
curl 'http://127.0.0.1:4318/api/hosts?limit=50' \
  -H "Authorization: Bearer dev-admin-token"
```

## Metrics

### `GET /api/metrics`

Returns recent metric samples. When ClickHouse is configured, results come from ClickHouse with runtime fallback.

```bash
curl 'http://127.0.0.1:4318/api/metrics?limit=50' \
  -H "Authorization: Bearer dev-admin-token"
```

### `POST /api/ingest/metrics`

Requires `ingest` or `admin` token.

Accepts a single metric, an array of metrics, or an object with a `metrics` array.

```bash
curl -X POST http://127.0.0.1:4318/api/ingest/metrics \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"name":"requests","value":1,"resource":{"service":"api"}}'
```

## Logs

### `GET /api/logs`

Returns recent logs. When ClickHouse is configured, results come from ClickHouse with runtime fallback.

Query parameters:

| Parameter | Description |
|---|---|
| `limit` | Max results |
| `service` | Filter by service |
| `severity` | Filter by severity |
| `q` | Search message text |

```bash
curl 'http://127.0.0.1:4318/api/logs?service=orders-api&severity=error' \
  -H "Authorization: Bearer dev-admin-token"
```

### `POST /api/ingest/logs`

Requires `ingest` or `admin` token.

Accepts a single log, an array of logs, or an object with a `logs` array.

```bash
curl -X POST http://127.0.0.1:4318/api/ingest/logs \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"severity":"info","message":"hello","resource":{"service":"api"}}'
```

## Traces

### `GET /api/traces`

Returns recent traces. When ClickHouse is configured, results come from ClickHouse with runtime fallback.

Query parameters:

| Parameter | Description |
|---|---|
| `limit` | Max results |
| `service` | Filter by service |
| `status` | Filter by `ok` or `error` |

```bash
curl 'http://127.0.0.1:4318/api/traces?service=orders-api' \
  -H "Authorization: Bearer dev-admin-token"
```

### `POST /api/ingest/traces`

Requires `ingest` or `admin` token.

Accepts a single trace, an array of traces, or an object with a `traces` array.

```bash
curl -X POST http://127.0.0.1:4318/api/ingest/traces \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"traceId":"trace-1","spans":[{"spanId":"span-1","name":"GET /","durationMs":12,"resource":{"service":"api"}}]}'
```

## Hosts Ingestion

### `POST /api/ingest/hosts`

Requires `ingest` or `admin` token.

```bash
curl -X POST http://127.0.0.1:4318/api/ingest/hosts \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"name":"host-1","status":"online","metrics":{"cpu":30,"memory":60}}'
```

## Alerts

### `GET /api/alerts`

Returns alerts.

```bash
curl http://127.0.0.1:4318/api/alerts
```

### `PATCH /api/alerts/{id}`

Requires `admin` token.

Updates alert status.

Statuses:

- `open`
- `acknowledged`
- `resolved`

```bash
curl -X PATCH http://127.0.0.1:4318/api/alerts/ALERT_ID \
  -H "Authorization: Bearer dev-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"status":"acknowledged"}'
```

## Incidents

### `GET /api/incidents`

Returns incidents.

```bash
curl http://127.0.0.1:4318/api/incidents \
  -H "Authorization: Bearer dev-admin-token"
```

### `POST /api/incidents`

Requires `admin` token.

Creates an incident.

```bash
curl -X POST http://127.0.0.1:4318/api/incidents \
  -H "Authorization: Bearer dev-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"title":"Checkout degradation","severity":"warning","owner":"platform"}'
```

## Uptime Monitors

### `GET /api/uptime-monitors`

Returns uptime monitor definitions and latest check results.

```bash
curl http://127.0.0.1:4318/api/uptime-monitors \
  -H "Authorization: Bearer dev-admin-token"
```

### `POST /api/uptime-monitors`

Requires `admin` token.

Creates an uptime monitor definition. SignalPlane checks due monitors in the background.

```bash
curl -X POST http://127.0.0.1:4318/api/uptime-monitors \
  -H "Authorization: Bearer dev-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"name":"API health","url":"http://localhost:8080/healthz","expectedStatus":200}'
```

### `POST /api/uptime-monitors/{id}/check`

Requires `admin` token.

Runs an immediate uptime check and stores the latest result.

```bash
curl -X POST http://127.0.0.1:4318/api/uptime-monitors/upt-demo-shop/check \
  -H "Authorization: Bearer dev-admin-token"
```

## System Dependencies

### `GET /api/system/dependencies`

Returns health checks for configured local platform dependencies such as PostgreSQL, ClickHouse, OpenTelemetry Collector, SMTP, and Mailpit.

```bash
curl http://127.0.0.1:4318/api/system/dependencies \
  -H "Authorization: Bearer dev-admin-token"
```

## Tokens

### `GET /api/tokens`

Requires `admin` token.

Returns tokens with masked token values.

```bash
curl http://127.0.0.1:4318/api/tokens \
  -H "Authorization: Bearer dev-admin-token"
```

### `POST /api/tokens`

Requires `admin` token.

Creates a token.

Scopes:

- `admin`
- `ingest`
- `read`

```bash
curl -X POST http://127.0.0.1:4318/api/tokens \
  -H "Authorization: Bearer dev-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"name":"orders-api","scope":"ingest","token":"orders-dev-token"}'
```

## Users

### `GET /api/users`

Requires admin access.

Returns local users without password hashes.

```bash
curl http://127.0.0.1:4318/api/users \
  -H "Authorization: Bearer dev-admin-token"
```

### `POST /api/users`

Requires admin access.

Creates a local user.

```bash
curl -X POST http://127.0.0.1:4318/api/users \
  -H "Authorization: Bearer dev-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"email":"viewer@example.com","displayName":"Viewer","role":"viewer","password":"change-me"}'
```

## Alert Rules

### `GET /api/alert-rules`

Requires admin access.

```bash
curl http://127.0.0.1:4318/api/alert-rules \
  -H "Authorization: Bearer dev-admin-token"
```

### `POST /api/alert-rules`

Requires admin access.

Creates a metric or log alert rule.

```bash
curl -X POST http://127.0.0.1:4318/api/alert-rules \
  -H "Authorization: Bearer dev-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"name":"High latency","signalType":"metric","metricName":"http.server.duration","operator":"gte","threshold":500,"severity":"warning"}'
```

## Notification Channels

### `GET /api/notification-channels`

Requires admin access.

### `POST /api/notification-channels`

Requires admin access.

Creates an email, webhook, or Slack-compatible webhook channel.

```bash
curl -X POST http://127.0.0.1:4318/api/notification-channels \
  -H "Authorization: Bearer dev-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"name":"Local email","type":"email","target":"oncall@signalplane.local"}'
```

### `POST /api/notification-channels/{id}/test`

Sends a test notification through the selected channel.

## OTLP HTTP

SignalPlane accepts OTLP HTTP JSON and protobuf payloads:

- `POST /v1/metrics`
- `POST /v1/logs`
- `POST /v1/traces`

These endpoints require an ingest or admin token and map OTLP resource attributes into SignalPlane service, host, environment, region, version, labels, and fields. Protobuf requests should use `Content-Type: application/x-protobuf`.

## OpenAPI Contract

### `GET /api/openapi`

Returns the authenticated OpenAPI 3.1 contract for the Silver API. The contract includes:

- Bearer, `X-SignalPlane-Token`, and session-cookie security schemes.
- Path definitions for read, admin, JSON ingestion, and OTLP HTTP ingestion APIs.
- JSON request-body schemas for SignalPlane-native ingestion and configuration endpoints.
- Response envelopes and shared schemas for resources, logs, metrics, traces, alerts, incidents, monitors, tokens, users, dependencies, and errors.

Use this endpoint as the source for generated clients and customer-facing API review. It requires a read/admin token or login session when read auth is enabled.
