# SignalPlane Silver On-Prem Deployment

This guide is for customer infrastructure, platform, and operations teams installing SignalPlane Silver inside a private data center or private cloud.

## Deployment Modes

| Mode | Use case | Dependencies |
|---|---|---|
| Podman stack | Laptop demo, sales demo, functional validation | Local containers for SignalPlane, PostgreSQL, ClickHouse, OpenTelemetry Collector, Mailpit |
| Kubernetes with external services | Production on-prem baseline | Customer Kubernetes, HA PostgreSQL, HA ClickHouse, SMTP, private registry, storage class |
| Kubernetes lab bundle | Internal lab or pre-production | Customer Kubernetes plus non-HA database/ClickHouse services supplied by the platform team |

For production, use the Kubernetes chart with external HA PostgreSQL and ClickHouse. The chart deploys SignalPlane API/UI replicas, health probes, a service, optional ingress, PDB, optional HPA, network policy, secrets, and one durable replay PVC per SignalPlane pod.

## Capacity Baseline

Use this as the first Silver production sizing target:

| Component | Baseline |
|---|---:|
| SignalPlane API/UI | 3 replicas, 2 vCPU and 2 GB RAM requested per replica, 4 vCPU and 4 GB RAM limit |
| SignalPlane replay disk | 20 GB fast RWO PVC per replica |
| PostgreSQL | HA primary/standby or cluster, 4 vCPU, 16 GB RAM, 250 GB SSD |
| ClickHouse | 3 nodes, 16 vCPU, 64-128 GB RAM, 2 TB fast SSD per node |
| OpenTelemetry Collectors | 3 replicas, 4 vCPU and 16 GB RAM each for shared intake |
| Object/archive storage | 10 TB initial archive and backup bucket/share |
| Hot retention | 30 days in ClickHouse |
| Archive retention | 180 days in object storage |

Scale ClickHouse first when telemetry volume or retention increases. Scale SignalPlane API replicas when UI/API traffic, alert-rule evaluation, or ingestion concurrency increases.

## Required Customer Services

Production needs these on-prem services before installation:

- Kubernetes 1.29+ or OpenShift 4.14+.
- Private container registry such as Harbor, Nexus, Artifactory, or OpenShift internal registry.
- HA PostgreSQL 16+ or a customer-approved PostgreSQL operator.
- HA ClickHouse 24.8+ or a customer-approved ClickHouse operator.
- ReadWriteOnce storage class for SignalPlane replay PVCs.
- Fast SSD/NVMe-backed storage class for ClickHouse.
- Ingress controller or route layer such as NGINX, HAProxy, OpenShift Routes, F5, or customer load balancer.
- TLS certificate from internal CA or cert-manager.
- SMTP relay or approved webhook path for notifications.
- Object storage or NAS/SAN target for ClickHouse backups and telemetry archive.
- Backup tooling and restore process owned by the platform team.

## Install Locally With Podman

Use this path for a complete laptop demo:

```bash
make stack-up
```

Open:

```text
http://127.0.0.1:4318
```

Default local credentials:

```text
email: admin@signalplane.local
password: admin-password
ingest token: dev-token
admin token: dev-admin-token
```

The Podman stack starts:

- SignalPlane API/UI.
- PostgreSQL for runtime state.
- ClickHouse for telemetry archival.
- OpenTelemetry Collector for OTLP intake.
- Mailpit for email notification testing.

The local collector listens on OTLP gRPC `127.0.0.1:4317` and OTLP HTTP `127.0.0.1:4319`, then forwards OTLP HTTP protobuf payloads to SignalPlane.

## Install On Kubernetes With Helm

Create a namespace:

```bash
kubectl create namespace signalplane
```

Create the runtime secret:

```bash
kubectl -n signalplane create secret generic signalplane-runtime \
  --from-literal=ingest-token='<replace-with-random-token>' \
  --from-literal=bootstrap-admin-token='<replace-with-random-token-or-empty>' \
  --from-literal=bootstrap-user-email='admin@customer.local' \
  --from-literal=bootstrap-user-password='<replace-with-random-password>' \
  --from-literal=postgres-url='postgres://signalplane:<password>@postgres.customer.local:5432/signalplane?sslmode=require' \
  --from-literal=postgres-password='<postgres-password>' \
  --from-literal=clickhouse-password='<clickhouse-password>'
```

