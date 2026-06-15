package store

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Store struct {
	mu            sync.RWMutex
	path          string
	persistence   stateStore
	telemetry     TelemetrySink
	notifications NotificationSink
	org           Organization
	envs          []Environment
	tokens        map[string]APIToken
	users         map[string]User
	sessions      map[string]Session
	services      map[string]Service
	hosts         map[string]Host
	metrics       []Metric
	logs          []Log
	traces        []Trace
	alerts        map[string]Alert
	incidents     map[string]Incident
	uptime        map[string]UptimeMonitor
	alertRules    map[string]AlertRule
	channels      map[string]NotificationChannel
	audit         []AuditEvent
}

type Options struct {
	Path                  string
	Backend               string
	Seed                  bool
	BootstrapToken        string
	BootstrapUserEmail    string
	BootstrapUserPassword string
	TelemetrySink         TelemetrySink
	NotificationSink      NotificationSink
	PostgresURL           string
	PostgresTimeout       time.Duration
}

type TelemetrySink interface {
	WriteMetrics([]Metric) error
	WriteLogs([]Log) error
	WriteTraces([]Trace) error
	WriteUptimeResult(UptimeMonitor) error
}

type NotificationSink interface {
	NotifyAlert(Alert, []NotificationChannel)
}

type stateStore interface {
	Load() (snapshot, bool, error)
	Save(snapshot) error
}

type closeableStateStore interface {
	Close()
}

type snapshot struct {
	Organization Organization                   `json:"organization"`
	Environments []Environment                  `json:"environments"`
	Tokens       map[string]APIToken            `json:"tokens"`
	Users        map[string]User                `json:"users"`
	Sessions     map[string]Session             `json:"sessions"`
	Services     map[string]Service             `json:"services"`
	Hosts        map[string]Host                `json:"hosts"`
	Metrics      []Metric                       `json:"metrics"`
	Logs         []Log                          `json:"logs"`
	Traces       []Trace                        `json:"traces"`
	Alerts       map[string]Alert               `json:"alerts"`
	Incidents    map[string]Incident            `json:"incidents"`
	Uptime       map[string]UptimeMonitor       `json:"uptime"`
	AlertRules   map[string]AlertRule           `json:"alertRules"`
	Channels     map[string]NotificationChannel `json:"notificationChannels"`
	Audit        []AuditEvent                   `json:"audit"`
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

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	DisplayName  string `json:"displayName"`
	Role         string `json:"role"`
	PasswordHash string `json:"passwordHash,omitempty"`
	CreatedAt    string `json:"createdAt"`
	DisabledAt   string `json:"disabledAt,omitempty"`
}

type UserInput struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	Password    string `json:"password"`
}

type Session struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Token     string `json:"token,omitempty"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt"`
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

type AlertRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	SignalType  string            `json:"signalType"`
	Enabled     bool              `json:"enabled"`
	MetricName  string            `json:"metricName,omitempty"`
	LogSeverity string            `json:"logSeverity,omitempty"`
	Query       string            `json:"query,omitempty"`
	Operator    string            `json:"operator,omitempty"`
	Threshold   float64           `json:"threshold,omitempty"`
	Severity    string            `json:"severity"`
	Labels      map[string]string `json:"labels"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt"`
}

type AlertRuleInput struct {
	Name        string            `json:"name"`
	SignalType  string            `json:"signalType"`
	Enabled     *bool             `json:"enabled,omitempty"`
	MetricName  string            `json:"metricName"`
	LogSeverity string            `json:"logSeverity"`
	Query       string            `json:"query"`
	Operator    string            `json:"operator"`
	Threshold   float64           `json:"threshold"`
	Severity    string            `json:"severity"`
	Labels      map[string]string `json:"labels"`
}

type NotificationChannel struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Target    string            `json:"target"`
	Enabled   bool              `json:"enabled"`
	Config    map[string]string `json:"config"`
	CreatedAt string            `json:"createdAt"`
	UpdatedAt string            `json:"updatedAt"`
}

