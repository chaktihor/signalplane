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