Review and copy the production values file:

```bash
cp deploy/helm/signalplane/examples/values-onprem-production.yaml /tmp/signalplane-values.yaml
```

Edit:

- `image.repository`
- `image.tag`
- `image.pullSecrets`
- `ingress.hosts`
- `ingress.tls`
- `persistence.storageClassName`
- `config.clickhouseURL`
- `config.clickhouseHTTPURL`
- `config.smtpAddr`
- `config.notificationFrom`
- `secret.existingSecret`

Install:

```bash
helm upgrade --install signalplane deploy/helm/signalplane \
  --namespace signalplane \
  --values /tmp/signalplane-values.yaml
```

Check rollout:

```bash
kubectl -n signalplane rollout status statefulset/signalplane
kubectl -n signalplane get pods,svc,ingress,pvc
```

Port-forward if ingress is not ready:

```bash
kubectl -n signalplane port-forward svc/signalplane 4318:4318
```

Check health:

```bash
curl http://127.0.0.1:4318/healthz
curl -H "Authorization: Bearer <admin-or-read-token>" http://127.0.0.1:4318/api/system/dependencies
```

## What Happens Under The Hood

1. Applications send JSON ingestion or OTLP HTTP telemetry to SignalPlane, or to an OpenTelemetry Collector that forwards to SignalPlane.
2. SignalPlane validates the API token.
3. Resource metadata is normalized into service, host, environment, region, version, and custom attributes.
4. Runtime state is persisted in PostgreSQL.
5. Telemetry is archived to ClickHouse.
6. If ClickHouse is unavailable, the SignalPlane pod appends failed writes to its `/data/telemetry-replay.jsonl` PVC.
7. The replay loop retries the spooled telemetry every 10 seconds.
8. Query APIs read from ClickHouse and fall back to runtime state if ClickHouse is unavailable.
9. Alert rules evaluate incoming metrics/logs/traces and create alerts.
10. Notification channels deliver email, webhook, or Slack-compatible messages.

## Logs, Metrics, And Traces

SignalPlane accepts:

- Native JSON: `/api/ingest/logs`, `/api/ingest/metrics`, `/api/ingest/traces`, `/api/ingest/hosts`.
- OTLP HTTP JSON: `/v1/logs`, `/v1/metrics`, `/v1/traces`.
- OTLP HTTP protobuf: `/v1/logs`, `/v1/metrics`, `/v1/traces`.

For production collectors, use OTLP HTTP protobuf to SignalPlane:

```yaml
exporters:
  otlphttp/signalplane:
    endpoint: https://signalplane.customer.local
    headers:
      Authorization: Bearer ${SIGNALPLANE_INGEST_TOKEN}
```

OTLP gRPC ingestion is not yet native in SignalPlane. Use the OpenTelemetry Collector as the gRPC receiver and forward to SignalPlane over OTLP HTTP.

## Security Checklist

- Replace every default token and password.
- Use TLS for ingress.
- Set `SIGNALPLANE_SECURE_COOKIES=true` for HTTPS deployments.
- Set `SIGNALPLANE_COOKIE_DOMAIN` to the production UI hostname if the customer requires an explicit cookie domain.
- Use `sslmode=require` or stronger for PostgreSQL.
- Keep ClickHouse behind the cluster network or private customer network.
- Store runtime secrets in Kubernetes Secrets, Vault, External Secrets, or Sealed Secrets.
- Give collectors ingest-scoped tokens, not admin tokens.
- Restrict ingress with network policy and customer firewall rules.
- Enable registry image scanning.
- Back up PostgreSQL, ClickHouse, and the replay PVC storage class according to customer policy.

## Upgrade

1. Push the new image to the private registry.
2. Update `image.tag` in the values file.
3. Run:

```bash
helm upgrade signalplane deploy/helm/signalplane \
  --namespace signalplane \
  --values /tmp/signalplane-values.yaml
```

4. Watch rollout and dependency health.

```bash
kubectl -n signalplane rollout status statefulset/signalplane
curl https://signalplane.customer.local/api/system/dependencies
```

## Uninstall

Remove the application:

```bash
helm uninstall signalplane --namespace signalplane
```

PVCs are retained by Kubernetes. Delete them only after a backup and explicit approval:

```bash
kubectl -n signalplane get pvc
```
