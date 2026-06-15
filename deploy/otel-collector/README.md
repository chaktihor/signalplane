# SignalPlane Collector Profiles

SignalPlane Silver uses collectors in two roles.

## Gateway Collector

`config.yaml` is the gateway profile. It receives OTLP gRPC/HTTP from applications,
SDKs, and local agents, then forwards OTLP HTTP to SignalPlane with the ingest
token.

The gateway profile does not store the ingest token in YAML. Set
`SIGNALPLANE_INGEST_TOKEN` in the collector environment and source it from a
secret manager, Kubernetes Secret, or local `.env` file:

```bash
SIGNALPLANE_INGEST_TOKEN='<collector-ingest-token>' podman compose up otel-collector
```

For the local Podman demo, `docker-compose.yml` falls back to `dev-token` when
`SIGNALPLANE_INGEST_TOKEN` is not set. Production deployments should create a
dedicated ingest-scoped token per collector group and rotate it without editing
collector config files.

Local ports:

| Protocol | Port |
|---|---:|
| OTLP gRPC | `4317` |
| OTLP HTTP | `4319` |
| Collector metrics | `8888` |

## Node Log Agent

`agent-config.yaml` is the node-local log-agent profile. It tails application log
files under `/var/log/signalplane-apps/*.log`, parses newline-delimited JSON,
adds resource metadata, batches records, queues them on local disk, retries
failed sends, compresses OTLP traffic, and forwards to the gateway collector.

The local Podman stack mounts a shared `app-logs` volume into:

- `demo-log-writer` at `/var/log/signalplane-apps`
- `signalplane-log-agent` at `/var/log/signalplane-apps:ro`

Production deployments should run this profile as a node agent or Kubernetes
DaemonSet close to the application workloads. Applications should write logs to
stdout, container logs, or local files; the agent should own batching,
backpressure, retry, enrichment, and forwarding.

## JSON Log Contract

The local filelog profile expects one JSON object per line:

```json
{
  "timestamp": "2026-06-15T14:30:00Z",
  "severity": "info",
  "message": "checkout completed",
  "traceId": "00000000000000000000000000000001",
  "spanId": "0000000000000001",
  "service": "agent-checkout-api",
  "host": "demo-node-1",
  "environment": "production",
  "region": "local",
  "version": "agent-demo-0.1.0",
  "orderId": "ord-agent-000001"
}
```

`service`, `host`, `environment`, `region`, and `version` become OpenTelemetry
resource attributes before the record leaves the node agent.
