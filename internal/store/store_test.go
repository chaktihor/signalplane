package store

import "testing"

func TestIngestLogsCreatesServiceAndAlert(t *testing.T) {
	s := New()
	logs := s.IngestLogs([]LogInput{{
		Severity: "error",
		Message:  "database timeout",
		Resource: Resource{Service: "orders-api", Host: "orders-1", Environment: "production"},
	}})

	if len(logs) != 1 {
		t.Fatalf("expected one log, got %d", len(logs))
	}
	if got := s.Summary().Counts.Services; got != 1 {
		t.Fatalf("expected service to be inferred, got %d services", got)
	}
	if got := s.Summary().Counts.OpenAlerts; got != 1 {
		t.Fatalf("expected error log to create alert, got %d open alerts", got)
	}
}

func TestIngestTracesFiltersByService(t *testing.T) {
	s := New()
	traces := s.IngestTraces([]TraceInput{{
		TraceID: "trace-1",
		Spans: []SpanInput{{
			SpanID:     "span-1",
			Name:       "GET /health",
			DurationMS: 12,
			Resource:   Resource{Service: "edge-api", Environment: "production"},
		}},
	}})

	if len(traces) != 1 {
		t.Fatalf("expected one trace, got %d", len(traces))
	}
	filtered := s.Traces(10, "edge-api", "")
	if len(filtered) != 1 {
		t.Fatalf("expected filtered trace, got %d", len(filtered))
	}
}

func TestOpenPersistsTelemetryAndTokens(t *testing.T) {
	path := t.TempDir() + "/signalplane.json"
	s, err := Open(Options{Path: path, Seed: false, BootstrapToken: "test-token"})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if !s.ValidToken("test-token", "ingest") {
		t.Fatal("expected bootstrap admin token to allow ingest")
	}
	s.IngestLogs([]LogInput{{
		Severity: "info",
		Message:  "persist me",
		Resource: Resource{Service: "persisted-api", Environment: "production"},
	}})

	reopened, err := Open(Options{Path: path, Seed: false, BootstrapToken: "test-token"})
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	if got := reopened.Summary().Counts.Logs; got != 1 {
		t.Fatalf("expected one persisted log, got %d", got)
	}
	if got := reopened.Summary().Counts.Services; got != 1 {
		t.Fatalf("expected one persisted service, got %d", got)
	}
}

func TestCreateTokenMasksListButValidatesRawToken(t *testing.T) {
	s := New()
	token := s.CreateToken(TokenInput{Name: "collector", Token: "collector-secret", Scope: "ingest"})
	if token.Token != "collector-secret" {
		t.Fatal("created token should return raw token once")
	}
	if !s.ValidToken("collector-secret", "ingest") {
		t.Fatal("expected raw token to validate")
	}
	listed := s.Tokens(10)
	if len(listed) != 1 {
		t.Fatalf("expected one token, got %d", len(listed))
	}
	if listed[0].Token == "collector-secret" {
		t.Fatal("listed token should be masked")
	}
}

func TestMetricIngestionUpdatesServiceStatsAndCreatesAlert(t *testing.T) {
	s := New()
	s.IngestMetrics([]MetricInput{
		{Name: "http.server.error_rate", Value: 12.5, Unit: "percent", Type: "gauge", Resource: Resource{Service: "checkout-api", Environment: "production"}},
		{Name: "http.server.duration", Value: 840, Unit: "ms", Type: "histogram", Resource: Resource{Service: "checkout-api", Environment: "production"}},
	})

	services := s.Services(10)
	if len(services) != 1 {
		t.Fatalf("expected one service, got %d", len(services))
	}
	if services[0].Status != "degraded" {
		t.Fatalf("expected degraded service, got %q", services[0].Status)
	}
	if services[0].Stats.ErrorRate != 12.5 {
		t.Fatalf("expected error-rate stat to update, got %f", services[0].Stats.ErrorRate)
	}
	if got := s.Summary().Counts.OpenAlerts; got == 0 {
		t.Fatal("expected metric threshold alert")
	}
}

func TestRecordUptimeResultUpdatesMonitorAndCreatesAlert(t *testing.T) {
	s := New()
	monitor := s.CreateUptimeMonitor(UptimeMonitor{ID: "upt-demo", Name: "Demo", URL: "http://localhost:8088/healthz", ExpectedStatus: 200})

	updated, ok := s.RecordUptimeResult(UptimeResult{ID: monitor.ID, Status: "down", StatusCode: 503, ResponseMS: 42, Error: "expected status 200, got 503"})
	if !ok {
		t.Fatal("expected uptime monitor to update")
	}
	if updated.Status != "down" || updated.ConsecutiveFailures != 1 {
		t.Fatalf("expected down monitor with one failure, got status=%q failures=%d", updated.Status, updated.ConsecutiveFailures)
	}
	if got := s.Summary().Counts.OpenAlerts; got != 1 {
		t.Fatalf("expected one uptime alert, got %d", got)
	}

	updated, ok = s.RecordUptimeResult(UptimeResult{ID: monitor.ID, Status: "up", StatusCode: 200, ResponseMS: 12})
	if !ok {
		t.Fatal("expected uptime monitor to update")
	}
	if updated.Status != "up" || updated.ConsecutiveFailures != 0 {
		t.Fatalf("expected recovered monitor, got status=%q failures=%d", updated.Status, updated.ConsecutiveFailures)
	}
}

func TestMetricAlertRuleCreatesAlert(t *testing.T) {
	s := New()
	s.CreateAlertRule(AlertRuleInput{
		Name:       "High latency",
		SignalType: "metric",
		MetricName: "http.server.duration",
		Operator:   "gte",
		Threshold:  500,
		Severity:   "critical",
	})
	s.IngestMetrics([]MetricInput{{
		Name: "http.server.duration", Value: 750, Unit: "ms", Type: "histogram",
		Resource: Resource{Service: "checkout-api", Environment: "production"},
	}})
	alerts := s.Alerts(10)
	if len(alerts) == 0 {
		t.Fatal("expected alert rule to create an alert")
	}
	if alerts[0].Source != "alert-rule" {
		t.Fatalf("expected alert-rule source, got %q", alerts[0].Source)
	}
}

func TestAuthenticateCreatesSessionWithRoleScopes(t *testing.T) {
	s, err := Open(Options{Seed: false, BootstrapUserEmail: "owner@example.com", BootstrapUserPassword: "secret"})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	session, user, ok := s.Authenticate("owner@example.com", "secret")
	if !ok {
		t.Fatal("expected bootstrap user to authenticate")
	}
	if user.Role != "owner" {
		t.Fatalf("expected owner role, got %q", user.Role)
	}
	if session.Token == "" {
		t.Fatal("expected session token")
	}
	if !s.ValidSession(session.Token, "admin") {
		t.Fatal("expected owner session to allow admin scope")
	}
}
