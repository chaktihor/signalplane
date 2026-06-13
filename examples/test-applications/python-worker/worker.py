import pathlib
import sys

sys.path.append(str(pathlib.Path(__file__).resolve().parents[1] / "lib"))
from sp_client import post

resource = {
    "service": "python-invoice-worker",
    "host": "worker-1",
    "environment": "production",
    "region": "local",
    "version": "0.1.0",
}

post("/api/ingest/metrics", {
    "metrics": [
        {"name": "job.duration", "value": 1400, "unit": "ms", "type": "histogram", "resource": resource},
        {"name": "job.completed", "value": 24, "unit": "jobs", "type": "counter", "resource": resource},
    ]
})

post("/api/ingest/logs", {
    "severity": "info",
    "message": "invoice batch completed 24 jobs",
    "resource": resource,
})

