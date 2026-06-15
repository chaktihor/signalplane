# SignalPlane Silver Demo Runbook

This runbook prepares a local demo with SignalPlane observing live applications through both direct telemetry and a local log-agent collector.

## Full Podman Silver Stack

For the strongest demo, start the full stack:

```bash
make stack-up
```

This starts SignalPlane, PostgreSQL, ClickHouse, the OpenTelemetry gateway
collector, the node-local log-agent collector, a demo log writer, and Mailpit.

Open:

```text
http://127.0.0.1:4318
```

The `agent-checkout-api` service should appear after the log agent tails and
forwards the demo app log file. Follow the agent logs if needed:

```bash
make stack-agent-logs
```

Verify from the API:

```bash
curl 'http://127.0.0.1:4318/api/logs?service=agent-checkout-api&limit=5'
```

## Terminal 1: Start SignalPlane

```bash
SIGNALPLANE_DATA_PATH=data/demo-signalplane.json make run
```

Open:

```text
http://127.0.0.1:4318
```

Expected:

- Health pill is `HEALTHY`.
- Demo seed data appears immediately.
- Counts update every few seconds.

## Terminal 2: Start Demo Shop

```bash
make demo-shop
```

Open:

```text
http://127.0.0.1:8088
```

The demo shop automatically sends telemetry to SignalPlane every few seconds. It also registers an uptime monitor named `Demo shop health`.

## Terminal 3: Generate A Demo Burst

```bash
make demo-traffic
```

This sends successful and failed checkout events from the demo application.

## Demo Talking Points

1. **Service discovery**: `demo-checkout-api`, `inventory-service`, `payment-gateway`, and `orders-postgres` appear from telemetry resource metadata.
2. **Host inventory**: `demo-checkout-1` appears with heartbeat CPU and memory values.
3. **Agent-first logs**: `agent-checkout-api` logs are collected by a node-local agent from a local log file, enriched, batched, retried, and forwarded through the gateway collector.
4. **Metrics**: request count, duration, error rate, revenue, orders, and failures appear in recent metrics.
5. **Traces**: checkout traces show the root checkout operation plus inventory, payment, and database spans.
6. **Alerts**: failed checkouts produce error-log, error-trace, high-latency, and high-error-rate alerts.
7. **Uptime**: `Demo shop health` checks the app health endpoint and records latest status/latency.

## Reset Demo Data

Stop SignalPlane, then remove the demo data file:

```bash
rm -f data/demo-signalplane.json
```

Restart SignalPlane and the demo shop.

## Useful URLs

- SignalPlane: `http://127.0.0.1:4318`
- Demo shop: `http://127.0.0.1:8088`
- Successful checkout: `http://127.0.0.1:8088/checkout`
- Failed checkout: `http://127.0.0.1:8088/checkout?fail=true`
- Traffic burst: `http://127.0.0.1:8088/traffic?count=12&failEvery=4`
