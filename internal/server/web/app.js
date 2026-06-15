const $ = (selector) => document.querySelector(selector);

async function getJSON(path) {
  const response = await fetch(path);
  if (!response.ok) throw new Error(`Request failed: ${response.status}`);
  return response.json();
}

function text(value) {
  return String(value ?? "").replace(/[&<>"']/g, (char) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    "\"": "&quot;",
    "'": "&#039;"
  })[char]);
}

function setHealth(health) {
  const pill = $("#health-pill");
  pill.className = `pill ${health}`;
  pill.textContent = health.toUpperCase();
}

function renderCounts(counts) {
  const metrics = [
    ["Services", counts.services],
    ["Hosts", counts.hosts],
    ["Users", counts.users],
    ["Logs", counts.logs],
    ["Traces", counts.traces],
    ["Metrics", counts.metrics],
    ["Open alerts", counts.openAlerts],
    ["Alert rules", counts.alertRules],
    ["Channels", counts.notificationChannels],
    ["Incidents", counts.incidents],
    ["Uptime checks", counts.uptimeMonitors]
  ];
  $("#overview").innerHTML = metrics.map(([label, value]) => `
    <div class="metric"><span>${text(label)}</span><strong>${text(value)}</strong></div>
  `).join("");
}

function badge(value) {
  const normalized = String(value || "unknown").toLowerCase();
  return `<span class="status ${text(normalized)}">${text(normalized)}</span>`;
}

function renderServices(services) {
  $("#services-body").innerHTML = services.map((service) => `
    <tr>
      <td><strong>${text(service.name)}</strong><span>${text(service.environment)} / ${text(service.region)}</span></td>
      <td>${badge(service.status)}</td>
      <td>${text(service.owner)}</td>
      <td>${number(service.stats?.requestRate)} req/s</td>
      <td>${number(service.stats?.p95LatencyMs)} ms</td>
      <td>${number(service.stats?.errorRate)}%</td>
    </tr>
  `).join("");
}

function renderHosts(hosts) {
  $("#hosts-list").innerHTML = hosts.map((host) => `
    <div class="item">
      <strong>${text(host.name)} ${badge(host.status)}</strong>
      <span>${text(host.region)} - agent ${text(host.agentVersion)} - cpu ${number(host.metrics?.cpu)}% - memory ${number(host.metrics?.memory)}%</span>
    </div>
  `).join("");
}

function renderAlerts(alerts) {
  $("#alerts-list").innerHTML = alerts.length ? alerts.map((alert) => `
    <div class="item ${text(alert.severity)}">
      <strong>${badge(alert.severity)} ${text(alert.title)}</strong>
      <span>${text(alert.status)} - ${text(alert.source)} - ${text(alert.message)}</span>
    </div>
  `).join("") : empty("No alerts", "Everything is quiet.");
}

function renderMetrics(metrics) {
  $("#metrics-list").innerHTML = metrics.length ? metrics.map((metric) => `
    <div class="item">
      <strong>${text(metric.name)} <span class="metric-value">${number(metric.value)} ${text(metric.unit)}</span></strong>
      <span>${text(metric.resource?.service || metric.resource?.host || "unknown")} - ${text(metric.type)} - ${time(metric.timestamp)}</span>
    </div>
  `).join("") : empty("No metrics", "Send telemetry to populate metric samples.");
}

function renderUptime(monitors) {
  $("#uptime-list").innerHTML = monitors.length ? monitors.map((monitor) => `
    <div class="item">
      <strong>${text(monitor.name)} ${badge(monitor.status)}</strong>
      <span>${text(monitor.url)} - ${monitor.lastStatusCode || "not checked"} - ${number(monitor.lastResponseMs)} ms - checked ${time(monitor.lastCheckedAt)}</span>
      ${monitor.lastError ? `<span class="error-text">${text(monitor.lastError)}</span>` : ""}
    </div>
  `).join("") : empty("No uptime monitors", "Register a monitor to track endpoint availability.");
}

function renderLogs(logs) {
  $("#logs-list").innerHTML = logs.length ? logs.map((log) => `
    <div class="item ${text(log.severity)}">
      <strong>${badge(log.severity)} ${text(log.resource?.service || log.resource?.host || "unknown")}</strong>
      <span>${text(log.message)}</span>
      <span>${text(log.traceId || "")} ${time(log.timestamp)}</span>
    </div>
  `).join("") : empty("No logs", "Send log telemetry to inspect events.");
}

function renderTraces(traces) {
  $("#traces-list").innerHTML = traces.length ? traces.map((trace) => `
    <div class="item">
      <strong>${text(trace.rootService)} - ${text(trace.operation)} ${badge(trace.status)}</strong>
      <span>${number(trace.durationMs)} ms - ${text(trace.spans.length)} spans - ${text(trace.traceId)}</span>
    </div>
  `).join("") : empty("No traces", "Send trace telemetry to inspect request flow.");
}

function renderIncidents(incidents) {
  $("#incidents-list").innerHTML = incidents.length ? incidents.map((incident) => `
    <div class="item ${text(incident.severity)}">
      <strong>${badge(incident.severity)} ${text(incident.title)}</strong>
      <span>${text(incident.status)} - owner ${text(incident.owner)} - ${text((incident.affectedServices || []).join(", "))}</span>
    </div>
  `).join("") : empty("No incidents", "Create incidents from alerts or manual response work.");
}

function renderDependencies(dependencies) {
  $("#dependencies-list").innerHTML = dependencies.length ? dependencies.map((dependency) => `
    <div class="dependency">
      <strong>${text(dependency.name)} ${badge(dependency.status)}</strong>
      <span>${text(dependency.kind)} - ${text(dependency.target)} - ${number(dependency.latencyMs)} ms</span>
      ${dependency.error ? `<span class="error-text">${text(dependency.error)}</span>` : ""}
    </div>
  `).join("") : empty("No dependency checks", "Set dependency environment variables to monitor the local platform stack.");
}

function empty(title, body) {
  return `<div class="item"><strong>${text(title)}</strong><span>${text(body)}</span></div>`;
}

function number(value) {
  if (value === undefined || value === null || value === "") return "0";
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) return text(value);
  return parsed.toLocaleString(undefined, { maximumFractionDigits: 1 });
}

function time(value) {
  if (!value) return "never";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return text(value);
  return date.toLocaleTimeString();
}

async function refresh() {
  const [bootstrap, services, hosts, alerts, logs, traces, metrics, uptime, incidents, dependencies] = await Promise.all([
    getJSON("/api/bootstrap"),
    getJSON("/api/services"),
    getJSON("/api/hosts"),
    getJSON("/api/alerts"),
    getJSON("/api/logs?limit=10"),
    getJSON("/api/traces?limit=10"),
    getJSON("/api/metrics?limit=10"),
    getJSON("/api/uptime-monitors"),
    getJSON("/api/incidents"),
    getJSON("/api/system/dependencies")
  ]);
  setHealth(bootstrap.health);
  renderCounts(bootstrap.counts);
  renderServices(services.services);
  renderHosts(hosts.hosts);
  renderAlerts(alerts.alerts);
  renderLogs(logs.logs);
  renderTraces(traces.traces);
  renderMetrics(metrics.metrics);
  renderUptime(uptime.uptimeMonitors);
  renderIncidents(incidents.incidents);
  renderDependencies(dependencies.dependencies);
  $("#last-updated").textContent = `Updated ${new Date().toLocaleTimeString()}`;
}

$("#refresh").addEventListener("click", refresh);
refresh().catch((error) => {
  $("#health-pill").textContent = error.message;
});
setInterval(() => refresh().catch(() => {}), 5000);