type NotificationChannelInput struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Target  string            `json:"target"`
	Enabled *bool             `json:"enabled,omitempty"`
	Config  map[string]string `json:"config"`
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
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	URL                 string  `json:"url"`
	Method              string  `json:"method"`
	ExpectedStatus      int     `json:"expectedStatus"`
	IntervalSeconds     int     `json:"intervalSeconds"`
	TimeoutSeconds      int     `json:"timeoutSeconds"`
	Status              string  `json:"status"`
	LastCheckedAt       string  `json:"lastCheckedAt,omitempty"`
	LastStatusCode      int     `json:"lastStatusCode,omitempty"`
	LastResponseMS      float64 `json:"lastResponseMs,omitempty"`
	LastError           string  `json:"lastError,omitempty"`
	ConsecutiveFailures int     `json:"consecutiveFailures,omitempty"`
	CreatedAt           string  `json:"createdAt"`
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
	Services             int `json:"services"`
	Hosts                int `json:"hosts"`
	Tokens               int `json:"tokens"`
	Users                int `json:"users"`
	Metrics              int `json:"metrics"`
	Logs                 int `json:"logs"`
	Traces               int `json:"traces"`
	Alerts               int `json:"alerts"`
	OpenAlerts           int `json:"openAlerts"`
	Incidents            int `json:"incidents"`
	UptimeMonitors       int `json:"uptimeMonitors"`
	AlertRules           int `json:"alertRules"`
	NotificationChannels int `json:"notificationChannels"`
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

type UptimeResult struct {
	ID         string
	Status     string
	StatusCode int
	ResponseMS float64
	Error      string
	CheckedAt  string
}

func New() *Store {
	stamp := now()
	return &Store{
		org:        Organization{ID: "org-default", Name: "SignalPlane Local", CreatedAt: stamp},
		envs:       []Environment{{ID: "env-production", Name: "production", CreatedAt: stamp}},
		tokens:     make(map[string]APIToken),
		users:      make(map[string]User),
		sessions:   make(map[string]Session),
		services:   make(map[string]Service),
		hosts:      make(map[string]Host),
		alerts:     make(map[string]Alert),
		incidents:  make(map[string]Incident),
		uptime:     make(map[string]UptimeMonitor),
		alertRules: make(map[string]AlertRule),
		channels:   make(map[string]NotificationChannel),
	}
}

func Open(options Options) (*Store, error) {
	if options.BootstrapToken == "" {
		options.BootstrapToken = "dev-token"
	}
	persistence, err := stateStoreFromOptions(options)
	if err != nil {
		return nil, err
	}
	if persistence != nil {
		snap, ok, err := persistence.Load()
		if err != nil {
			return nil, err
		}
		if ok {
			loaded := storeFromSnapshot(snap)
			loaded.persistence = persistence
			loaded.telemetry = options.TelemetrySink
			loaded.notifications = options.NotificationSink
			loaded.ensureBootstrapToken(options.BootstrapToken)
			loaded.ensureBootstrapUser(options.BootstrapUserEmail, options.BootstrapUserPassword)
			return loaded, loaded.saveLocked()
		}
	}

	var s *Store
	if options.Seed {
		s = NewSeeded()
	} else {
		s = New()
	}
	s.path = options.Path
	s.persistence = persistence
	s.telemetry = options.TelemetrySink
	s.notifications = options.NotificationSink
	s.ensureBootstrapToken(options.BootstrapToken)
	s.ensureBootstrapUser(options.BootstrapUserEmail, options.BootstrapUserPassword)
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() {
	if closer, ok := s.persistence.(closeableStateStore); ok {
		closer.Close()
	}
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

func stateStoreFromOptions(options Options) (stateStore, error) {
	backend := strings.ToLower(options.Backend)
	if backend == "" {
		backend = "json"
	}
	switch backend {
	case "json", "file":
		if options.Path == "" {
			return nil, nil
		}
		return fileStateStore{path: options.Path}, nil
	case "postgres", "postgresql":
		return newPostgresStateStore(PostgresOptions{URL: options.PostgresURL, Timeout: options.PostgresTimeout})
	default:
		return nil, errors.New("unsupported store backend: " + options.Backend)
	}
}

type fileStateStore struct {
	path string
}

func (store fileStateStore) Load() (snapshot, bool, error) {
	snap, err := loadSnapshot(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return snapshot{}, false, nil
	}
	return snap, err == nil, err
}

func (store fileStateStore) Save(snap snapshot) error {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
		return err
	}
	tmp := store.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, store.path)
}

