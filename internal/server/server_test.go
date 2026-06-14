package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chaktihor/signalplane/internal/platform"
	"github.com/chaktihor/signalplane/internal/store"
)

func TestHealthAndBootstrap(t *testing.T) {
	app := New(Config{IngestToken: "test-token"}, store.NewSeeded(), slog.Default())

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected health 200, got %d", resp.Code)
	}

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/bootstrap", nil)
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected bootstrap 200, got %d", resp.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode bootstrap: %v", err)
	}
	if body["health"] == "" {
		t.Fatal("expected health in bootstrap payload")
	}
}

func TestIngestRequiresToken(t *testing.T) {
	app := New(Config{IngestToken: "test-token"}, store.New(), slog.Default())
	payload := []byte(`{"severity":"info","message":"hello","resource":{"service":"api"}}`)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ingest/logs", bytes.NewReader(payload))
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without token, got %d", resp.Code)
	}

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/ingest/logs", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer test-token")
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected accepted with token, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestStateChangingEndpointsRequireAdminToken(t *testing.T) {
	app := New(Config{IngestToken: "test-token"}, store.New(), slog.Default())

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/incidents", bytes.NewReader([]byte(`{"title":"database degraded"}`)))
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected incident creation to require admin token, got %d", resp.Code)
	}

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/incidents", bytes.NewReader([]byte(`{"title":"database degraded"}`)))
	req.Header.Set("Authorization", "Bearer test-token")
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected incident creation with admin token, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestManualUptimeCheck(t *testing.T) {
	data := store.New()
	data.CreateUptimeMonitor(store.UptimeMonitor{ID: "upt-test", Name: "test target", URL: "://bad-url", ExpectedStatus: http.StatusOK})
	app := New(Config{IngestToken: "test-token"}, data, slog.Default())

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/uptime-monitors/upt-test/check", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected uptime check 200, got %d: %s", resp.Code, resp.Body.String())
	}

	monitor, ok := data.UptimeMonitor("upt-test")
	if !ok {
		t.Fatal("expected uptime monitor to exist")
	}
	if monitor.Status != "down" || monitor.LastError == "" {
		t.Fatalf("expected down monitor with error, got status=%q error=%q", monitor.Status, monitor.LastError)
	}
}

func TestDependencyEndpoint(t *testing.T) {
	app := New(Config{Dependencies: []platform.DependencyCheck{{ID: "bad", Name: "Bad TCP", Kind: "tcp", Target: "127.0.0.1:1"}}}, store.New(), slog.Default())

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/system/dependencies", nil)
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected dependency endpoint 200, got %d", resp.Code)
	}
	var body struct {
		Dependencies []platform.DependencyStatus `json:"dependencies"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode dependency response: %v", err)
	}
	if len(body.Dependencies) != 1 {
		t.Fatalf("expected one dependency status, got %d", len(body.Dependencies))
	}
	if body.Dependencies[0].Status != "down" {
		t.Fatalf("expected dependency to be down, got %q", body.Dependencies[0].Status)
	}
}
