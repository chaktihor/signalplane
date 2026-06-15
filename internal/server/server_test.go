package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chaktihor/signalplane/internal/platform"
	"github.com/chaktihor/signalplane/internal/store"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
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

func TestOTLPHTTPProtobufIngestion(t *testing.T) {
	data := store.New()
	app := New(Config{IngestToken: "test-token"}, data, slog.Default())
	now := uint64(time.Now().UTC().UnixNano())
	resource := &resourcepb.Resource{Attributes: []*commonpb.KeyValue{
		{Key: "service.name", Value: stringValue("proto-api")},
		{Key: "host.name", Value: stringValue("proto-host")},
		{Key: "deployment.environment", Value: stringValue("test")},
	}}

	postProto(t, app, "/v1/logs", &logspb.LogsData{
		ResourceLogs: []*logspb.ResourceLogs{{
			Resource: resource,
			ScopeLogs: []*logspb.ScopeLogs{{
				LogRecords: []*logspb.LogRecord{{
					TimeUnixNano:   now,
					SeverityNumber: logspb.SeverityNumber_SEVERITY_NUMBER_ERROR,
					Body:           stringValue("proto failure"),
					TraceId:        []byte("1234567890123456"),
					SpanId:         []byte("12345678"),
					Attributes: []*commonpb.KeyValue{
						{Key: "component", Value: stringValue("checkout")},
					},
				}},
			}},
		}},
	})
	if logs := data.Logs(10, "proto-api", "error", "proto failure"); len(logs) != 1 {
		t.Fatalf("expected one protobuf log, got %d", len(logs))
	}

	postProto(t, app, "/v1/metrics", &metricspb.MetricsData{
		ResourceMetrics: []*metricspb.ResourceMetrics{{
			Resource: resource,
			ScopeMetrics: []*metricspb.ScopeMetrics{{
				Metrics: []*metricspb.Metric{{
					Name: "http.server.requests",
					Unit: "1",
					Data: &metricspb.Metric_Gauge{Gauge: &metricspb.Gauge{DataPoints: []*metricspb.NumberDataPoint{{
						TimeUnixNano: now,
						Value:        &metricspb.NumberDataPoint_AsDouble{AsDouble: 42},
						Attributes: []*commonpb.KeyValue{
							{Key: "route", Value: stringValue("/checkout")},
						},
					}}}},
				}},
			}},
		}},
	})
	if metrics := data.Metrics(10); len(metrics) != 1 || metrics[0].Name != "http.server.requests" || metrics[0].Value != 42 {
		t.Fatalf("expected protobuf metric, got %#v", metrics)
	}

	postProto(t, app, "/v1/traces", &tracepb.TracesData{
		ResourceSpans: []*tracepb.ResourceSpans{{
			Resource: resource,
			ScopeSpans: []*tracepb.ScopeSpans{{
				Spans: []*tracepb.Span{{
					TraceId:           []byte("abcdefghijklmnop"),
					SpanId:            []byte("span0001"),
					Name:              "GET /checkout",
					StartTimeUnixNano: now,
					EndTimeUnixNano:   now + uint64((2 * time.Millisecond).Nanoseconds()),
					Status:            &tracepb.Status{Code: tracepb.Status_STATUS_CODE_ERROR},
				}},
			}},
		}},
	})
	if traces := data.Traces(10, "proto-api", "error"); len(traces) != 1 {
		t.Fatalf("expected one protobuf trace, got %d", len(traces))
	}
}

func postProto(t *testing.T, app *Server, path string, payload proto.Message) {
	t.Helper()
	body, err := proto.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal protobuf: %v", err)
	}
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/x-protobuf")
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected protobuf OTLP 200 for %s, got %d: %s", path, resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Content-Type"); got != "application/x-protobuf" {
		t.Fatalf("expected protobuf response content type, got %q", got)
	}
}

func stringValue(value string) *commonpb.AnyValue {
	return &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: value}}
}