func loadSnapshot(path string) (snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return snapshot{}, err
	}
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return snapshot{}, err
	}
	return snap, nil
}

func storeFromSnapshot(snap snapshot) *Store {
	s := New()
	s.org = snap.Organization
	s.envs = snap.Environments
	s.tokens = snap.Tokens
	s.users = snap.Users
	s.sessions = snap.Sessions
	s.services = snap.Services
	s.hosts = snap.Hosts
	s.metrics = snap.Metrics
	s.logs = snap.Logs
	s.traces = snap.Traces
	s.alerts = snap.Alerts
	s.incidents = snap.Incidents
	s.uptime = snap.Uptime
	s.alertRules = snap.AlertRules
	s.channels = snap.Channels
	s.audit = snap.Audit
	s.normalizeLoaded()
	return s
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
	if s.users == nil {
		s.users = make(map[string]User)
	}
	if s.sessions == nil {
		s.sessions = make(map[string]Session)
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
	if s.alertRules == nil {
		s.alertRules = make(map[string]AlertRule)
	}
	if s.channels == nil {
		s.channels = make(map[string]NotificationChannel)
	}
}

func (s *Store) saveLocked() error {
	snap := snapshot{
		Organization: s.org,
		Environments: append([]Environment(nil), s.envs...),
		Tokens:       cloneMapValues(s.tokens),
		Users:        cloneMapValues(s.users),
		Sessions:     cloneMapValues(s.sessions),
		Services:     cloneMapValues(s.services),
		Hosts:        cloneMapValues(s.hosts),
		Metrics:      append([]Metric(nil), s.metrics...),
		Logs:         append([]Log(nil), s.logs...),
		Traces:       append([]Trace(nil), s.traces...),
		Alerts:       cloneMapValues(s.alerts),
		Incidents:    cloneMapValues(s.incidents),
		Uptime:       cloneMapValues(s.uptime),
		AlertRules:   cloneMapValues(s.alertRules),
		Channels:     cloneMapValues(s.channels),
		Audit:        append([]AuditEvent(nil), s.audit...),
	}
	if s.persistence != nil {
		return s.persistence.Save(snap)
	}
	return nil
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

func (s *Store) ensureBootstrapUser(email, password string) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return
	}
	if s.users == nil {
		s.users = make(map[string]User)
	}
	for _, existing := range s.users {
		if existing.Email == email {
			return
		}
	}
	user := User{
		ID:           "user-owner",
		Email:        email,
		DisplayName:  "SignalPlane Owner",
		Role:         "owner",
		PasswordHash: hashPassword(password),
		CreatedAt:    now(),
	}
	s.users[user.ID] = user
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
			Tokens: len(s.tokens), Users: len(s.users), Traces: len(s.traces), Alerts: len(s.alerts), OpenAlerts: open, Incidents: len(s.incidents), UptimeMonitors: len(s.uptime),
			AlertRules: len(s.alertRules), NotificationChannels: len(s.channels),
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

func (s *Store) ValidSession(token, scope string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessionByTokenLocked(token)
	if !ok {
		return false
	}
	user, ok := s.users[session.UserID]
	if !ok || user.DisabledAt != "" {
		return false
	}
	return roleAllows(user.Role, scope)
}

func (s *Store) UserForSession(token string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessionByTokenLocked(token)
	if !ok {
		return User{}, false
	}
	user, ok := s.users[session.UserID]
	if !ok {
		return User{}, false
	}
	user.PasswordHash = ""
	return user, true
}

func (s *Store) sessionByTokenLocked(token string) (Session, bool) {
	if token == "" {
		return Session{}, false
	}
	nowTime := time.Now().UTC()
	for _, session := range s.sessions {
		if session.Token != token || session.RevokedAt != "" {
			continue
		}
		expires, err := time.Parse(time.RFC3339Nano, session.ExpiresAt)
		if err == nil && nowTime.After(expires) {
			return Session{}, false
		}
		return session, true
	}
	return Session{}, false
}

func (s *Store) Authenticate(email, password string) (Session, User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	email = strings.ToLower(strings.TrimSpace(email))
	for _, user := range s.users {
		if user.Email != email || user.DisabledAt != "" {
			continue
		}
		if !verifyPassword(password, user.PasswordHash) {
			break
		}
		session := Session{
			ID:        newID("ses"),
			UserID:    user.ID,
			Token:     newID("sp_session"),
			CreatedAt: now(),
			ExpiresAt: time.Now().UTC().Add(12 * time.Hour).Format(time.RFC3339Nano),
		}
		s.sessions[session.ID] = session
		s.auditLocked("user.login", "user", user.ID, nil)
		_ = s.saveLocked()
		user.PasswordHash = ""
		return session, user, true
	}
	return Session{}, User{}, false
}

func (s *Store) RevokeSession(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, session := range s.sessions {
		if session.Token == token && session.RevokedAt == "" {
			session.RevokedAt = now()
			s.sessions[id] = session
			_ = s.saveLocked()
			return
		}
	}
}

func (s *Store) Users(max int) []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := mapValues(s.users)
	sort.SliceStable(users, func(i, j int) bool { return users[i].Email < users[j].Email })
	for i := range users {
		users[i].PasswordHash = ""
	}
	return take(users, saneLimit(max))
}

