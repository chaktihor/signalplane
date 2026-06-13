package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Store struct {
	mu        sync.RWMutex
	path      string
	org       Organization
	envs      []Environment
	tokens    map[string]APIToken
	services  map[string]Service
	hosts     map[string]Host
	metrics   []Metric
	logs      []Log
	traces    []Trace
	alerts    map[string]Alert
	incidents map[string]Incident
	uptime    map[string]UptimeMonitor
	audit     []AuditEvent
}

type Options struct {
	Path           string
	Seed           bool
	BootstrapToken string
}

type snapshot struct {
	Organization Organization             `json:"organization"`
	Environments []Environment            `json:"environments"`
	Tokens       map[string]APIToken      `json:"tokens"`
	Services     map[string]Service       `json:"services"`
	Hosts        map[string]Host          `json:"hosts"`
	Metrics      []Metric                 `json:"metrics"`
	Logs         []Log                    `json:"logs"`
	Traces       []Trace                  `json:"traces"`
	Alerts       map[string]Alert         `json:"alerts"`
	Incidents    map[string]Incident      `json:"incidents"`
	Uptime       map[string]UptimeMonitor `json:"uptime"`
	Audit        []AuditEvent             `json:"audit"`
}

type Organization struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

type Environment struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

type APIToken struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Token     string `json:"token,omitempty"`
	Scope     string `json:"scope"`
	CreatedAt string `json:"createdAt"`
	RevokedAt string `json:"revokedAt,omitempty"`
}

type Service struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Environment  string       `json:"environment"`
	Owner        string       `json:"owner"`
	Status       string       `json:"status"`
	Version      string       `json:"version"`
	Region       string       `json:"region"`
	Tags         []string     `json:"tags"`
	Dependencies []string     `json:"dependencies"`
	Stats        ServiceStats `json:"stats"`
	CreatedAt    string       `json:"createdAt"`
	UpdatedAt    string       `json:"updatedAt"`
}

type ServiceStats struct {
	RequestRate  float64 `json:"requestRate"`
	ErrorRate    float64 `json:"errorRate"`
	P95LatencyMS float64 `json:"p95LatencyMs"`
}

type Host struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Environment  string             `json:"environment"`
	Region       string             `json:"region"`
	Status       string             `json:"status"`
	AgentVersion string             `json:"agentVersion"`
	LastSeenAt   string             `json:"lastSeenAt"`
	Tags         []string           `json:"tags"`
	Metrics      map[string]float64 `json:"metrics"`
	CreatedAt    string             `json:"createdAt"`
	UpdatedAt    string             `json:"updatedAt"`
}

