const baseURL = process.env.SIGNALPLANE_URL || "http://127.0.0.1:4318";
const token = process.env.SIGNALPLANE_TOKEN || "dev-token";

async function post(path, payload) {
  const response = await fetch(`${baseURL}${path}`, {
    method: "POST",
    headers: {
      "Authorization": `Bearer ${token}`,
      "Content-Type": "application/json"
    },
    body: JSON.stringify(payload)
  });
  console.log(path, response.status, await response.text());
}

const resource = {
  service: "node-payments-service",
  host: "node-payments-1",
  environment: "production",
  region: "local",
  version: "0.1.0"
};

await post("/api/ingest/metrics", {
  metrics: [
    { name: "dependency.gateway.duration", value: 186, unit: "ms", type: "histogram", resource },
    { name: "payment.authorizations", value: 342, unit: "requests", type: "counter", resource }
  ]
});

await post("/api/ingest/logs", {
  severity: "warning",
  message: "payment gateway latency above baseline",
  traceId: "trace-node-payments-1",
  resource
});

await post("/api/ingest/traces", {
  traceId: "trace-node-payments-1",
  spans: [
    { spanId: "span-node-root", name: "POST /authorize", durationMs: 186, status: "ok", resource },
    { spanId: "span-node-gateway", parentId: "span-node-root", name: "POST gateway.example/charge", durationMs: 144, status: "ok", resource: { ...resource, service: "external-payment-gateway" } }
  ]
});

