# SignalPlane Silver Quickstart

## Run Locally

```bash
make run
```

Open:

```text
http://127.0.0.1:4318
```

The default local ingestion token is:

```text
dev-token
```

SignalPlane stores local data in:

```text
data/signalplane.json
```

Override it with:

```bash
SIGNALPLANE_DATA_PATH=/path/to/signalplane.json make run
```

## Run With Podman Compose

```bash
make stack-up
```

Open:

```text
http://127.0.0.1:4318
```

Podman Compose persists data in the `signalplane-data` volume.

## Health Check

```bash
curl http://127.0.0.1:4318/healthz
```

## Ingest A Log

```bash
curl -X POST http://127.0.0.1:4318/api/ingest/logs \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "severity": "error",
    "message": "database timeout",
    "traceId": "trace-local-1",
    "resource": {
      "service": "orders-api",
      "host": "orders-1",
      "environment": "production"
    }
  }'
```

## Create An Ingestion Token

Use an admin token or owner login session to create scoped collector tokens. The local Podman stack bootstraps `dev-admin-token` for admin automation:

```bash
curl -X POST http://127.0.0.1:4318/api/tokens \
  -H "Authorization: Bearer dev-admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "orders-collector",
    "scope": "ingest",
    "token": "orders-dev-token"
  }'
```

List tokens:

```bash
curl http://127.0.0.1:4318/api/tokens \
  -H "Authorization: Bearer dev-admin-token"
```

## Ingest A Metric

```bash
curl -X POST http://127.0.0.1:4318/api/ingest/metrics \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "http.server.duration",
    "value": 93,
    "unit": "ms",
    "type": "histogram",
    "resource": {
      "service": "orders-api",
      "host": "orders-1",
      "environment": "production"
    }
  }'
```

## Ingest A Trace

```bash
curl -X POST http://127.0.0.1:4318/api/ingest/traces \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "traceId": "trace-local-1",
    "spans": [
      {
        "spanId": "span-root",
        "name": "POST /orders",
        "durationMs": 93,
        "status": "ok",
        "resource": {
          "service": "orders-api",
          "environment": "production"
        }
      }
    ]
  }'
```

## Build

```bash
make build
./bin/signalplane
```
