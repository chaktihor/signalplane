import json
import os
import urllib.request


BASE_URL = os.environ.get("SIGNALPLANE_URL", "http://127.0.0.1:4318").rstrip("/")
TOKEN = os.environ.get("SIGNALPLANE_TOKEN", "dev-token")


def post(path, payload):
    data = json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(
        BASE_URL + path,
        data=data,
        method="POST",
        headers={
            "Authorization": f"Bearer {TOKEN}",
            "Content-Type": "application/json",
        },
    )
    with urllib.request.urlopen(request, timeout=5) as response:
        body = response.read().decode("utf-8")
        print(f"{path}: {response.status} {body[:220]}")

