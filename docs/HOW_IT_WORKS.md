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

Future Silver work will add configurable alert rules.

### Uptime Monitors

Uptime monitors currently store monitor definitions. Scheduled execution is planned next.

## Request Flow

1. An app sends telemetry to `/api/ingest/logs`, `/api/ingest/metrics`, `/api/ingest/traces`, or `/api/ingest/hosts`.
2. SignalPlane checks the token.
3. SignalPlane normalizes resource metadata.
4. SignalPlane creates or updates services and hosts.
5. SignalPlane stores telemetry in memory.
6. SignalPlane writes a snapshot to disk.
7. The dashboard reads data through APIs such as `/api/bootstrap`, `/api/services`, `/api/logs`, and `/api/traces`.

## Security Model In Silver

The Silver preview has a simple token model:

- `dev-token` is the default local bootstrap/admin token.
- Tokens are persisted.
- Tokens have scopes: `admin`, `ingest`, or `read`.
- Ingestion endpoints require a valid token.
- Token management endpoints require admin access.
- Alert status updates, incident creation, and uptime monitor creation require admin access.

This is not a full user-login system yet.

## Persistence Model In Silver

SignalPlane writes an atomic JSON snapshot after state-changing operations.

This keeps the preview simple and installable. It is not intended as the final storage model for high-volume production telemetry.

## What Comes Next

The next Silver improvements should be:

- Configurable alert rules.
- Webhook/email notification channels.
- Scheduled uptime checks.
- Real service detail pages.
- Better search and filtering.
- GitHub Actions CI.
- OpenTelemetry ingestion.
