# SignalPlane User Guide

This guide explains how to use the Silver self-hosted baseline after installation.

## Open The Dashboard

Start SignalPlane, then open:

```text
http://127.0.0.1:4318
```

The dashboard has these areas:

- **Overview**: Counts for services, hosts, logs, traces, metrics, alerts, incidents, and uptime monitors.
- **Services**: Applications and dependencies inferred from telemetry.
- **Hosts**: Machines, process locations, or pod-like resources inferred from telemetry.
- **Logs**: Recent log events.
- **Traces**: Recent traces and spans.
- **Alerts**: Open alerts created from error telemetry.

The local Podman stack uses separate tokens:

- Ingest token for collectors and applications: `dev-token`
- Admin/read token for API queries and configuration: `dev-admin-token`

## First Things To Check

1. Confirm the health pill says `HEALTHY`.
2. Confirm demo services appear.
3. Send a test log.
4. Refresh the dashboard.
5. Confirm the new service appears.

## Send Your First Log

```bash
curl -X POST http://127.0.0.1:4318/api/ingest/logs \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "severity": "info",
    "message": "hello from my service",
    "resource": {
      "service": "my-service",
      "host": "my-host",
      "environment": "production"
    }
  }'
```

Refresh the dashboard. You should see:

- `my-service` in Services.
- `my-host` in Hosts.
- The log in Recent Logs.

## Create An Alert With An Error Log

```bash
curl -X POST http://127.0.0.1:4318/api/ingest/logs \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "severity": "error",
    "message": "database timeout",
    "traceId": "trace-example-1",
    "resource": {
      "service": "orders-api",
      "host": "orders-1",
      "environment": "production"
    }
  }'
```

SignalPlane will create:

- A log record.
- An inferred service.
- An inferred host.
- An open alert.

## View Services

Services are inferred from telemetry. A service appears when telemetry includes:

```json
{
  "resource": {
    "service": "orders-api"
  }
}
```

API:

```bash
curl -H "Authorization: Bearer dev-admin-token" http://127.0.0.1:4318/api/services
```

## View Hosts

Hosts are inferred from telemetry. A host appears when telemetry includes:

```json
{
  "resource": {
    "host": "orders-1"
  }
}
```

API:

```bash
curl -H "Authorization: Bearer dev-admin-token" http://127.0.0.1:4318/api/hosts
```

## View Logs

List recent logs:

```bash
curl -H "Authorization: Bearer dev-admin-token" http://127.0.0.1:4318/api/logs
```

Filter by service:

```bash
curl -H "Authorization: Bearer dev-admin-token" 'http://127.0.0.1:4318/api/logs?service=orders-api'
```

Filter by severity:

```bash
curl -H "Authorization: Bearer dev-admin-token" 'http://127.0.0.1:4318/api/logs?severity=error'
```

Search message text:

```bash
curl -H "Authorization: Bearer dev-admin-token" 'http://127.0.0.1:4318/api/logs?q=timeout'
```

## View Traces

List recent traces:

```bash
curl -H "Authorization: Bearer dev-admin-token" http://127.0.0.1:4318/api/traces
```

Filter by service:

```bash
curl -H "Authorization: Bearer dev-admin-token" 'http://127.0.0.1:4318/api/traces?service=orders-api'
```

Filter by status:

```bash
curl -H "Authorization: Bearer dev-admin-token" 'http://127.0.0.1:4318/api/traces?status=error'
```

## View Alerts

```bash
curl -H "Authorization: Bearer dev-admin-token" http://127.0.0.1:4318/api/alerts
```

Acknowledge an alert:

```bash
curl -X PATCH http://127.0.0.1:4318/api/alerts/ALERT_ID \
  -H "Authorization: Bearer dev-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"status": "acknowledged"}'
```

Resolve an alert:

```bash
curl -X PATCH http://127.0.0.1:4318/api/alerts/ALERT_ID \
  -H "Authorization: Bearer dev-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"status": "resolved"}'
```

## Create An Incident

```bash
curl -X POST http://127.0.0.1:4318/api/incidents \
  -H "Authorization: Bearer dev-admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Checkout degradation",
    "severity": "warning",
    "owner": "platform",
    "affectedServices": ["orders-api"]
  }'
```

## Register An Uptime Monitor

```bash
curl -X POST http://127.0.0.1:4318/api/uptime-monitors \
  -H "Authorization: Bearer dev-admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Orders API health",
    "url": "http://localhost:8080/healthz",
    "method": "GET",
    "expectedStatus": 200,
    "intervalSeconds": 60,
    "timeoutSeconds": 5
  }'
```

SignalPlane checks due monitors in the background and records the latest status, response code, latency, and failure count.

## Run The Example Applications

SignalPlane includes test applications for common workloads:

```bash
./examples/test-applications/run_all.sh
```

This sends telemetry from:

- Go backend API.
- Node microservice.
- Python web backend.
- Python worker.
- Database simulator.
- C host probe.
- Kubernetes-style workload metadata.
- Uptime target monitor.

## Current Limitations

Silver still has these product-interface gaps:

- Uptime history and availability rollups.
- Saved dashboards.
- Dedicated explorer pages for each signal.
- Runtime retention settings UI.
- Native OTLP gRPC compatibility.
- Login, alert-rule, and notification-channel forms in the web UI; the APIs are available.
