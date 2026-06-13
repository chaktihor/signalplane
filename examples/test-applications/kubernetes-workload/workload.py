import pathlib
import sys

sys.path.append(str(pathlib.Path(__file__).resolve().parents[1] / "lib"))
from sp_client import post

resource = {
    "service": "k8s-cart-service",
    "host": "cart-pod-7d9f6c9c8b-x1",
    "environment": "production",
    "region": "local",
    "version": "0.1.0",
    "attributes": {
        "k8s.cluster": "local-kind",
        "k8s.namespace": "shop",
        "k8s.workload": "cart-service",
        "k8s.pod": "cart-pod-7d9f6c9c8b-x1",
        "container": "cart",
    },
}

post("/api/ingest/metrics", {
    "metrics": [
        {"name": "k8s.pod.cpu.request", "value": 250, "unit": "millicores", "type": "gauge", "resource": resource},
        {"name": "k8s.pod.restart_count", "value": 1, "unit": "restarts", "type": "counter", "resource": resource},
    ]
})

post("/api/ingest/logs", {
    "severity": "info",
    "message": "cart workload processed request with Kubernetes metadata",
    "resource": resource,
})

