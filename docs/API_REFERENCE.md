# SignalPlane API Reference

Base URL:

```text
http://127.0.0.1:4318
```

## Authentication

Ingestion, token-management, and state-changing configuration endpoints require an API token.

Use:

```text
Authorization: Bearer <token>
```

or:

```text
X-SignalPlane-Token: <token>
```

Default local bootstrap/admin token:

```text
dev-token
```

## Health

### `GET /healthz`

Returns server health.

```bash
curl http://127.0.0.1:4318/healthz
```

## Bootstrap

### `GET /api/bootstrap`

Returns summary counts, health, recent alerts, top services, and top hosts.

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
curl 'http://127.0.0.1:4318/api/services?limit=50'
```

## Hosts

### `GET /api/hosts`

Returns inferred hosts.

```bash
curl 'http://127.0.0.1:4318/api/hosts?limit=50'
```

## Metrics

### `GET /api/metrics`

Returns recent metric samples.

```bash
curl 'http://127.0.0.1:4318/api/metrics?limit=50'
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

Returns recent logs.

Query parameters:

| Parameter | Description |
|---|---|
| `limit` | Max results |
| `service` | Filter by service |
| `severity` | Filter by severity |
| `q` | Search message text |

```bash
curl 'http://127.0.0.1:4318/api/logs?service=orders-api&severity=error'
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

Returns recent traces.

Query parameters:

| Parameter | Description |
|---|---|
| `limit` | Max results |
| `service` | Filter by service |
| `status` | Filter by `ok` or `error` |

```bash
curl 'http://127.0.0.1:4318/api/traces?service=orders-api'
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
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"status":"acknowledged"}'
```

## Incidents

### `GET /api/incidents`

Returns incidents.

```bash
curl http://127.0.0.1:4318/api/incidents
```

### `POST /api/incidents`

Requires `admin` token.

Creates an incident.

```bash
curl -X POST http://127.0.0.1:4318/api/incidents \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"title":"Checkout degradation","severity":"warning","owner":"platform"}'
```

## Uptime Monitors

### `GET /api/uptime-monitors`

Returns uptime monitor definitions and latest check results.

```bash
curl http://127.0.0.1:4318/api/uptime-monitors
```

### `POST /api/uptime-monitors`

Requires `admin` token.

Creates an uptime monitor definition. SignalPlane checks due monitors in the background.

```bash
curl -X POST http://127.0.0.1:4318/api/uptime-monitors \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"name":"API health","url":"http://localhost:8080/healthz","expectedStatus":200}'
```

### `POST /api/uptime-monitors/{id}/check`

Requires `admin` token.

Runs an immediate uptime check and stores the latest result.

```bash
curl -X POST http://127.0.0.1:4318/api/uptime-monitors/upt-demo-shop/check \
  -H "Authorization: Bearer dev-token"
```

## System Dependencies

### `GET /api/system/dependencies`

Returns health checks for configured local platform dependencies such as PostgreSQL, ClickHouse, OpenTelemetry Collector, SMTP, and Mailpit.

```bash
curl http://127.0.0.1:4318/api/system/dependencies
```

## Tokens

### `GET /api/tokens`

Requires `admin` token.

Returns tokens with masked token values.

```bash
curl http://127.0.0.1:4318/api/tokens \
  -H "Authorization: Bearer dev-token"
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
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"name":"orders-api","scope":"ingest","token":"orders-dev-token"}'
```

## OpenAPI Placeholder

### `GET /api/openapi`

Returns a lightweight API listing. A full OpenAPI schema is planned.
