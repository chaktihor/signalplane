# SignalPlane Architecture

## 1. Architecture Goals

- Use open telemetry standards.
- Keep the local developer stack easy to run.
- Separate ingestion, processing, storage, querying, alerting, and UI concerns.
- Support self-hosted and managed deployment models.
- Allow storage backends to evolve without breaking the product model.
- Make data governance and cost controls part of the ingestion path.

## 2. Logical Components

| Component | Responsibility |
|---|---|
| Web UI | Product interface for exploration, dashboards, incidents, SLOs, automation, governance, and admin |
| API Service | Authenticated product API, RBAC, metadata, saved objects, admin operations |
| Ingestion Gateway | OTLP, log, metric, event, and synthetic result ingestion |
| Collector Fleet | Edge collection, enrichment, buffering, and forwarding |
| Stream Processor | Parse, enrich, mask, sample, route, correlate, and evaluate near-real-time rules |
| Query Service | Unified read API for dashboards, explorers, alerts, and reports |
| Alert Engine | Rule evaluation, state transitions, deduplication, and notification dispatch |
| Incident Service | Incident records, timelines, ownership, affected entities, and response state |
| Topology Service | Entity discovery, dependency graph, ownership graph, and historical topology |
| Workflow Engine | Triggered automations, approvals, retries, integrations, and audit trail |
| Identity Service | Users, teams, roles, policies, tokens, SSO, SCIM, and audit |
| Storage Layer | Metrics, logs, traces, events, metadata, topology, archives, and object storage |

## 3. Suggested MVP Stack

| Layer | Suggested Default |
|---|---|
| Frontend | React or Next.js |
| API | Go, Rust, or TypeScript service |
| Collector | OpenTelemetry Collector distribution plus SignalPlane extensions |
| Metadata | PostgreSQL |
| Metrics | Prometheus-compatible backend or ClickHouse |
| Logs | ClickHouse or OpenSearch |
| Traces | ClickHouse or Tempo-compatible backend |
| Queue | NATS, Kafka, or Redpanda |
| Cache | Redis-compatible cache |
| Object Storage | S3-compatible storage |
| Deployment | Docker Compose first, Helm second |

## 4. Ingestion Flow

1. Agents, collectors, SDKs, integrations, and APIs send telemetry to the ingestion gateway.
2. The ingestion gateway authenticates tokens, enforces payload limits, and validates protocol shape.
3. The stream processor enriches telemetry with ownership, environment, deployment, cloud, and Kubernetes metadata.
4. Sensitive data rules mask, drop, or route records before storage.
5. Data is written to the appropriate storage tier.
6. Derived signals update service health, topology, alerts, SLOs, and incident state.

## 5. Query Flow

1. UI requests charts, tables, traces, logs, topology, or incident context from the query service.
2. Query service checks authorization and data scope.
3. Query service plans requests across storage backends.
4. Query service applies limits, pagination, and result shaping.
5. UI renders results with links to related telemetry and entities.

## 6. Alert Flow

1. Alert rules are evaluated on schedules or streaming windows.
2. Alert engine creates or updates alert state.
3. Incident service groups related alerts where enabled.
4. Notification routing determines destinations.
5. Workflow engine executes configured automations.
6. Audit events are recorded for state changes and actions.

## 7. Security Boundaries

- Ingestion tokens should be scoped to organization and environment.
- User tokens should be scoped by role and policy.
- Secrets should be stored only in a dedicated secrets provider.
- Sensitive data masking should happen before durable storage whenever possible.
- Audit events should be append-only.
- Workflow execution should use scoped credentials.

## 8. Extension Points

- Collector receivers, processors, and exporters.
- Log parsers and enrichment rules.
- Technology integrations.
- Dashboard templates.
- Alert rule templates.
- Workflow actions.
- Custom apps.
- Query functions.

## 9. Deployment Modes

### 9.1 Local

- Docker Compose.
- Single organization.
- Default local credentials.
- Sample data option.

### 9.2 Team

- Single Kubernetes cluster.
- External metadata database.
- Persistent analytics storage.
- SSO optional.

### 9.3 Enterprise

- Multi-cluster.
- High availability.
- Regional ingestion.
- Tenant isolation.
- External identity provider.
- Object storage archive.
- Disaster recovery.

## 10. Architectural Risks

- Storage choice may constrain query language and cross-signal analytics.
- High-cardinality telemetry can create cost and performance issues.
- AI-assisted features require trustworthy topology and clean metadata.
- Session replay and logs introduce privacy risks.
- Workflow automation requires strong permission and approval boundaries.

