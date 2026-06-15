# SignalPlane Helm Chart

This chart deploys SignalPlane Silver API/UI replicas for on-prem or cloud Kubernetes.

## Production Shape

- StatefulSet with one durable replay PVC per replica.
- Service and headless service.
- ConfigMap for non-secret environment variables.
- Kubernetes Secret or existing secret for tokens and credentials.
- Readiness and liveness probes on `/healthz`.
- PodDisruptionBudget.
- Optional HorizontalPodAutoscaler.
- Optional Ingress.
- Optional NetworkPolicy.

Production installs should provide external HA PostgreSQL and ClickHouse. The chart does not install production database clusters.

## Install

```bash
kubectl create namespace signalplane
kubectl -n signalplane create secret generic signalplane-runtime \
  --from-literal=ingest-token='<replace-me>' \
  --from-literal=bootstrap-admin-token='<replace-me-or-empty>' \
  --from-literal=bootstrap-user-email='admin@customer.local' \
  --from-literal=bootstrap-user-password='<replace-me>' \
  --from-literal=postgres-url='postgres://signalplane:<password>@postgres.customer.local:5432/signalplane?sslmode=require' \
  --from-literal=postgres-password='<postgres-password>' \
  --from-literal=clickhouse-password='<clickhouse-password>'
helm upgrade --install signalplane deploy/helm/signalplane \
  --namespace signalplane \
  --values deploy/helm/signalplane/examples/values-onprem-production.yaml
```

## Local Template Check

```bash
helm template signalplane deploy/helm/signalplane \
  --namespace signalplane \
  --values deploy/helm/signalplane/examples/values-onprem-production.yaml
```

## Required Values

Update these for each customer:

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

Use a separate ingest token for collectors and an admin token or owner login for configuration. Leave `bootstrap-admin-token` empty when the customer does not want a long-lived bootstrap admin token.

When `secret.create=true`, the chart intentionally fails rendering unless these values are set:

- `secret.values.ingestToken`
- `secret.values.postgresURL`
- `secret.values.postgresPassword`
- `secret.values.clickhousePassword`
- Either `secret.values.bootstrapAdminToken`, or both `secret.values.bootstrapUserEmail` and `secret.values.bootstrapUserPassword` when `config.requireReadAuth=true`

When `secret.create=false`, `secret.existingSecret` is required and must contain all keys listed in the install command above.

Use [../../../docs/ON_PREM_DEPLOYMENT.md](../../../docs/ON_PREM_DEPLOYMENT.md) for the full deployment runbook.
