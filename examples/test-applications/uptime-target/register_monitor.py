import pathlib
import sys

sys.path.append(str(pathlib.Path(__file__).resolve().parents[1] / "lib"))
from sp_client import post

post("/api/uptime-monitors", {
    "name": "Local SignalPlane health",
    "url": "http://127.0.0.1:4318/healthz",
    "method": "GET",
    "expectedStatus": 200,
    "intervalSeconds": 60,
    "timeoutSeconds": 5,
})
