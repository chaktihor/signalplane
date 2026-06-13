# SignalPlane Test Applications

These tiny applications exercise the Silver telemetry paths without external dependencies.

Each folder represents a common application type:

| Folder | Application Type | What It Sends |
|---|---|---|
| `go-backend-api` | Backend API | Request metrics, logs, distributed trace |
| `node-microservice` | Microservice | Dependency latency, warning logs, trace spans |
| `python-web-backend` | Web backend | HTTP duration, access logs, frontend-facing trace |
| `python-worker` | Worker / batch job | Job duration, success/failure logs |
| `dependency-db-simulator` | Database / dependency | Query latency, saturation metric, slow-query log |
| `c-host-probe` | Host / VM probe | Host heartbeat, CPU/memory/disk-style metrics |
| `kubernetes-workload` | Kubernetes-style app metadata | Pod/workload labels through resource attributes |
| `uptime-target` | HTTP uptime target | Uptime monitor definition |

## Run All

Start SignalPlane first:

```bash
make run
```

Then, in another terminal:

```bash
./examples/test-applications/run_all.sh
```

Defaults:

- Base URL: `http://127.0.0.1:4318`
- Ingestion token: `dev-token`

Override them if needed:

```bash
SIGNALPLANE_URL=http://127.0.0.1:4318 SIGNALPLANE_TOKEN=dev-token ./examples/test-applications/run_all.sh
```

