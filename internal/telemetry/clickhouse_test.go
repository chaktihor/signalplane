package telemetry

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/chaktihor/signalplane/internal/store"
)

func TestNewClickHouseSinkNormalizesURLAndCredentials(t *testing.T) {
	sink, err := NewClickHouseSink(ClickHouseOptions{
		URL:      "http://signalplane:secret@clickhouse:8123/ping",
		Database: "signals",
	})
	if err != nil {
		t.Fatalf("NewClickHouseSink returned error: %v", err)
	}
	if sink.baseURL != "http://clickhouse:8123" {
		t.Fatalf("baseURL = %q, want stripped URL", sink.baseURL)
	}
	if sink.username != "signalplane" || sink.password != "secret" {
		t.Fatalf("credentials = %q/%q, want userinfo credentials", sink.username, sink.password)
	}
	if sink.database != "signals" {
		t.Fatalf("database = %q, want signals", sink.database)
	}
}

func TestClickHouseSinkWritesMetricJSONEachRow(t *testing.T) {
	var gotQuery string
	var gotAuth string
	var gotBody string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotQuery = r.URL.Query().Get("query")
		gotAuth = r.Header.Get("Authorization")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		gotBody = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    r,
		}, nil
	})
	sink := &ClickHouseSink{
		baseURL:      "http://clickhouse:8123",
		database:     "signalplane",
		organization: "org-test",
		username:     "signalplane",
		password:     "secret",
		client:       &http.Client{Timeout: time.Second, Transport: transport},
	}
	err := sink.WriteMetrics([]store.Metric{{
		Timestamp: "2026-06-15T01:02:03.123456789Z",
		Name:      "http.server.requests",
		Type:      "counter",
		Unit:      "requests",
		Value:     42,
		Labels:    map[string]string{"route": "/checkout"},
		Resource: store.Resource{
			Service:     "checkout-api",
			Host:        "host-api-1",
			Environment: "production",
			Region:      "local",
			Attributes:  map[string]string{"version": "0.1.0"},
		},
	}})
	if err != nil {
		t.Fatalf("WriteMetrics returned error: %v", err)
	}

	if gotQuery != "INSERT INTO signalplane.metrics FORMAT JSONEachRow" {
		t.Fatalf("query = %q, want metrics insert", gotQuery)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("signalplane:secret"))
	if gotAuth != wantAuth {
		t.Fatalf("authorization = %q, want basic auth", gotAuth)
	}
	lines := strings.Split(strings.TrimSpace(gotBody), "\n")
	if len(lines) != 1 {
		t.Fatalf("body lines = %d, want 1: %q", len(lines), gotBody)
	}
	var row map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &row); err != nil {
		t.Fatalf("unmarshal body row: %v", err)
	}
	assertRowValue(t, row, "timestamp", "2026-06-15 01:02:03.123456789")
	assertRowValue(t, row, "organization_id", "org-test")
	assertRowValue(t, row, "environment", "production")
	assertRowValue(t, row, "service", "checkout-api")
	assertRowValue(t, row, "metric_name", "http.server.requests")
}

func assertRowValue(t *testing.T, row map[string]any, key string, want string) {
	t.Helper()
	got, ok := row[key].(string)
	if !ok || got != want {
		t.Fatalf("row[%q] = %#v, want %q", key, row[key], want)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