type Resource struct {
	Service     string            `json:"service,omitempty"`
	Host        string            `json:"host,omitempty"`
	Environment string            `json:"environment,omitempty"`
	Region      string            `json:"region,omitempty"`
	Version     string            `json:"version,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

type Metric struct {
	ID        string            `json:"id"`
	Timestamp string            `json:"timestamp"`
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Unit      string            `json:"unit"`
	Type      string            `json:"type"`
	Labels    map[string]string `json:"labels"`
	Resource  Resource          `json:"resource"`
}

type Log struct {
	ID        string            `json:"id"`
	Timestamp string            `json:"timestamp"`
	Severity  string            `json:"severity"`
	Message   string            `json:"message"`
	TraceID   string            `json:"traceId,omitempty"`
	SpanID    string            `json:"spanId,omitempty"`
	Fields    map[string]string `json:"fields"`
	Resource  Resource          `json:"resource"`
}

type Trace struct {
	ID          string     `json:"id"`
	TraceID     string     `json:"traceId"`
	Timestamp   string     `json:"timestamp"`
	RootService string     `json:"rootService"`
	Operation   string     `json:"operation"`
	DurationMS  float64    `json:"durationMs"`
	Status      string     `json:"status"`
	Spans       []Span     `json:"spans"`
	Resources   []Resource `json:"resources"`
}

type Span struct {
	ID         string            `json:"id"`
	ParentID   string            `json:"parentId,omitempty"`
	Name       string            `json:"name"`
	Service    string            `json:"service"`
	DurationMS float64           `json:"durationMs"`
	Status     string            `json:"status"`
	Attributes map[string]string `json:"attributes"`
}

type Alert struct {
	ID             string            `json:"id"`
	Timestamp      string            `json:"timestamp"`
	Title          string            `json:"title"`
	Severity       string            `json:"severity"`
	Status         string            `json:"status"`
	Source         string            `json:"source"`
	EntityID       string            `json:"entityId,omitempty"`
	Message        string            `json:"message"`
	Labels         map[string]string `json:"labels"`
	RelatedLogID   string            `json:"relatedLogId,omitempty"`
	RelatedTraceID string            `json:"relatedTraceId,omitempty"`
	AcknowledgedAt string            `json:"acknowledgedAt,omitempty"`
	ResolvedAt     string            `json:"resolvedAt,omitempty"`
}

type Incident struct {
	ID               string            `json:"id"`
	Timestamp        string            `json:"timestamp"`
	Title            string            `json:"title"`
	Severity         string            `json:"severity"`
	Status           string            `json:"status"`
	Owner            string            `json:"owner"`
	AffectedServices []string          `json:"affectedServices"`
	AffectedHosts    []string          `json:"affectedHosts"`
	AlertIDs         []string          `json:"alertIds"`
	Notes            []string          `json:"notes"`
	Timeline         []TimelineEvent   `json:"timeline"`
	Labels           map[string]string `json:"labels"`
}

type TimelineEvent struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

type UptimeMonitor struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	URL             string `json:"url"`
	Method          string `json:"method"`
	ExpectedStatus  int    `json:"expectedStatus"`
	IntervalSeconds int    `json:"intervalSeconds"`
	TimeoutSeconds  int    `json:"timeoutSeconds"`
	Status          string `json:"status"`
	LastCheckedAt   string `json:"lastCheckedAt,omitempty"`
	CreatedAt       string `json:"createdAt"`
}

type AuditEvent struct {
	ID         string            `json:"id"`
	Timestamp  string            `json:"timestamp"`
	Action     string            `json:"action"`
	EntityType string            `json:"entityType"`
	EntityID   string            `json:"entityId"`
	Details    map[string]string `json:"details,omitempty"`
}

type Summary struct {
	Organization Organization  `json:"organization"`
	Environments []Environment `json:"environments"`
	Counts       Counts        `json:"counts"`
	Health       string        `json:"health"`
	RecentAlerts []Alert       `json:"recentAlerts"`
	TopServices  []Service     `json:"topServices"`
	TopHosts     []Host        `json:"topHosts"`
}

type Counts struct {
	Services       int `json:"services"`
	Hosts          int `json:"hosts"`
	Tokens         int `json:"tokens"`
	Metrics        int `json:"metrics"`
	Logs           int `json:"logs"`
	Traces         int `json:"traces"`
	Alerts         int `json:"alerts"`
	OpenAlerts     int `json:"openAlerts"`
	Incidents      int `json:"incidents"`
	UptimeMonitors int `json:"uptimeMonitors"`
}

type MetricInput struct {
	Timestamp string            `json:"timestamp"`
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Unit      string            `json:"unit"`
	Type      string            `json:"type"`
	Labels    map[string]string `json:"labels"`
	Resource  Resource          `json:"resource"`
}

type LogInput struct {
	Timestamp string            `json:"timestamp"`
	Severity  string            `json:"severity"`
	Message   string            `json:"message"`
	TraceID   string            `json:"traceId"`
	SpanID    string            `json:"spanId"`
	Fields    map[string]string `json:"fields"`
	Resource  Resource          `json:"resource"`
}

type TraceInput struct {
	TraceID string      `json:"traceId"`
	Spans   []SpanInput `json:"spans"`
}

type SpanInput struct {
	SpanID     string            `json:"spanId"`
	ParentID   string            `json:"parentId"`
	Name       string            `json:"name"`
	DurationMS float64           `json:"durationMs"`
	Status     string            `json:"status"`
	Resource   Resource          `json:"resource"`
	Attributes map[string]string `json:"attributes"`
	Timestamp  string            `json:"timestamp"`
}

type HostInput struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Environment  string             `json:"environment"`
	Region       string             `json:"region"`
	Status       string             `json:"status"`
	AgentVersion string             `json:"agentVersion"`
	Tags         []string           `json:"tags"`
	Metrics      map[string]float64 `json:"metrics"`
}

type TokenInput struct {
	Name  string `json:"name"`
	Token string `json:"token"`
	Scope string `json:"scope"`
}

func New() *Store {
	stamp := now()
	return &Store{
		org:       Organization{ID: "org-default", Name: "SignalPlane Local", CreatedAt: stamp},
		envs:      []Environment{{ID: "env-production", Name: "production", CreatedAt: stamp}},
		tokens:    make(map[string]APIToken),
		services:  make(map[string]Service),
		hosts:     make(map[string]Host),
		alerts:    make(map[string]Alert),
		incidents: make(map[string]Incident),
		uptime:    make(map[string]UptimeMonitor),
	}
}

func Open(options Options) (*Store, error) {
	if options.BootstrapToken == "" {
		options.BootstrapToken = "dev-token"
	}
	if options.Path != "" {
		loaded, err := load(options.Path)
		if err == nil {
			loaded.path = options.Path
			loaded.ensureBootstrapToken(options.BootstrapToken)
			return loaded, loaded.saveLocked()
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	var s *Store
	if options.Seed {
		s = NewSeeded()
	} else {
		s = New()
	}
	s.path = options.Path
	s.ensureBootstrapToken(options.BootstrapToken)
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return s, nil
}

func NewSeeded() *Store {
	s := New()
	s.UpsertHost(HostInput{
		ID: "host-api-1", Name: "api-1", Environment: "production", Region: "local", Status: "online",
		AgentVersion: "sp-node-dev", Tags: []string{"silver", "demo"},
		Metrics: map[string]float64{"cpu": 37, "memory": 64, "disk": 51},
	})
	s.upsertServiceLocked(Service{
		ID: "checkout-api", Name: "checkout-api", Environment: "production", Owner: "platform", Status: "healthy",
		Version: "0.1.0", Region: "local", Tags: []string{"silver", "demo"}, Dependencies: []string{"postgres", "payments-api"},
		Stats: ServiceStats{RequestRate: 42, ErrorRate: 0.8, P95LatencyMS: 182}, CreatedAt: now(), UpdatedAt: now(),
	})
	s.IngestMetrics([]MetricInput{
		{Name: "http.server.duration", Value: 182, Unit: "ms", Type: "histogram", Resource: Resource{Service: "checkout-api", Host: "host-api-1", Environment: "production"}},
		{Name: "http.server.requests", Value: 420, Unit: "requests", Type: "counter", Resource: Resource{Service: "checkout-api", Host: "host-api-1", Environment: "production"}},
	})
	s.IngestLogs([]LogInput{
		{Severity: "info", Message: "Checkout service started", Resource: Resource{Service: "checkout-api", Host: "host-api-1", Environment: "production"}},
		{Severity: "warning", Message: "Payment dependency latency above baseline", TraceID: "trace-demo-1", Resource: Resource{Service: "checkout-api", Host: "host-api-1", Environment: "production"}},
	})
	s.IngestTraces([]TraceInput{{
		TraceID: "trace-demo-1",
		Spans: []SpanInput{
			{SpanID: "span-root", Name: "POST /checkout", DurationMS: 212, Status: "ok", Resource: Resource{Service: "checkout-api", Environment: "production"}},
			{SpanID: "span-payment", ParentID: "span-root", Name: "POST /payments", DurationMS: 151, Status: "ok", Resource: Resource{Service: "payments-api", Environment: "production"}},
		},
	}})
	s.CreateUptimeMonitor(UptimeMonitor{Name: "SignalPlane local API", URL: "http://localhost:4318/healthz", Method: "GET", ExpectedStatus: 200, IntervalSeconds: 60, TimeoutSeconds: 10})
	return s
}

func load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	s := New()
	s.org = snap.Organization
	s.envs = snap.Environments
	s.tokens = snap.Tokens
	s.services = snap.Services
	s.hosts = snap.Hosts
	s.metrics = snap.Metrics
	s.logs = snap.Logs
	s.traces = snap.Traces
	s.alerts = snap.Alerts
	s.incidents = snap.Incidents
	s.uptime = snap.Uptime
	s.audit = snap.Audit
	s.normalizeLoaded()
	return s, nil
}

func (s *Store) normalizeLoaded() {
	if s.org.ID == "" {
		s.org = Organization{ID: "org-default", Name: "SignalPlane Local", CreatedAt: now()}
	}
	if len(s.envs) == 0 {
		s.envs = []Environment{{ID: "env-production", Name: "production", CreatedAt: now()}}
	}
	if s.tokens == nil {
		s.tokens = make(map[string]APIToken)
	}
	if s.services == nil {
		s.services = make(map[string]Service)
	}
	if s.hosts == nil {
		s.hosts = make(map[string]Host)
	}
	if s.alerts == nil {
		s.alerts = make(map[string]Alert)
	}
	if s.incidents == nil {
		s.incidents = make(map[string]Incident)
	}
	if s.uptime == nil {
		s.uptime = make(map[string]UptimeMonitor)
	}
}

func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	snap := snapshot{
		Organization: s.org,
		Environments: append([]Environment(nil), s.envs...),
		Tokens:       cloneMapValues(s.tokens),
		Services:     cloneMapValues(s.services),
		Hosts:        cloneMapValues(s.hosts),
		Metrics:      append([]Metric(nil), s.metrics...),
		Logs:         append([]Log(nil), s.logs...),
		Traces:       append([]Trace(nil), s.traces...),
		Alerts:       cloneMapValues(s.alerts),
		Incidents:    cloneMapValues(s.incidents),
		Uptime:       cloneMapValues(s.uptime),
		Audit:        append([]AuditEvent(nil), s.audit...),
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) ensureBootstrapToken(token string) {
	if token == "" {
		return
	}
	if s.tokens == nil {
		s.tokens = make(map[string]APIToken)
	}
	for _, existing := range s.tokens {
		if existing.Token == token && existing.RevokedAt == "" {
			return
		}
	}
	s.tokens["token-dev"] = APIToken{ID: "token-dev", Name: "local-dev", Token: token, Scope: "admin", CreatedAt: now()}
}

func (s *Store) Summary() Summary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	services := mapValues(s.services)
	hosts := mapValues(s.hosts)
	alerts := mapValues(s.alerts)
	sortAlerts(alerts)
	open := 0
	critical := 0
	for _, alert := range alerts {
		if alert.Status == "open" {
			open++
			if alert.Severity == "critical" {
				critical++
			}
		}
	}
	health := "healthy"
	if critical > 0 {
		health = "critical"
	} else if open > 0 {
		health = "warning"
	}
	return Summary{
		Organization: s.org,
		Environments: append([]Environment(nil), s.envs...),
		Counts: Counts{
			Services: len(s.services), Hosts: len(s.hosts), Metrics: len(s.metrics), Logs: len(s.logs),
			Tokens: len(s.tokens), Traces: len(s.traces), Alerts: len(s.alerts), OpenAlerts: open, Incidents: len(s.incidents), UptimeMonitors: len(s.uptime),
		},
		Health: health, RecentAlerts: take(alerts, 5), TopServices: take(services, 5), TopHosts: take(hosts, 5),
	}
}

func (s *Store) ValidToken(token, scope string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, candidate := range s.tokens {
		if candidate.RevokedAt != "" || candidate.Token != token {
			continue
		}
		return candidate.Scope == scope || candidate.Scope == "admin"
	}
	return false
}

func (s *Store) Tokens(max int) []APIToken {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tokens := mapValues(s.tokens)
	sort.SliceStable(tokens, func(i, j int) bool { return tokens[i].CreatedAt > tokens[j].CreatedAt })
	for i := range tokens {
		tokens[i].Token = maskToken(tokens[i].Token)
	}
	return take(tokens, saneLimit(max))
}

func (s *Store) CreateToken(input TokenInput) APIToken {
	s.mu.Lock()
	defer s.mu.Unlock()
	token := APIToken{
		ID:        newID("tok"),
		Name:      firstNonEmpty(input.Name, "unnamed-token"),
		Token:     firstNonEmpty(input.Token, newID("sp")),
		Scope:     firstNonEmpty(input.Scope, "ingest"),
		CreatedAt: now(),
	}
	if token.Scope != "ingest" && token.Scope != "admin" && token.Scope != "read" {
		token.Scope = "ingest"
	}
	s.tokens[token.ID] = token
	s.auditLocked("token.created", "token", token.ID, map[string]string{"scope": token.Scope})
	_ = s.saveLocked()
	return token
}

func (s *Store) UpsertHost(input HostInput) Host {
	s.mu.Lock()
	defer s.mu.Unlock()
	host := s.upsertHostLocked(input)
	_ = s.saveLocked()
	return host
}

func (s *Store) IngestMetrics(inputs []MetricInput) []Metric {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Metric, 0, len(inputs))
	for _, input := range inputs {
		resource := normalizeResource(input.Resource)
		s.ensureEntitiesLocked(resource)
		metric := Metric{
			ID: newID("met"), Timestamp: coalesce(input.Timestamp, now()), Name: coalesce(input.Name, "custom.metric"),
			Value: input.Value, Unit: coalesce(input.Unit, "count"), Type: coalesce(input.Type, "gauge"),
			Labels: cloneMap(input.Labels), Resource: resource,
		}
		s.metrics = prependBounded(s.metrics, metric, 5000)
		out = append(out, metric)
	}
	_ = s.saveLocked()
	return out
}

func (s *Store) IngestLogs(inputs []LogInput) []Log {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Log, 0, len(inputs))
	for _, input := range inputs {
		resource := normalizeResource(input.Resource)
		s.ensureEntitiesLocked(resource)
		log := Log{
			ID: newID("log"), Timestamp: coalesce(input.Timestamp, now()), Severity: normalizeSeverity(input.Severity),
			Message: input.Message, TraceID: input.TraceID, SpanID: input.SpanID, Fields: cloneMap(input.Fields), Resource: resource,
		}
		s.logs = prependBounded(s.logs, log, 5000)
		out = append(out, log)
		if log.Severity == "error" || log.Severity == "fatal" {
			s.createAlertLocked(Alert{
				Title:    "Error log in " + firstNonEmpty(resource.Service, resource.Host, "unknown source"),
				Severity: severityForLog(log.Severity), Source: "logs", EntityID: firstNonEmpty(resource.Service, resource.Host),
				Message: log.Message, RelatedLogID: log.ID,
			})
		}
	}
	_ = s.saveLocked()
	return out
}

func (s *Store) IngestTraces(inputs []TraceInput) []Trace {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Trace, 0, len(inputs))
	for _, input := range inputs {
		traceID := coalesce(input.TraceID, newID("trace"))
		trace := Trace{ID: newID("trc"), TraceID: traceID, Timestamp: now(), Status: "ok"}
		seenResources := make(map[string]Resource)
		for i, spanInput := range input.Spans {
			resource := normalizeResource(spanInput.Resource)
			s.ensureEntitiesLocked(resource)
			service := firstNonEmpty(resource.Service, "unknown")
			status := normalizeTraceStatus(spanInput.Status)
			if status == "error" {
				trace.Status = "error"
			}
			if spanInput.DurationMS > trace.DurationMS {
				trace.DurationMS = spanInput.DurationMS
			}
			if i == 0 {
				trace.RootService = service
				trace.Operation = coalesce(spanInput.Name, "request")
				trace.Timestamp = coalesce(spanInput.Timestamp, now())
			}
			trace.Spans = append(trace.Spans, Span{
				ID: coalesce(spanInput.SpanID, newID("span")), ParentID: spanInput.ParentID, Name: coalesce(spanInput.Name, "span"),
				Service: service, DurationMS: spanInput.DurationMS, Status: status, Attributes: cloneMap(spanInput.Attributes),
			})
			seenResources[resourceKey(resource)] = resource
		}
		for _, resource := range seenResources {
			trace.Resources = append(trace.Resources, resource)
		}
		if trace.RootService == "" {
			trace.RootService = "unknown"
		}
		if trace.Operation == "" {
			trace.Operation = "request"
		}
		s.traces = prependBounded(s.traces, trace, 5000)
		out = append(out, trace)
		if trace.Status == "error" {
			s.createAlertLocked(Alert{Title: "Trace error in " + trace.RootService, Severity: "warning", Source: "traces", EntityID: trace.RootService, Message: trace.Operation + " completed with errors", RelatedTraceID: trace.TraceID})
		}
	}
	_ = s.saveLocked()
	return out
}

func (s *Store) CreateUptimeMonitor(input UptimeMonitor) UptimeMonitor {
	s.mu.Lock()
	defer s.mu.Unlock()
	monitor := input
	monitor.ID = coalesce(monitor.ID, newID("upt"))
	monitor.Name = firstNonEmpty(monitor.Name, monitor.URL, "HTTP monitor")
	monitor.Method = coalesce(monitor.Method, "GET")
	if monitor.ExpectedStatus == 0 {
		monitor.ExpectedStatus = 200
	}
	if monitor.IntervalSeconds == 0 {
		monitor.IntervalSeconds = 60
	}
	if monitor.TimeoutSeconds == 0 {
		monitor.TimeoutSeconds = 10
	}
	monitor.Status = coalesce(monitor.Status, "unknown")
	monitor.CreatedAt = coalesce(monitor.CreatedAt, now())
	s.uptime[monitor.ID] = monitor
	s.auditLocked("uptime.created", "uptimeMonitor", monitor.ID, nil)
	_ = s.saveLocked()
	return monitor
}

func (s *Store) CreateIncident(input Incident) Incident {
	s.mu.Lock()
	defer s.mu.Unlock()
	incident := input
	incident.ID = coalesce(incident.ID, newID("inc"))
	incident.Timestamp = coalesce(incident.Timestamp, now())
	incident.Title = coalesce(incident.Title, "Untitled incident")
	incident.Severity = normalizeAlertSeverity(incident.Severity)
	incident.Status = coalesce(incident.Status, "open")
	incident.Owner = coalesce(incident.Owner, "unassigned")
	incident.Timeline = append(incident.Timeline, TimelineEvent{Timestamp: now(), Message: "Incident created"})
	if incident.Labels == nil {
		incident.Labels = map[string]string{}
	}
	s.incidents[incident.ID] = incident
	s.auditLocked("incident.created", "incident", incident.ID, nil)
	_ = s.saveLocked()
	return incident
}

func (s *Store) UpdateAlert(id, status string) (Alert, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	alert, ok := s.alerts[id]
	if !ok {
		return Alert{}, false
	}
	alert.Status = coalesce(status, alert.Status)
	if alert.Status == "acknowledged" && alert.AcknowledgedAt == "" {
		alert.AcknowledgedAt = now()
	}
	if alert.Status == "resolved" && alert.ResolvedAt == "" {
		alert.ResolvedAt = now()
	}
	s.alerts[id] = alert
	s.auditLocked("alert.updated", "alert", id, map[string]string{"status": alert.Status})
	_ = s.saveLocked()
	return alert, true
}

func (s *Store) Services(max int) []Service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return take(mapValues(s.services), saneLimit(max))
}
func (s *Store) Hosts(max int) []Host {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return take(mapValues(s.hosts), saneLimit(max))
}
func (s *Store) Metrics(max int) []Metric {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return take(append([]Metric(nil), s.metrics...), saneLimit(max))
}
func (s *Store) Logs(max int, service, severity, query string) []Log {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Log
	for _, log := range s.logs {
		if service != "" && log.Resource.Service != service {
			continue
		}
		if severity != "" && log.Severity != severity {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(log.Message), strings.ToLower(query)) {
			continue
		}
		out = append(out, log)
	}
	return take(out, saneLimit(max))
}
func (s *Store) Traces(max int, service, status string) []Trace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Trace
	for _, trace := range s.traces {
		if status != "" && trace.Status != status {
			continue
		}
		if service != "" && !traceHasService(trace, service) {
			continue
		}
		out = append(out, trace)
	}
	return take(out, saneLimit(max))
}
func (s *Store) Alerts(max int) []Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	alerts := mapValues(s.alerts)
	sortAlerts(alerts)
	return take(alerts, saneLimit(max))
}
func (s *Store) Incidents(max int) []Incident {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return take(mapValues(s.incidents), saneLimit(max))
}
func (s *Store) UptimeMonitors(max int) []UptimeMonitor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return take(mapValues(s.uptime), saneLimit(max))
}
func (s *Store) AuditEvents(max int) []AuditEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return take(append([]AuditEvent(nil), s.audit...), saneLimit(max))
}

func (s *Store) upsertHostLocked(input HostInput) Host {
	id := firstNonEmpty(input.ID, input.Name, input.ResourceName(), newID("host"))
	prev := s.hosts[id]
	host := Host{
		ID: id, Name: firstNonEmpty(input.Name, prev.Name, id), Environment: firstNonEmpty(input.Environment, prev.Environment, "production"),
		Region: firstNonEmpty(input.Region, prev.Region, "local"), Status: firstNonEmpty(input.Status, prev.Status, "online"),
		AgentVersion: firstNonEmpty(input.AgentVersion, prev.AgentVersion, "sp-node-dev"), LastSeenAt: now(),
		Tags: unique(append(prev.Tags, input.Tags...)), Metrics: mergeFloatMaps(prev.Metrics, input.Metrics),
		CreatedAt: firstNonEmpty(prev.CreatedAt, now()), UpdatedAt: now(),
	}
	s.hosts[id] = host
	return host
}

func (input HostInput) ResourceName() string { return "" }

func (s *Store) ensureEntitiesLocked(resource Resource) {
	if resource.Host != "" {
		s.upsertHostLocked(HostInput{Name: resource.Host, Environment: resource.Environment, Region: resource.Region})
	}
	if resource.Service != "" {
		s.upsertServiceLocked(Service{ID: resource.Service, Name: resource.Service, Environment: firstNonEmpty(resource.Environment, "production"), Region: firstNonEmpty(resource.Region, "local"), Status: "healthy", Version: firstNonEmpty(resource.Version, "unknown")})
	}
}

func (s *Store) upsertServiceLocked(input Service) Service {
	id := firstNonEmpty(input.ID, input.Name, newID("svc"))
	prev := s.services[id]
	service := Service{
		ID: id, Name: firstNonEmpty(input.Name, prev.Name, id), Environment: firstNonEmpty(input.Environment, prev.Environment, "production"),
		Owner: firstNonEmpty(input.Owner, prev.Owner, "unassigned"), Status: firstNonEmpty(input.Status, prev.Status, "healthy"),
		Version: firstNonEmpty(input.Version, prev.Version, "unknown"), Region: firstNonEmpty(input.Region, prev.Region, "local"),
		Tags: unique(append(prev.Tags, input.Tags...)), Dependencies: unique(append(prev.Dependencies, input.Dependencies...)),
		Stats: mergeStats(prev.Stats, input.Stats), CreatedAt: firstNonEmpty(prev.CreatedAt, now()), UpdatedAt: now(),
	}
	s.services[id] = service
	return service
}

func (s *Store) createAlertLocked(input Alert) Alert {
	alert := input
	alert.ID = coalesce(alert.ID, newID("alt"))
	alert.Timestamp = coalesce(alert.Timestamp, now())
	alert.Title = coalesce(alert.Title, "Untitled alert")
	alert.Severity = normalizeAlertSeverity(alert.Severity)
	alert.Status = coalesce(alert.Status, "open")
	if alert.Labels == nil {
		alert.Labels = map[string]string{}
	}
	s.alerts[alert.ID] = alert
	s.auditLocked("alert.created", "alert", alert.ID, map[string]string{"source": alert.Source})
	return alert
}

func (s *Store) auditLocked(action, entityType, entityID string, details map[string]string) {
	s.audit = prependBounded(s.audit, AuditEvent{ID: newID("aud"), Timestamp: now(), Action: action, EntityType: entityType, EntityID: entityID, Details: details}, 1000)
}

func normalizeResource(resource Resource) Resource {
	if resource.Environment == "" {
		resource.Environment = "production"
	}
	if resource.Region == "" {
		resource.Region = "local"
	}
	if resource.Attributes == nil {
		resource.Attributes = map[string]string{}
	}
	return resource
}

func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func newID(prefix string) string {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return prefix + "-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return prefix + "-" + hex.EncodeToString(bytes[:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
func coalesce(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
func normalizeSeverity(value string) string {
	switch strings.ToLower(value) {
	case "debug", "info", "warning", "error", "fatal":
		return strings.ToLower(value)
	case "warn":
		return "warning"
	default:
		return "info"
	}
}
func normalizeTraceStatus(value string) string {
	if strings.ToLower(value) == "error" {
		return "error"
	}
	return "ok"
}
func normalizeAlertSeverity(value string) string {
	switch strings.ToLower(value) {
	case "info", "warning", "critical":
		return strings.ToLower(value)
	default:
		return "warning"
	}
}
func severityForLog(value string) string {
	if value == "fatal" {
		return "critical"
	}
	return "warning"
}
func saneLimit(value int) int {
	if value <= 0 {
		return 100
	}
	if value > 500 {
		return 500
	}
	return value
}
func take[T any](items []T, size int) []T {
	if len(items) <= size {
		return items
	}
	return items[:size]
}
func prependBounded[T any](items []T, item T, max int) []T {
	items = append([]T{item}, items...)
	if len(items) > max {
		return items[:max]
	}
	return items
}
func mapValues[T any](m map[string]T) []T {
	out := make([]T, 0, len(m))
	for _, value := range m {
		out = append(out, value)
	}
	return out
}
func cloneMap(input map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range input {
		out[key] = value
	}
	return out
}
func cloneMapValues[T any](input map[string]T) map[string]T {
	out := map[string]T{}
	for key, value := range input {
		out[key] = value
	}
	return out
}
func mergeFloatMaps(base, patch map[string]float64) map[string]float64 {
	out := map[string]float64{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range patch {
		out[key] = value
	}
	return out
}
func mergeStats(base, patch ServiceStats) ServiceStats {
	if patch.RequestRate != 0 {
		base.RequestRate = patch.RequestRate
	}
	if patch.ErrorRate != 0 {
		base.ErrorRate = patch.ErrorRate
	}
	if patch.P95LatencyMS != 0 {
		base.P95LatencyMS = patch.P95LatencyMS
	}
	return base
}
func unique(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
func resourceKey(resource Resource) string {
	return resource.Service + "|" + resource.Host + "|" + resource.Environment + "|" + resource.Region
}
func traceHasService(trace Trace, service string) bool {
	for _, span := range trace.Spans {
		if span.Service == service {
			return true
		}
	}
	for _, resource := range trace.Resources {
		if resource.Service == service {
			return true
		}
	}
	return false
}
func sortAlerts(alerts []Alert) {
	sort.SliceStable(alerts, func(i, j int) bool { return alerts[i].Timestamp > alerts[j].Timestamp })
}
func maskToken(token string) string {
	if len(token) <= 6 {
		return "******"
	}
	return token[:3] + "..." + token[len(token)-3:]
}