func (s *Store) CreateUser(input UserInput) User {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := User{
		ID:           newID("usr"),
		Email:        strings.ToLower(strings.TrimSpace(input.Email)),
		DisplayName:  firstNonEmpty(input.DisplayName, input.Email),
		Role:         normalizeRole(input.Role),
		PasswordHash: hashPassword(input.Password),
		CreatedAt:    now(),
	}
	s.users[user.ID] = user
	s.auditLocked("user.created", "user", user.ID, map[string]string{"role": user.Role})
	_ = s.saveLocked()
	user.PasswordHash = ""
	return user
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

func (s *Store) AlertRules(max int) []AlertRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rules := mapValues(s.alertRules)
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].Name < rules[j].Name })
	return take(rules, saneLimit(max))
}

func (s *Store) CreateAlertRule(input AlertRuleInput) AlertRule {
	s.mu.Lock()
	defer s.mu.Unlock()
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	rule := AlertRule{
		ID:          newID("arl"),
		Name:        firstNonEmpty(input.Name, "alert-rule"),
		SignalType:  normalizeSignalType(input.SignalType),
		Enabled:     enabled,
		MetricName:  input.MetricName,
		LogSeverity: normalizeSeverity(input.LogSeverity),
		Query:       input.Query,
		Operator:    normalizeOperator(input.Operator),
		Threshold:   input.Threshold,
		Severity:    normalizeAlertSeverity(input.Severity),
		Labels:      cloneMap(input.Labels),
		CreatedAt:   now(),
		UpdatedAt:   now(),
	}
	s.alertRules[rule.ID] = rule
	s.auditLocked("alert_rule.created", "alertRule", rule.ID, map[string]string{"signalType": rule.SignalType})
	_ = s.saveLocked()
	return rule
}

func (s *Store) NotificationChannels(max int) []NotificationChannel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	channels := mapValues(s.channels)
	sort.SliceStable(channels, func(i, j int) bool { return channels[i].Name < channels[j].Name })
	return take(channels, saneLimit(max))
}

func (s *Store) NotificationChannel(id string) (NotificationChannel, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	channel, ok := s.channels[id]
	return channel, ok
}

