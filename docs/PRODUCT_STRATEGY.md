# SignalPlane Product Strategy

SignalPlane is intended to become an open, self-hostable observability platform comparable in scope to Dynatrace, with a staged delivery model that keeps the first product useful while leaving room for enterprise-grade capabilities.

## Product Thesis

Modern teams do not need more isolated telemetry screens. They need a command plane that connects metrics, logs, traces, topology, incidents, reliability, user experience, runtime risk, and automation into one operational workflow.

SignalPlane should compete on:

- OpenTelemetry-first collection and portable deployment.
- Strong topology and ownership model.
- Fast incident investigation across metrics, logs, traces, hosts, services, releases, and users.
- Practical alerting, SLOs, workflows, and governance.
- Transparent, explainable AI assistance in later tiers.
- Extensible APIs, integrations, collectors, dashboards, workflows, and apps.

## Silver

Silver is the self-hosted MVP for small teams and early adopters.

Core outcomes:

- Install locally with Docker Compose or a single binary.
- Send logs, metrics, traces, and host telemetry quickly.
- See inferred services and hosts.
- Search recent logs and traces.
- Create and manage basic alerts, incidents, tokens, and uptime monitors.
- Use a simple dashboard to understand local system health.

Current repository status:

- Single Go service.
- Embedded web dashboard.
- HTTP JSON ingestion.
- Scoped API tokens.
- File-backed JSON persistence.
- Demo seed data and sample telemetry emitters.

Major Silver gaps:

- Full authentication and user/session UI.
- Configurable alert rules.
- Uptime history and availability rollups.
- Notification delivery.
- OpenTelemetry OTLP ingestion.
- Real dashboard builder and explorer pages.
- Production-grade storage.

## Gold

Gold is the production cloud-native tier for engineering, SRE, platform, and reliability teams.

Target capabilities:

- Helm-based Kubernetes deployment.
- Collector fleet and remote configuration.
- Kubernetes topology and workload health.
- Cloud integrations for AWS, Azure, and Google Cloud.
- Advanced logs, traces, APM, RUM, synthetics, SLOs, and burn-rate alerts.
- Incident grouping, deduplication, deployment correlation, and ownership routing.
- Workflow automation with approvals.
- SSO, SCIM, team RBAC, audit logs, and cost controls.

## Platinum

Platinum is the enterprise intelligence tier.

Target capabilities:

- Unified telemetry lake across observability, topology, security, user experience, and business events.
- Signal Query language and Signal Graph topology.
- AI-assisted anomaly detection, incident summaries, probable cause, and remediation suggestions.
- Runtime vulnerability prioritization and cloud/Kubernetes posture.
- Release validation and reliability gates.
- Custom apps, extension marketplace, multi-tenancy, data residency, record-level controls, and disaster recovery.

## Product Principle

Do not build Platinum abstractions before Silver workflows are real. The near-term product should prove ingestion, exploration, alerting, service context, and operational response with a small self-hosted system. Each later tier should deepen those workflows rather than become a separate product.
