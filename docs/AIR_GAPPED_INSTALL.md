# SignalPlane Silver Air-Gapped Install

Use this guide when the customer environment cannot pull from the internet.

## Required Artifacts

Prepare these outside the air-gapped network:

- SignalPlane container image.
- PostgreSQL image if the customer does not provide PostgreSQL separately.
- ClickHouse image if the customer does not provide ClickHouse separately.
- OpenTelemetry Collector image if collectors are deployed by this project team.
- Helm chart directory `deploy/helm/signalplane`.
- Production values file.
- Kubernetes runtime secret manifest or sealed secret.
- ClickHouse schema SQL from `deploy/clickhouse/init`.
- PostgreSQL schema SQL from `deploy/postgres/init`.
- Release notes and checksum manifest.

Current local stack image inventory:

| Component | Image |
|---|---|
| SignalPlane | Built from `Containerfile` |
| PostgreSQL | `postgres:16-alpine` |
| ClickHouse | `clickhouse/clickhouse-server:24.8-alpine` |
| OpenTelemetry Collector | `otel/opentelemetry-collector-contrib:0.111.0` |
| Mailpit | `axllent/mailpit:v1.21` |

Mailpit is for demos and should not be part of production.

## Build And Export Images

Build SignalPlane:

```bash
podman build -t signalplane:0.1.0 -f Containerfile .
```

Pull dependency images:

```bash
podman pull postgres:16-alpine
podman pull clickhouse/clickhouse-server:24.8-alpine
podman pull otel/opentelemetry-collector-contrib:0.111.0
```

Save images:

```bash
podman save -o signalplane-images.tar \
  signalplane:0.1.0 \
  postgres:16-alpine \
  clickhouse/clickhouse-server:24.8-alpine \
  otel/opentelemetry-collector-contrib:0.111.0
```

Create checksums:

```bash
shasum -a 256 signalplane-images.tar > signalplane-images.tar.sha256
```

Transfer:

- `signalplane-images.tar`
- `signalplane-images.tar.sha256`
- `deploy/helm/signalplane`
- production values file
- release notes

## Import Into Customer Registry

Inside the air-gapped network:

```bash
shasum -a 256 -c signalplane-images.tar.sha256
podman load -i signalplane-images.tar
```

Retag for the customer registry:

```bash
podman tag signalplane:0.1.0 registry.customer.local/observability/signalplane:0.1.0
podman tag postgres:16-alpine registry.customer.local/base/postgres:16-alpine
podman tag clickhouse/clickhouse-server:24.8-alpine registry.customer.local/base/clickhouse-server:24.8-alpine
podman tag otel/opentelemetry-collector-contrib:0.111.0 registry.customer.local/base/opentelemetry-collector-contrib:0.111.0
```

Push:

```bash
podman push registry.customer.local/observability/signalplane:0.1.0
podman push registry.customer.local/base/postgres:16-alpine
podman push registry.customer.local/base/clickhouse-server:24.8-alpine
podman push registry.customer.local/base/opentelemetry-collector-contrib:0.111.0
```

## Install

Create or verify the image pull secret:

```bash
kubectl -n signalplane create secret docker-registry customer-registry-pull \
  --docker-server=registry.customer.local \
  --docker-username='<user>' \
  --docker-password='<password>'
```

Create the runtime secret:

```bash
kubectl -n signalplane create secret generic signalplane-runtime \
  --from-literal=ingest-token='<random-token>' \
  --from-literal=bootstrap-user-email='admin@customer.local' \
  --from-literal=bootstrap-user-password='<random-password>' \
  --from-literal=postgres-url='postgres://signalplane:<password>@postgres.customer.local:5432/signalplane?sslmode=require' \
  --from-literal=postgres-password='<postgres-password>' \
  --from-literal=clickhouse-password='<clickhouse-password>'
```

Install:

```bash
helm upgrade --install signalplane ./deploy/helm/signalplane \
  --namespace signalplane \
  --values values-onprem-production.yaml
```

## Offline Upgrade

1. Build and export the new images.
2. Transfer the image tarball and chart changes.
3. Import and push to the customer registry.
4. Update only `image.tag` and required config changes.
5. Run `helm upgrade`.
6. Keep the previous image tag in the registry until rollback is no longer needed.

Rollback:

```bash
helm rollback signalplane --namespace signalplane
```

## Air-Gap Validation

Before the customer demo, verify:

- Pods use only customer registry image references.
- Kubernetes events show no internet pull attempts.
- Ingress TLS certificate is trusted on the customer network.
- `/healthz` returns `ok`.
- `/api/system/dependencies` shows PostgreSQL and ClickHouse as up.
- A collector can send OTLP HTTP protobuf telemetry.
- An alert rule can trigger a notification through the approved SMTP/webhook path.
- PostgreSQL and ClickHouse backups complete successfully.