func (s *Store) CreateNotificationChannel(input NotificationChannelInput) NotificationChannel {
	s.mu.Lock()
	defer s.mu.Unlock()
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	channel := NotificationChannel{
		ID:        newID("chn"),
		Name:      firstNonEmpty(input.Name, input.Target, "notification-channel"),
		Type:      normalizeChannelType(input.Type),
		Target:    input.Target,
		Enabled:   enabled,
		Config:    cloneMap(input.Config),
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	s.channels[channel.ID] = channel
	s.auditLocked("notification_channel.created", "notificationChannel", channel.ID, map[string]string{"type": channel.Type})
	_ = s.saveLocked()
	return channel
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
		s.updateServiceStatsFromMetricLocked(metric)
		s.evaluateMetricAlertLocked(metric)
		s.evaluateMetricRulesLocked(metric)
		out = append(out, metric)
	}
	_ = s.saveLocked()
	sink := s.telemetry
	s.mu.Unlock()
	if sink != nil && len(out) > 0 {
		_ = sink.WriteMetrics(out)
	}
	return out
}

func (s *Store) IngestLogs(inputs []LogInput) []Log {
	s.mu.Lock()
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
		s.evaluateLogRulesLocked(log)
	}
	_ = s.saveLocked()
	sink := s.telemetry
	s.mu.Unlock()
	if sink != nil && len(out) > 0 {
		_ = sink.WriteLogs(out)
	}
	return out
}

