# SignalPlane Roadmap

## Current Baseline

The repository now contains a Silver self-hosted baseline:

- Single Go binary serving API and web UI.
- Podman Compose stack.
- File-backed JSON persistence and PostgreSQL-backed runtime persistence in the stack.
- ClickHouse telemetry archival and query APIs.
- Scoped API tokens.
- HTTP JSON and OTLP HTTP JSON/protobuf ingestion for metrics, logs, traces, and hosts.
- Inferred services and hosts.
- Built-in alerts, configurable metric/log alert rules, and notification channels.
- Incident records.
- Uptime monitor definitions and local checks.
- Helm chart for on-prem/cloud Kubernetes deployment.
- Example telemetry producers.
- Installation, operations, telemetry, API, on-prem, HA, air-gap, cloud capacity, and user docs.

This baseline supports demos and on-prem pilot design. The next work should harden release/upgrade, UI workflows, and normalized storage before expanding into Gold.

## Next Engineering Priorities

1. Add native OTLP gRPC ingestion while keeping the simple JSON API and OTLP HTTP support.
2. Add real authentication, session UI, and organization/user/role management.
3. Build proper explorer pages for logs, traces, metrics, services, hosts, alerts, incidents, and uptime.
4. Add configurable alert rules, notification channels, and uptime history.
5. Move production telemetry to durable queryable storage while keeping local file mode for quick demos.
6. Add CI, release artifacts, and basic performance tests.

## Phase 0: Product Foundation

- Finalize brand and product vocabulary.
- Finalize product requirements.
- Choose license.
- Choose MVP technical stack.
- Define contribution guidelines.
- Define repository structure.
- Create project board and issue templates.

## Phase 1: Silver MVP

- Podman Compose stack.
- Web UI shell.
- Local authentication.
- Organization and environment model.
- API tokens.
- OTLP traces.
- OTLP metrics.
- HTTP log ingestion.
- Linux node agent prototype.
- Metrics explorer.
- Logs explorer.
- Trace explorer.
- Service catalog.
- Host inventory.
- Dashboard builder.
- Static alerts.
- Email and webhook notifications.
- Basic incident records.
- Uptime checks.
- Quickstart documentation.

## Phase 2: Silver Hardening

- OpenAPI docs.
- Installer scripts.
- Sample app.
- Demo data.
- Retention settings.
- Data export.
- Audit events.
- Agent health.
- Ingestion health.
- Basic performance tests.
- Release packaging.

## Phase 3: Gold Cloud-Native

- Helm chart.
- Kubernetes operator.
- Kubernetes topology.
- Cloud metrics integrations.
- Integration marketplace.
- Advanced log pipelines.
- Service flow.
- Deployment events.
- SLOs and burn rates.
- Incident grouping.
- Maintenance windows.
- SSO.
- SCIM.
- Team RBAC.

## Phase 4: Gold Experience And Automation

- Real User Monitoring.
- API synthetics.
- Browser synthetics.
- Private synthetic locations.
- Workflow builder.
- Ticketing integrations.
- Paging integrations.
- CI/CD integrations.
- Cost controls.
- Cardinality controls.
- Continuous profiling preview.

## Phase 5: Platinum Data And Intelligence

- Unified telemetry lake.
- Signal Query.
- Signal Graph.
- Historical topology.
- AI-assisted incident summaries.
- Probable root cause suggestions.
- Natural-language query assistance.
- Predictive capacity signals.
- Advanced release validation.

## Phase 6: Platinum Enterprise

- Multi-tenancy.
- Data residency.
- Record-level controls.
- Legal hold.
- Runtime risk.
- Cloud and Kubernetes posture.
- Custom apps.
- Extension marketplace.
- Multi-region ingestion.
- Disaster recovery.
- Enterprise compliance reports.
