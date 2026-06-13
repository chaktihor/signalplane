# SignalPlane Telemetry Guide

This guide explains how to send telemetry to SignalPlane.

## Authentication

Ingestion endpoints require a token.

Default local bootstrap token:

```text
dev-token
```

Use either header:

```text
Authorization: Bearer dev-token
```

or:

```text
X-SignalPlane-Token: dev-token
```

## Create A Scoped Ingestion Token

```bash
curl -X POST http://127.0.0.1:4318/api/tokens \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "orders-api",
    "scope": "ingest",
    "token": "orders-dev-token"
  }'
```

Use `orders-dev-token` for ingestion:

```text
Authorization: Bearer orders-dev-token
```

## Resource Metadata

Every telemetry payload should include resource metadata.

```json
{
  "resource": {
    "service": "orders-api",
    "host": "orders-1",
    "environment": "production",
    "region": "local",
    "version": "0.1.0",
    "attributes": {
      "team": "checkout",
      "runtime": "go"
    }
  }
}
```

Recommended fields:

| Field | Purpose |
|---|---|
| `service` | Application, dependency, worker, database, or external service name |
| `host` | Host, VM, container, pod, or process instance |
| `environment` | `development`, `staging`, `production`, etc. |
| `region` | Region or location |
| `version` | App or deployment version |
| `attributes` | Extra metadata such as team, namespace, runtime, pod, container |

## Metrics

Endpoint:

```text
POST /api/ingest/metrics
```

Single metric:

```bash
curl -X POST http://127.0.0.1:4318/api/ingest/metrics \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "http.server.duration",
    "value": 93,
    "unit": "ms",
    "type": "histogram",
    "labels": {
      "route": "/orders",
      "method": "POST"
    },
    "resource": {
      "service": "orders-api",
      "host": "orders-1",
      "environment": "production"
    }
  }'
```

Multiple metrics:

```bash
curl -X POST http://127.0.0.1:4318/api/ingest/metrics \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "metrics": [
      {
        "name": "http.server.requests",
        "value": 1280,
        "unit": "requests",
        "type": "counter",
        "resource": {
          "service": "orders-api",
          "host": "orders-1",
          "environment": "production"
        }
      },
      {
        "name": "http.server.duration",
        "value": 93,
        "unit": "ms",
        "type": "histogram",
        "resource": {
          "service": "orders-api",
          "host": "orders-1",
          "environment": "production"
        }
      }
    ]
  }'
```

Metric fields:

| Field | Required | Description |
|---|---|---|
| `name` | Yes | Metric name |
| `value` | Yes | Numeric value |
| `unit` | No | Unit such as `ms`, `percent`, `requests`, `bytes` |
| `type` | No | `counter`, `gauge`, `histogram`, or custom value |
| `labels` | No | Metric dimensions |
| `resource` | Recommended | Service/host/environment metadata |

## Logs

Endpoint:

```text
POST /api/ingest/logs
```

```bash
curl -X POST http://127.0.0.1:4318/api/ingest/logs \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "severity": "error",
    "message": "database timeout",
    "traceId": "trace-orders-1",
    "spanId": "span-db",
    "fields": {
      "db.system": "postgres",
      "timeout_ms": "5000"
    },
    "resource": {
      "service": "orders-api",
      "host": "orders-1",
      "environment": "production"
    }
  }'
```

Log fields:

| Field | Required | Description |
|---|---|---|
| `severity` | No | `debug`, `info`, `warning`, `error`, or `fatal` |
| `message` | Yes | Log message |
| `traceId` | No | Trace correlation ID |
| `spanId` | No | Span correlation ID |
| `fields` | No | Structured fields |
| `resource` | Recommended | Service/host/environment metadata |

`error` logs create warning alerts. `fatal` logs create critical alerts.

## Traces

Endpoint:

```text
POST /api/ingest/traces
```

```bash
curl -X POST http://127.0.0.1:4318/api/ingest/traces \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "traceId": "trace-orders-1",
    "spans": [
      {
        "spanId": "span-root",
        "name": "POST /orders",
        "durationMs": 93,
        "status": "ok",
        "resource": {
          "service": "orders-api",
          "host": "orders-1",
          "environment": "production"
        }
      },
      {
        "spanId": "span-db",
        "parentId": "span-root",
        "name": "SELECT orders",
        "durationMs": 41,
        "status": "ok",
        "resource": {
          "service": "postgres-orders-db",
          "host": "db-1",
          "environment": "production"
        }
      }
    ]
  }'
```

Trace fields:

| Field | Required | Description |
|---|---|---|
| `traceId` | Recommended | Trace correlation ID |
| `spans` | Yes | List of spans |

Span fields:

| Field | Required | Description |
|---|---|---|
| `spanId` | Recommended | Span ID |
| `parentId` | No | Parent span ID |
| `name` | Yes | Operation name |
| `durationMs` | No | Duration in milliseconds |
| `status` | No | `ok` or `error` |
| `resource` | Recommended | Service/host/environment metadata |
| `attributes` | No | Span attributes |

Error traces create warning alerts.

## Hosts

Endpoint:

```text
POST /api/ingest/hosts
```

```bash
curl -X POST http://127.0.0.1:4318/api/ingest/hosts \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "host-1",
    "name": "host-1",
    "environment": "production",
    "region": "local",
    "status": "online",
    "agentVersion": "custom-probe-0.1.0",
    "tags": ["api", "linux"],
    "metrics": {
      "cpu": 31.2,
      "memory": 68.4,
      "disk": 44.1
    }
  }'
```

## Language Examples

SignalPlane currently supports manual HTTP JSON ingestion from any language.

Examples are available in:

```text
examples/test-applications
```

Run all examples:

```bash
./examples/test-applications/run_all.sh
```

Supported examples:

- Go backend API.
- Node microservice.
- Python web backend.
- Python worker.
- Database simulator.
- C host probe.
- Kubernetes-style workload metadata.
- Uptime monitor registration.

## Future OpenTelemetry Support

The current Silver preview uses SignalPlane JSON ingestion. A future milestone should add OTLP HTTP/gRPC ingestion so any OpenTelemetry-compatible language SDK can send telemetry directly.