func (s *Store) IngestTraces(inputs []TraceInput) []Trace {
	s.mu.Lock()
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
	sink := s.telemetry
	s.mu.Unlock()
	if sink != nil && len(out) > 0 {
		_ = sink.WriteTraces(out)
	}
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

func (s *Store) RecordUptimeResult(result UptimeResult) (UptimeMonitor, bool) {
	s.mu.Lock()
	monitor, ok := s.uptime[result.ID]
	if !ok {
		s.mu.Unlock()
		return UptimeMonitor{}, false
	}
	monitor.Status = coalesce(result.Status, "unknown")
	monitor.LastCheckedAt = coalesce(result.CheckedAt, now())
	monitor.LastStatusCode = result.StatusCode
	monitor.LastResponseMS = result.ResponseMS
	monitor.LastError = result.Error
	if monitor.Status == "up" {
		monitor.ConsecutiveFailures = 0
	} else {
		monitor.ConsecutiveFailures++
		s.createAlertOnceLocked(Alert{
			Title:    "Uptime check failed: " + monitor.Name,
			Severity: "critical",
			Source:   "uptime",
			EntityID: monitor.ID,
			Message:  firstNonEmpty(monitor.LastError, "expected status "+strconv.Itoa(monitor.ExpectedStatus)+", got "+strconv.Itoa(monitor.LastStatusCode)),
			Labels:   map[string]string{"monitor": monitor.Name, "url": monitor.URL},
		})
	}
	s.uptime[monitor.ID] = monitor
	s.auditLocked("uptime.checked", "uptimeMonitor", monitor.ID, map[string]string{"status": monitor.Status})
	_ = s.saveLocked()
	sink := s.telemetry
	s.mu.Unlock()
	if sink != nil {
		_ = sink.WriteUptimeResult(monitor)
	}
	return monitor, true
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
	services := mapValues(s.services)
	sort.SliceStable(services, func(i, j int) bool { return services[i].Name < services[j].Name })
	return take(services, saneLimit(max))
}
func (s *Store) Hosts(max int) []Host {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hosts := mapValues(s.hosts)
	sort.SliceStable(hosts, func(i, j int) bool { return hosts[i].Name < hosts[j].Name })
	return take(hosts, saneLimit(max))
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
	monitors := mapValues(s.uptime)
	sort.SliceStable(monitors, func(i, j int) bool { return monitors[i].Name < monitors[j].Name })
	return take(monitors, saneLimit(max))
}
func (s *Store) UptimeMonitor(id string) (UptimeMonitor, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	monitor, ok := s.uptime[id]
	return monitor, ok
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
		status := ""
		if _, exists := s.services[resource.Service]; !exists {
			status = "healthy"
		}
		s.upsertServiceLocked(Service{ID: resource.Service, Name: resource.Service, Environment: firstNonEmpty(resource.Environment, "production"), Region: firstNonEmpty(resource.Region, "local"), Status: status, Version: firstNonEmpty(resource.Version, "unknown")})
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
	s.notifyAlertLocked(alert)
	return alert
}

func (s *Store) createAlertOnceLocked(input Alert) Alert {
	for _, existing := range s.alerts {
		if existing.Status == "open" && existing.Source == input.Source && existing.EntityID == input.EntityID && existing.Title == input.Title {
			existing.Timestamp = now()
			existing.Severity = normalizeAlertSeverity(firstNonEmpty(input.Severity, existing.Severity))
			existing.Message = firstNonEmpty(input.Message, existing.Message)
			if input.Labels != nil {
				existing.Labels = input.Labels
			}
			s.alerts[existing.ID] = existing
			s.notifyAlertLocked(existing)
			return existing
		}
	}
	return s.createAlertLocked(input)
}

func (s *Store) notifyAlertLocked(alert Alert) {
	if s.notifications == nil {
		return
	}
	channels := make([]NotificationChannel, 0, len(s.channels))
	for _, channel := range s.channels {
		if channel.Enabled {
			channels = append(channels, channel)
		}
	}
	if len(channels) == 0 {
		return
	}
	go s.notifications.NotifyAlert(alert, channels)
}

func (s *Store) updateServiceStatsFromMetricLocked(metric Metric) {
	if metric.Resource.Service == "" {
		return
	}
	name := strings.ToLower(metric.Name)
	service := s.upsertServiceLocked(Service{ID: metric.Resource.Service, Name: metric.Resource.Service, Environment: metric.Resource.Environment, Region: metric.Resource.Region, Version: metric.Resource.Version})
	stats := service.Stats
	switch {
	case strings.Contains(name, "request_rate") || strings.Contains(name, "request.rate"):
		stats.RequestRate = metric.Value
	case strings.Contains(name, "error_rate") || strings.Contains(name, "error.rate"):
		stats.ErrorRate = metric.Value
	case strings.Contains(name, "duration") || strings.Contains(name, "latency"):
		stats.P95LatencyMS = metric.Value
	}

	status := service.Status
	if stats.ErrorRate >= 5 || stats.P95LatencyMS >= 750 {
		status = "degraded"
	} else if status == "" {
		status = "healthy"
	}
	s.upsertServiceLocked(Service{ID: service.ID, Name: service.Name, Environment: service.Environment, Region: service.Region, Version: service.Version, Status: status, Stats: stats})
}

func (s *Store) evaluateMetricAlertLocked(metric Metric) {
	name := strings.ToLower(metric.Name)
	entity := firstNonEmpty(metric.Resource.Service, metric.Resource.Host, metric.Name)
	switch {
	case strings.Contains(name, "error_rate") && metric.Value >= 5:
		severity := "warning"
		if metric.Value >= 10 {
			severity = "critical"
		}
		s.createAlertOnceLocked(Alert{
			Title:    "High error rate in " + entity,
			Severity: severity,
			Source:   "metrics",
			EntityID: entity,
			Message:  metric.Name + " is " + formatFloat(metric.Value) + "%",
			Labels:   map[string]string{"metric": metric.Name},
		})
	case (strings.Contains(name, "duration") || strings.Contains(name, "latency")) && strings.EqualFold(metric.Unit, "ms") && metric.Value >= 500:
		severity := "warning"
		if metric.Value >= 1000 {
			severity = "critical"
		}
		s.createAlertOnceLocked(Alert{
			Title:    "High latency in " + entity,
			Severity: severity,
			Source:   "metrics",
			EntityID: entity,
			Message:  metric.Name + " is " + formatFloat(metric.Value) + " ms",
			Labels:   map[string]string{"metric": metric.Name},
		})
	case strings.Contains(name, "cpu") && strings.EqualFold(metric.Unit, "percent") && metric.Value >= 90:
		s.createAlertOnceLocked(Alert{
			Title:    "High CPU on " + entity,
			Severity: "critical",
			Source:   "metrics",
			EntityID: entity,
			Message:  metric.Name + " is " + formatFloat(metric.Value) + "%",
			Labels:   map[string]string{"metric": metric.Name},
		})
	}
}

func (s *Store) evaluateMetricRulesLocked(metric Metric) {
	for _, rule := range s.alertRules {
		if !rule.Enabled || rule.SignalType != "metric" {
			continue
		}
		if rule.MetricName != "" && rule.MetricName != metric.Name {
			continue
		}
		if !compare(metric.Value, rule.Operator, rule.Threshold) {
			continue
		}
		entity := firstNonEmpty(metric.Resource.Service, metric.Resource.Host, metric.Name)
		s.createAlertOnceLocked(Alert{
			Title:    firstNonEmpty(rule.Name, "Metric rule triggered") + ": " + metric.Name,
			Severity: rule.Severity,
			Source:   "alert-rule",
			EntityID: entity,
			Message:  metric.Name + " is " + formatFloat(metric.Value) + " " + metric.Unit + " (" + rule.Operator + " " + formatFloat(rule.Threshold) + ")",
			Labels:   mergeStringMaps(rule.Labels, map[string]string{"rule": rule.ID, "metric": metric.Name, "signal": "metric"}),
		})
	}
}

func (s *Store) evaluateLogRulesLocked(log Log) {
	for _, rule := range s.alertRules {
		if !rule.Enabled || rule.SignalType != "log" {
			continue
		}
		if rule.LogSeverity != "" && rule.LogSeverity != "info" && rule.LogSeverity != log.Severity {
			continue
		}
		if rule.Query != "" && !strings.Contains(strings.ToLower(log.Message), strings.ToLower(rule.Query)) {
			continue
		}
		entity := firstNonEmpty(log.Resource.Service, log.Resource.Host, "logs")
		s.createAlertOnceLocked(Alert{
			Title:        firstNonEmpty(rule.Name, "Log rule triggered"),
			Severity:     rule.Severity,
			Source:       "alert-rule",
			EntityID:     entity,
			Message:      log.Message,
			RelatedLogID: log.ID,
			Labels:       mergeStringMaps(rule.Labels, map[string]string{"rule": rule.ID, "severity": log.Severity, "signal": "log"}),
		})
	}
}

func compare(value float64, operator string, threshold float64) bool {
	switch normalizeOperator(operator) {
	case "gt":
		return value > threshold
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	default:
		return value >= threshold
	}
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
func normalizeSignalType(value string) string {
	switch strings.ToLower(value) {
	case "metric", "log":
		return strings.ToLower(value)
	default:
		return "metric"
	}
}
func normalizeOperator(value string) string {
	switch strings.ToLower(value) {
	case "gt", "gte", "lt", "lte", "eq":
		return strings.ToLower(value)
	default:
		return "gte"
	}
}
func normalizeChannelType(value string) string {
	switch strings.ToLower(value) {
	case "email", "webhook", "slack_webhook":
		return strings.ToLower(value)
	default:
		return "webhook"
	}
}
func normalizeRole(value string) string {
	switch strings.ToLower(value) {
	case "owner", "admin", "editor", "viewer":
		return strings.ToLower(value)
	default:
		return "viewer"
	}
}
func roleAllows(role, scope string) bool {
	switch normalizeRole(role) {
	case "owner", "admin":
		return true
	case "editor":
		return scope == "ingest" || scope == "read"
	case "viewer":
		return scope == "read"
	default:
		return false
	}
}
func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hash)
}
func verifyPassword(password, encoded string) bool {
	if encoded == "" {
		return false
	}
	if strings.HasPrefix(encoded, "$2a$") || strings.HasPrefix(encoded, "$2b$") || strings.HasPrefix(encoded, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(encoded), []byte(password)) == nil
	}
	parts := strings.SplitN(encoded, ":", 2)
	if len(parts) != 2 {
		return false
	}
	sum := sha256.Sum256([]byte(parts[0] + ":" + password))
	return subtle.ConstantTimeCompare([]byte(parts[1]), []byte(hex.EncodeToString(sum[:]))) == 1
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
func mergeStringMaps(inputs ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, input := range inputs {
		for key, value := range input {
			out[key] = value
		}
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
func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 1, 64)
}
