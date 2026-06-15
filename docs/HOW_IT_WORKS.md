# How SignalPlane Works

SignalPlane collects telemetry from applications and infrastructure, stores it, correlates it into services and hosts, and presents it through an API and web dashboard.

## Current Silver Architecture

```text
Applications / scripts / probes
        |
        | HTTP JSON telemetry
        v
SignalPlane Go server
        |
        | validates token
        | normalizes resource metadata
        | infers services and hosts
        | creates alerts from error signals
        v
In-memory runtime store
        |
        | atomic snapshot
        v
data/signalplane.json
        |
        v
Embedded web dashboard and API responses
```

## Core Concepts

### Service

A service is an application, API, worker, database, queue, external dependency, or workload that emits telemetry.

Examples:

- `orders-api`
- `payments-service`
- `invoice-worker`
- `postgres-orders-db`
- `external-payment-gateway`

SignalPlane currently infers services from telemetry resource metadata:

```json
{
  "resource": {
    "service": "orders-api"
  }
}
```

### Host

A host is a machine, VM, container host, pod-like instance, or process runtime location.

Examples:

- `api-1`
- `worker-1`
- `db-1`
- `cart-pod-7d9f6c9c8b-x1`

SignalPlane currently infers hosts from telemetry resource metadata:

```json
{
  "resource": {
    "host": "api-1"
  }
}
```

### Resource Metadata

Resource metadata attaches telemetry to the system that produced it.

Supported fields:

```json
{
  "service": "orders-api",
  "host": "orders-1",
  "environment": "production",
  "region": "local",
  "version": "0.1.0",
  "attributes": {
    "team": "checkout",
    "k8s.namespace": "shop"
  }
}
```

### Metrics

Metrics are numeric measurements over time.

Examples:

- Request count.
- Request latency.
- CPU usage.
- Memory usage.
- Queue depth.
- Job duration.
- Database query latency.

### Logs

Logs are timestamped events with severity and message text.

Supported severities:

- `debug`
- `info`
- `warning`
- `error`
- `fatal`

Error and fatal logs create warning or critical alerts.

### Traces

Traces describe a request or workflow across one or more spans.

A trace includes:

- Trace ID.
- Spans.
- Operation names.
- Duration.
- Service names.
- Status.
- Attributes.

Trace spans let SignalPlane show which services participated in a request.

### Alerts

Alerts represent a condition that needs attention.

In the current Silver preview, alerts are created automatically from:

- `error` logs.
- `fatal` logs.
- Error traces.
- High error-rate metrics.
- High latency metrics.
- Failed uptime checks.

Configurable metric and log alert rules run alongside the built-in Silver alert checks.

### Uptime Monitors

Uptime monitors store HTTP check definitions and the local SignalPlane process checks due monitors in the background. Each check records status, response time, status code, and consecutive failures.

## Request Flow

1. An app sends telemetry to `/api/ingest/logs`, `/api/ingest/metrics`, `/api/ingest/traces`, `/api/ingest/hosts`, or OTLP HTTP JSON paths under `/v1`.
2. SignalPlane checks the token.
3. SignalPlane normalizes resource metadata.
4. SignalPlane creates or updates services and hosts.
5. SignalPlane updates the runtime model used by the API and dashboard.
6. SignalPlane persists that runtime snapshot to JSON or PostgreSQL, depending on `SIGNALPLANE_STORE_BACKEND`.
7. SignalPlane evaluates built-in alerts and configured alert rules.
8. If `SIGNALPLANE_TELEMETRY_BACKEND=clickhouse`, SignalPlane archives metrics, logs, traces, spans, and uptime results into ClickHouse.
9. Failed ClickHouse writes are queued for replay when `SIGNALPLANE_TELEMETRY_REPLAY_PATH` is set.
10. The dashboard reads data through APIs such as `/api/bootstrap`, `/api/services`, `/api/logs`, and `/api/traces`. Telemetry reads use ClickHouse when available and fall back to runtime state.

## Security Model In Silver

Silver has local users, login sessions, and API tokens:

- `dev-token` is the default local bootstrap/admin token.
- `SIGNALPLANE_BOOTSTRAP_USER_EMAIL` and `SIGNALPLANE_BOOTSTRAP_USER_PASSWORD` create the first owner account.
- Tokens are persisted.
- Tokens have scopes: `admin`, `ingest`, or `read`.
- Users have roles: `owner`, `admin`, `editor`, or `viewer`.
- Ingestion endpoints require a valid token.
- Token management endpoints require admin access.
- Alert status updates, incident creation, and uptime monitor creation require admin access.

## Persistence Model In Silver

SignalPlane writes a runtime snapshot after state-changing operations.

Source runs default to an atomic JSON file so the preview stays simple and installable. In the full Podman stack, runtime state is stored in PostgreSQL and telemetry is archived into ClickHouse. Telemetry query APIs read from ClickHouse when configured.

## What Comes Next

The next Silver improvements should be:

- Uptime history and availability rollups.
- Real service detail pages.
- Better search and filtering.
- GitHub Actions CI.
- OTLP protobuf/gRPC compatibility.
