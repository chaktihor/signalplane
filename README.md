# SignalPlane

SignalPlane is an open-source observability, reliability, security, digital experience, runtime risk, and automation platform for modern software systems.

The product vision is to give teams a single place to understand telemetry, system topology, user experience, incidents, service reliability, runtime risk, and operational automation without locking data or workflows into a proprietary ecosystem.

The intent is to build a Dynatrace-equivalent product in three tiers:

- **Silver**: self-hosted observability MVP for small teams.
- **Gold**: production-grade cloud-native observability, reliability, and automation.
- **Platinum**: enterprise-scale telemetry intelligence, runtime risk, governance, and extensibility.

## Core Principles

- **Open by default**: Built around OpenTelemetry, open schemas, open APIs, and portable deployment patterns.
- **Signal over noise**: Correlate telemetry into actionable incidents instead of flooding teams with disconnected alerts.
- **Topology-aware**: Connect services, hosts, containers, Kubernetes resources, cloud dependencies, releases, SLOs, users, and business flows.
- **Operator-friendly**: Make onboarding, debugging, alerting, and automation practical for real teams under pressure.
- **Extensible**: Support plugins, integrations, custom collectors, custom dashboards, custom workflows, and custom apps.
- **Privacy-conscious**: Treat logs, user sessions, secrets, and personally identifiable data as governed data from day one.

## Repository Contents

- [Installation](docs/INSTALLATION.md)
- [How SignalPlane Works](docs/HOW_IT_WORKS.md)
- [User Guide](docs/USER_GUIDE.md)
- [Telemetry Guide](docs/TELEMETRY_GUIDE.md)
- [API Reference](docs/API_REFERENCE.md)
- [Operations Guide](docs/OPERATIONS.md)
- [On-Prem Deployment](docs/ON_PREM_DEPLOYMENT.md)
- [HA Architecture](docs/HA_ARCHITECTURE.md)
- [Air-Gapped Install](docs/AIR_GAPPED_INSTALL.md)
- [Cloud Capacity Planning](docs/CLOUD_CAPACITY_PLANNING.md)
- [Silver Quickstart](docs/SILVER_QUICKSTART.md)
- [Silver Demo Runbook](docs/SILVER_DEMO_RUNBOOK.md)
- [Silver Readiness](docs/SILVER_READINESS.md)
- [Product Strategy](docs/PRODUCT_STRATEGY.md)
- [Project Structure](docs/PROJECT_STRUCTURE.md)
- [Product Requirements](docs/PRODUCT_REQUIREMENTS.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Roadmap](docs/ROADMAP.md)
- [Brand](docs/BRAND.md)
- [Contributing](CONTRIBUTING.md)

## Initial Product Tiers

| Tier | Audience | Goal |
|---|---|---|
| **Silver** | Small teams and early adopters | Core metrics, logs, traces, dashboards, uptime checks, and threshold alerts |
| **Gold** | Production engineering teams | Kubernetes, cloud integrations, RUM, synthetics, SLOs, correlation, workflows, and governance |
| **Platinum** | Large enterprises and platforms | Unified telemetry lake, AI-assisted diagnosis, runtime security, release validation, custom apps, and multi-tenant governance |

## Status

SignalPlane now has a Silver self-hosted product baseline in this repository: a runnable Go service, embedded dashboard, HTTP JSON and OTLP HTTP JSON/protobuf telemetry ingestion, login sessions, scoped tokens, PostgreSQL-backed runtime persistence in the Podman stack, ClickHouse telemetry archival and query APIs, alert rules, notification channels, replay handling, Helm deployment assets, and example telemetry producers.

The current implementation is suitable for local demos, on-prem pilots, and production-design validation. Larger production rollouts should still add normalized PostgreSQL repositories, richer UI forms, native OTLP gRPC ingestion, and release/upgrade automation.

## Silver Self-Hosted Baseline

The first Silver slice is a Go service that serves both the API and web UI.

Install and run from source:

```bash
make run
```

Then open `http://127.0.0.1:4318`.

For containerized local installs with Podman:

```bash
make stack-up
```

Silver can persist API/dashboard runtime state to either an atomic JSON snapshot or PostgreSQL. Source runs default to JSON at `data/signalplane.json`; Podman Compose runs with PostgreSQL for runtime state and archives incoming telemetry into ClickHouse.

Start here:

1. [Installation](docs/INSTALLATION.md)
2. [How SignalPlane Works](docs/HOW_IT_WORKS.md)
3. [User Guide](docs/USER_GUIDE.md)
4. [Telemetry Guide](docs/TELEMETRY_GUIDE.md)
5. [API Reference](docs/API_REFERENCE.md)

## What Works Today

- Single Go binary serving API and UI.
- Podman Compose install.
- Kubernetes Helm chart for on-prem/cloud deployment.
- JSON or PostgreSQL-backed runtime persistence.
- ClickHouse telemetry archival and query APIs for the full local stack.
- Durable telemetry replay queue for failed ClickHouse writes.
- Local users, login sessions, roles, and persisted scoped API tokens.
- Metrics ingestion through SignalPlane JSON, OTLP HTTP JSON, and OTLP HTTP protobuf.
- Log ingestion through SignalPlane JSON, OTLP HTTP JSON, and OTLP HTTP protobuf.
- Trace ingestion through SignalPlane JSON, OTLP HTTP JSON, and OTLP HTTP protobuf.
- Host ingestion.
- Service and host inference.
- Error-log and error-trace alert creation.
- Configurable metric/log alert rules.
- Email, generic webhook, and Slack-compatible webhook notification channels.
- Incident records.
- Uptime monitor definitions and local uptime checks.
- Demo checkout application that continuously emits logs, metrics, traces, host telemetry, and uptime registration.
- Example apps for Go, Node.js, Python, C, database/dependency, worker, and Kubernetes-style metadata.

## Run The Silver Demo

Run the full local platform stack with PostgreSQL, ClickHouse, OpenTelemetry Collector, Mailpit, and SignalPlane:

```bash
make stack-up
```

For a fast single-process demo without dependency containers:

Start SignalPlane:

```bash
SIGNALPLANE_DATA_PATH=data/demo-signalplane.json make run
```

In another terminal, start the observed demo application:

```bash
make demo-shop
```

Generate a visible traffic burst:

```bash
make demo-traffic
```

See [Silver Demo Runbook](docs/SILVER_DEMO_RUNBOOK.md) for the full demo sequence.

See [Silver Readiness](docs/SILVER_READINESS.md) for the remaining work before this should be called a full Silver product release.

## Send First Telemetry

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

Run the full example suite:

```bash
./examples/test-applications/run_all.sh
```
