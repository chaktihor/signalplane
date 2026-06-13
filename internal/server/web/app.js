const $ = (selector) => document.querySelector(selector);

async function getJSON(path) {
  const response = await fetch(path);
  if (!response.ok) throw new Error(`Request failed: ${response.status}`);
  return response.json();
}

function setHealth(health) {
  const pill = $("#health-pill");
  pill.className = `pill ${health}`;
  pill.textContent = health.toUpperCase();
}

function renderMetrics(counts) {
  const metrics = [
    ["Services", counts.services],
    ["Hosts", counts.hosts],
    ["Logs", counts.logs],
    ["Traces", counts.traces],
    ["Metrics", counts.metrics],
    ["Open alerts", counts.openAlerts],
    ["Incidents", counts.incidents],
    ["Uptime checks", counts.uptimeMonitors]
  ];
  $("#overview").innerHTML = metrics.map(([label, value]) => `
    <div class="metric"><span>${label}</span><strong>${value}</strong></div>
  `).join("");
}

function renderServices(services) {
  $("#services-body").innerHTML = services.map((service) => `
    <tr>
      <td>${service.name}</td>
      <td>${service.status}</td>
      <td>${service.owner}</td>
      <td>${service.stats.p95LatencyMs} ms</td>
      <td>${service.stats.errorRate}%</td>
    </tr>
  `).join("");
}

function renderHosts(hosts) {
  $("#hosts-list").innerHTML = hosts.map((host) => `
    <div class="item">
      <strong>${host.name}</strong>
      <span>${host.status} · ${host.region} · agent ${host.agentVersion}</span>
    </div>
  `).join("");
}

function renderAlerts(alerts) {
  $("#alerts-list").innerHTML = alerts.length ? alerts.map((alert) => `
    <div class="item">
      <strong>${alert.severity.toUpperCase()} · ${alert.title}</strong>
      <span>${alert.status} · ${alert.source} · ${alert.message}</span>
    </div>
  `).join("") : `<div class="item"><strong>No alerts</strong><span>Everything is quiet.</span></div>`;
}

function renderLogs(logs) {
  $("#logs-list").innerHTML = logs.map((log) => `
    <div class="item">
      <strong>${log.severity.toUpperCase()} ${log.resource.service || log.resource.host || "unknown"}</strong>
      <span>${log.message}</span>
    </div>
  `).join("");
}

function renderTraces(traces) {
  $("#traces-list").innerHTML = traces.map((trace) => `
    <div class="item">
      <strong>${trace.rootService} · ${trace.operation}</strong>
      <span>${trace.status} · ${trace.durationMs} ms · ${trace.spans.length} spans</span>
    </div>
  `).join("");
}

async function refresh() {
  const [bootstrap, services, hosts, alerts, logs, traces] = await Promise.all([
    getJSON("/api/bootstrap"),
    getJSON("/api/services"),
    getJSON("/api/hosts"),
    getJSON("/api/alerts"),
    getJSON("/api/logs?limit=8"),
    getJSON("/api/traces?limit=8")
  ]);
  setHealth(bootstrap.health);
  renderMetrics(bootstrap.counts);
  renderServices(services.services);
  renderHosts(hosts.hosts);
  renderAlerts(alerts.alerts);
  renderLogs(logs.logs);
  renderTraces(traces.traces);
}

$("#refresh").addEventListener("click", refresh);
refresh().catch((error) => {
  $("#health-pill").textContent = error.message;
});

