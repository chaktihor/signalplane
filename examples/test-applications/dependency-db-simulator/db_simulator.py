import pathlib
import sys

sys.path.append(str(pathlib.Path(__file__).resolve().parents[1] / "lib"))
from sp_client import post

resource = {
    "service": "postgres-orders-db",
    "host": "db-1",
    "environment": "production",
    "region": "local",
    "version": "15",
}

post("/api/ingest/metrics", {
    "metrics": [
        {"name": "db.query.duration", "value": 312, "unit": "ms", "type": "histogram", "resource": resource},
        {"name": "db.connections.active", "value": 72, "unit": "connections", "type": "gauge", "resource": resource},
        {"name": "db.cache.hit_rate", "value": 94.2, "unit": "percent", "type": "gauge", "resource": resource},
    ]
})

post("/api/ingest/logs", {
    "severity": "warning",
    "message": "slow query detected on orders lookup",
    "resource": resource,
})

