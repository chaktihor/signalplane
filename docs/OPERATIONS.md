# SignalPlane Operations Guide

This guide covers day-to-day operation of the Silver developer preview.

## Start And Stop

Run from source:

```bash
make run
```

Build and run:

```bash
make build
./bin/signalplane
```

Run the full local stack:

```bash
make stack-up
```

Stop the stack:

```bash
make stack-down
```

Follow stack logs:

```bash
make stack-logs
```

## Data Location

Default local data path:

```text
data/signalplane.json
```

Docker data path inside the container:

```text
/data/signalplane.json
```

Docker Compose volumes:

```text
postgres-data
clickhouse-data
signalplane-data
```

## Backup

For local source runs, copy:

```bash
cp data/signalplane.json signalplane-backup.json
```

For Docker Compose, copy from the running container or use a volume backup workflow.

## Restore

For local source runs:

```bash
mkdir -p data
cp signalplane-backup.json data/signalplane.json
make run
```

For custom paths:

```bash
SIGNALPLANE_DATA_PATH=/path/to/signalplane-backup.json make run
```

## Reset Local Data

Stop SignalPlane, then remove:

```bash
rm -rf data
```

For Docker Compose:

```bash
make stack-reset
```

## Change The Bootstrap Token

Set:

```bash
SIGNALPLANE_INGEST_TOKEN=change-me make run
```

The bootstrap token is persisted as an admin token. If the data file already exists, SignalPlane adds the configured bootstrap token if it is not already present.

## Disable Demo Data

```bash
SIGNALPLANE_SEED_DEMO_DATA=false make run
```

This only affects first initialization. Existing data files are loaded as-is.

## Expose On A Network

```bash
SIGNALPLANE_ADDR=0.0.0.0:4318 make run
```

For Docker Compose, this is already set inside the container.

## Health Checks

```bash
curl http://127.0.0.1:4318/healthz
```

Check configured platform dependencies:

```bash
curl http://127.0.0.1:4318/api/system/dependencies
```

## Troubleshooting

### Port Already In Use

Run on a different port:

```bash
SIGNALPLANE_ADDR=127.0.0.1:4320 make run
```

### Ingestion Returns Unauthorized

Check that your request includes:

```text
Authorization: Bearer <token>
```

or:

```text
X-SignalPlane-Token: <token>
```

Check tokens:

```bash
curl http://127.0.0.1:4318/api/tokens \
  -H "Authorization: Bearer dev-token"
```

### Data Does Not Persist

Confirm `SIGNALPLANE_DATA_PATH` points to a writable path.

For local runs, the default path is:

```text
data/signalplane.json
```

For Docker runs, confirm the `/data` volume is mounted.

### Dashboard Loads But Counts Do Not Change

Check ingestion response codes. Successful ingestion returns HTTP `202`.

Check bootstrap counts:

```bash
curl http://127.0.0.1:4318/api/bootstrap
```

### Platform Dependency Shows Down

Confirm the full stack is running:

```bash
docker compose ps
```

Then inspect dependency logs:

```bash
make stack-logs
```

For source runs, dependency checks only appear when the corresponding `SIGNALPLANE_*` dependency environment variables are set.

## Production Caveat

The Silver developer preview is not yet production-grade. It uses JSON snapshot persistence and does not yet have:

- Full user login.
- Full RBAC.
- Runtime writes to PostgreSQL and ClickHouse.
- Retention policies.
- Notification delivery.
- Uptime history and availability rollups.
- OpenTelemetry ingestion.
