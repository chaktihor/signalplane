import pathlib
import sys
import time

sys.path.append(str(pathlib.Path(__file__).resolve().parents[1] / "lib"))
from sp_client import post

resource = {
    "service": "python-web-backend",
    "host": "python-web-1",
    "environment": "production",
    "region": "local",
    "version": "0.1.0",
}

post("/api/ingest/metrics", {
    "metrics": [
        {"name": "web.route.duration", "value": 73, "unit": "ms", "type": "histogram", "resource": resource},
        {"name": "web.active.sessions", "value": 58, "unit": "sessions", "type": "gauge", "resource": resource},
    ]
})

post("/api/ingest/logs", {
    "logs": [
        {"severity": "info", "message": "GET /dashboard completed 200", "traceId": "trace-python-web-1", "resource": resource},
    ]
})

post("/api/ingest/traces", {
    "traceId": "trace-python-web-1",
    "spans": [
        {"spanId": "span-python-web-root", "name": "GET /dashboard", "durationMs": 73, "status": "ok", "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "resource": resource},
    ],
})

