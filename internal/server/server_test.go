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
	app := New(Config{IngestToken: "test-token", AllowPublicRead: true}, store.NewSeeded(), slog.Default())

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
	data := store.New()
	data.CreateToken(store.TokenInput{Name: "admin", Token: "admin-token", Scope: "admin"})
	app := New(Config{IngestToken: "ingest-token"}, data, slog.Default())

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/incidents", bytes.NewReader([]byte(`{"title":"database degraded"}`)))
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected incident creation to require admin token, got %d", resp.Code)
	}

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/incidents", bytes.NewReader([]byte(`{"title":"database degraded"}`)))
	req.Header.Set("Authorization", "Bearer ingest-token")
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected incident creation with ingest token to be unauthorized, got %d", resp.Code)
	}

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/incidents", bytes.NewReader([]byte(`{"title":"database degraded"}`)))
	req.Header.Set("Authorization", "Bearer admin-token")
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected incident creation with admin token, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestReadEndpointsCanRequireReadToken(t *testing.T) {
	data := store.NewSeeded()
	data.CreateToken(store.TokenInput{Name: "reader", Token: "read-token", Scope: "read"})
	app := New(Config{IngestToken: "ingest-token"}, data, slog.Default())

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected services to require read auth, got %d", resp.Code)
	}

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/services", nil)
	req.Header.Set("Authorization", "Bearer ingest-token")
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected ingest token not to grant read access, got %d", resp.Code)
	}

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/services", nil)
	req.Header.Set("Authorization", "Bearer read-token")
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected read token to grant services access, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestOpenAPIEndpointReturnsContract(t *testing.T) {
	data := store.New()
	data.CreateToken(store.TokenInput{Name: "reader", Token: "read-token", Scope: "read"})
	app := New(Config{}, data, slog.Default())

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/openapi", nil)
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected openapi to require read auth, got %d", resp.Code)
	}

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/openapi", nil)
	req.Header.Set("Authorization", "Bearer read-token")
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected openapi 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode openapi response: %v", err)
	}
	if body["openapi"] != "3.1.0" {
		t.Fatalf("expected openapi 3.1.0, got %#v", body["openapi"])
	}
	if _, ok := body["security"]; ok {
		t.Fatal("did not expect global OpenAPI security; public operations must remain public unless secured explicitly")
	}
	paths, ok := body["paths"].(map[string]any)
	if !ok {
		t.Fatalf("expected paths object, got %T", body["paths"])
	}
	for _, path := range []string{"/api/logs", "/api/ingest/logs", "/v1/logs", "/api/openapi"} {
		if _, ok := paths[path]; !ok {
			t.Fatalf("expected path %s in openapi contract", path)
		}
	}
	ingestLogs := paths["/api/ingest/logs"].(map[string]any)["post"].(map[string]any)
	if _, ok := ingestLogs["requestBody"].(map[string]any); !ok {
		t.Fatal("expected ingest logs operation to declare a request body")
	}
	alertRuleList := paths["/api/alert-rules"].(map[string]any)["get"].(map[string]any)
	if !operationHasScope(alertRuleList, "admin") {
		t.Fatal("expected alert rule list operation to require admin scope")
	}
	notificationList := paths["/api/notification-channels"].(map[string]any)["get"].(map[string]any)
	if !operationHasScope(notificationList, "admin") {
		t.Fatal("expected notification channel list operation to require admin scope")
	}
	for path, responseSchema := range map[string]string{"/v1/logs": "LogIngestResponse", "/v1/metrics": "MetricIngestResponse", "/v1/traces": "TraceIngestResponse"} {
		operation := paths[path].(map[string]any)["post"].(map[string]any)
		responses := operation["responses"].(map[string]any)
		if _, ok := responses["200"]; !ok {
			t.Fatalf("expected %s to document protobuf 200 success", path)
		}
		if _, ok := responses["202"]; !ok {
			t.Fatalf("expected %s to document JSON 202 accepted success", path)
		}
		requestBody := operation["requestBody"].(map[string]any)
		requestContent := requestBody["content"].(map[string]any)
		protobufRequest := requestContent["application/x-protobuf"].(map[string]any)
		assertBinarySchema(t, protobufRequest["schema"], path+" protobuf request")
		response200 := responses["200"].(map[string]any)
		content := response200["content"].(map[string]any)
		protobufResponse, ok := content["application/x-protobuf"].(map[string]any)
		if !ok {
			t.Fatalf("expected %s 200 response to document application/x-protobuf", path)
		}
		assertBinarySchema(t, protobufResponse["schema"], path+" protobuf response")
		response202 := responses["202"].(map[string]any)
		jsonContent := response202["content"].(map[string]any)["application/json"].(map[string]any)
		if got := schemaRef(t, jsonContent["schema"]); got != "#/components/schemas/"+responseSchema {
			t.Fatalf("expected %s 202 response schema %s, got %s", path, responseSchema, got)
		}
	}
	components := body["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	for _, schema := range []string{"LogInput", "Log", "MetricInput", "TraceInput", "Error"} {
		if _, ok := schemas[schema]; !ok {
			t.Fatalf("expected schema %s in openapi contract", schema)
		}
	}
	spanProps := schemaProperties(t, schemas["SpanInput"])
	if _, ok := spanProps["spanId"]; !ok {
		t.Fatal("expected SpanInput to use spanId")
	}
	if _, ok := spanProps["id"]; ok {
		t.Fatal("did not expect SpanInput to expose id; ingest uses spanId")
	}
	if _, ok := spanProps["service"]; ok {
		t.Fatal("did not expect SpanInput to expose service; ingest uses resource")
	}
	alertRuleProps := schemaProperties(t, schemas["AlertRuleInput"])
	if _, ok := alertRuleProps["signalType"]; !ok {
		t.Fatal("expected AlertRuleInput to use signalType")
	}
	if _, ok := alertRuleProps["signal"]; ok {
		t.Fatal("did not expect AlertRuleInput to expose stale signal field")
	}
	notificationProps := schemaProperties(t, schemas["NotificationChannelInput"])
	if _, ok := notificationProps["type"]; !ok {
		t.Fatal("expected NotificationChannelInput to use type")
	}
	if _, ok := notificationProps["kind"]; ok {
		t.Fatal("did not expect NotificationChannelInput to expose stale kind field")
	}
	uptimeResponseProps := schemaProperties(t, schemas["UptimeMonitorResponse"])
	if _, ok := uptimeResponseProps["uptimeMonitor"]; !ok {
		t.Fatal("expected uptime response envelope to use uptimeMonitor")
	}
	notificationResponseProps := schemaProperties(t, schemas["NotificationChannelResponse"])
	if _, ok := notificationResponseProps["notificationChannel"]; !ok {
		t.Fatal("expected notification response envelope to use notificationChannel")
	}
	securitySchemes := components["securitySchemes"].(map[string]any)
	if _, ok := securitySchemes["bearerAuth"]; !ok {
		t.Fatal("expected bearerAuth security scheme")
	}
	if _, ok := securitySchemes["signalplaneToken"]; !ok {
		t.Fatal("expected signalplaneToken security scheme")
	}
}

type openAPIRouteExpectation struct {
	method   string
	path     string
	scope    string
	statuses []string
}

func TestOpenAPIRouteMethodStatusAndAuthParity(t *testing.T) {
	data := store.New()
	data.CreateToken(store.TokenInput{Name: "reader", Token: "read-token", Scope: "read"})
	app := New(Config{}, data, slog.Default())

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/openapi", nil)
	req.Header.Set("Authorization", "Bearer read-token")
	app.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected openapi 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode openapi response: %v", err)
	}
	if _, ok := body["security"]; ok {
		t.Fatal("did not expect global security; route auth parity must be operation-specific")
	}
	paths, ok := body["paths"].(map[string]any)
	if !ok {
		t.Fatalf("expected paths object, got %T", body["paths"])
	}

	expectedRoutes := []openAPIRouteExpectation{
		openAPIRoute("get", "/healthz", "", "200"),
		openAPIRoute("post", "/api/auth/login", "", "200", "401"),
		openAPIRoute("post", "/api/auth/logout", "", "204", "200"),
		openAPIRoute("get", "/api/me", "session", "200", "401"),
		openAPIRoute("get", "/api/bootstrap", "read", "200", "401"),
		openAPIRoute("get", "/api/services", "read", "200", "401"),
		openAPIRoute("get", "/api/hosts", "read", "200", "401"),
		openAPIRoute("get", "/api/metrics", "read", "200", "401"),
		openAPIRoute("get", "/api/logs", "read", "200", "401"),
		openAPIRoute("get", "/api/traces", "read", "200", "401"),
		openAPIRoute("get", "/api/alerts", "read", "200", "401"),
		openAPIRoute("patch", "/api/alerts/{id}", "admin", "200", "401", "404"),
		openAPIRoute("get", "/api/alert-rules", "admin", "200", "401"),
		openAPIRoute("post", "/api/alert-rules", "admin", "201", "401"),
		openAPIRoute("get", "/api/incidents", "read", "200", "401"),
		openAPIRoute("post", "/api/incidents", "admin", "201", "401"),
		openAPIRoute("get", "/api/uptime-monitors", "read", "200", "401"),
		openAPIRoute("post", "/api/uptime-monitors", "admin", "201", "401"),
		openAPIRoute("post", "/api/uptime-monitors/{id}/check", "admin", "200", "401", "404"),
		openAPIRoute("get", "/api/notification-channels", "admin", "200", "401"),
		openAPIRoute("post", "/api/notification-channels", "admin", "201", "401"),
		openAPIRoute("post", "/api/notification-channels/{id}/test", "admin", "200", "401", "404"),
		openAPIRoute("get", "/api/system/dependencies", "read", "200", "401"),
		openAPIRoute("get", "/api/tokens", "admin", "200", "401"),
		openAPIRoute("post", "/api/tokens", "admin", "201", "401"),
		openAPIRoute("get", "/api/users", "admin", "200", "401"),
		openAPIRoute("post", "/api/users", "admin", "201", "401"),
		openAPIRoute("post", "/api/ingest/hosts", "ingest", "202", "401"),
		openAPIRoute("post", "/api/ingest/metrics", "ingest", "202", "401"),
		openAPIRoute("post", "/api/ingest/logs", "ingest", "202", "401"),
		openAPIRoute("post", "/api/ingest/traces", "ingest", "202", "401"),
		openAPIRoute("post", "/v1/metrics", "ingest", "200", "202", "400", "401", "415"),
		openAPIRoute("post", "/v1/logs", "ingest", "200", "202", "400", "401", "415"),
		openAPIRoute("post", "/v1/traces", "ingest", "200", "202", "400", "401", "415"),
		openAPIRoute("get", "/api/openapi", "read", "200", "401"),
	}

	expectedMethodsByPath := map[string]map[string]bool{}
	for _, route := range expectedRoutes {
		operation := openAPIOperation(t, paths, route.method, route.path)
		assertOpenAPIAuthScope(t, operation, route)
		assertOpenAPIStatuses(t, operation, route)
		if expectedMethodsByPath[route.path] == nil {
			expectedMethodsByPath[route.path] = map[string]bool{}
		}
		expectedMethodsByPath[route.path][route.method] = true
	}

	if len(paths) != len(expectedMethodsByPath) {
		t.Fatalf("expected %d documented paths, got %d", len(expectedMethodsByPath), len(paths))
	}
	for path, itemValue := range paths {
		item, ok := itemValue.(map[string]any)
		if !ok {
			t.Fatalf("expected path %s item object, got %T", path, itemValue)
		}
		expectedMethods, ok := expectedMethodsByPath[path]
		if !ok {
			t.Fatalf("unexpected OpenAPI path %s", path)
		}
		if len(item) != len(expectedMethods) {
			t.Fatalf("expected %s to document %d methods, got %d", path, len(expectedMethods), len(item))
		}
		for method := range item {
			if !expectedMethods[method] {
				t.Fatalf("unexpected OpenAPI method %s %s", method, path)
			}
		}
	}
}

func openAPIRoute(method, path, scope string, statuses ...string) openAPIRouteExpectation {
	return openAPIRouteExpectation{method: method, path: path, scope: scope, statuses: statuses}
}

func assertBinarySchema(t *testing.T, schema any, label string) {
	t.Helper()
	body, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("expected %s schema object, got %T", label, schema)
	}
	if body["type"] != "string" || body["format"] != "binary" {
		t.Fatalf("expected %s schema to be raw binary string, got %#v", label, body)
	}
	if _, ok := body["contentEncoding"]; ok {
		t.Fatalf("did not expect %s schema to use contentEncoding/base64", label)
	}
}

func schemaRef(t *testing.T, schema any) string {
	t.Helper()
	body, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("expected schema object, got %T", schema)
	}
	ref, ok := body["$ref"].(string)
	if !ok {
		t.Fatalf("expected schema $ref, got %#v", body)
	}
	return ref
}

func openAPIOperation(t *testing.T, paths map[string]any, method, path string) map[string]any {
	t.Helper()
	item, ok := paths[path].(map[string]any)
	if !ok {
		t.Fatalf("expected OpenAPI path %s", path)
	}
	operation, ok := item[method].(map[string]any)
	if !ok {
		t.Fatalf("expected OpenAPI operation %s %s", method, path)
	}
	return operation
}

func assertOpenAPIStatuses(t *testing.T, operation map[string]any, route openAPIRouteExpectation) {
	t.Helper()
	responses, ok := operation["responses"].(map[string]any)
	if !ok {
		t.Fatalf("expected %s %s responses object, got %T", route.method, route.path, operation["responses"])
	}
	for _, status := range route.statuses {
		if _, ok := responses[status]; !ok {
			t.Fatalf("expected %s %s to document response status %s", route.method, route.path, status)
		}
	}
}

func assertOpenAPIAuthScope(t *testing.T, operation map[string]any, route openAPIRouteExpectation) {
	t.Helper()
	scopes := operationSecurityScopes(t, operation)
	switch route.scope {
	case "":
		if len(scopes) != 0 {
			t.Fatalf("expected %s %s to be public, got security %#v", route.method, route.path, scopes)
		}
	case "session":
		if len(scopes) != 1 {
			t.Fatalf("expected %s %s to be session-only, got security %#v", route.method, route.path, scopes)
		}
		values, ok := scopes["sessionCookie"]
		if !ok || !containsString(values, "read") {
			t.Fatalf("expected %s %s to require sessionCookie read auth, got %#v", route.method, route.path, scopes)
		}
		if _, ok := scopes["bearerAuth"]; ok {
			t.Fatalf("did not expect %s %s to accept bearer auth", route.method, route.path)
		}
		if _, ok := scopes["signalplaneToken"]; ok {
			t.Fatalf("did not expect %s %s to accept signalplane token auth", route.method, route.path)
		}
	default:
		for _, scheme := range []string{"bearerAuth", "signalplaneToken", "sessionCookie"} {
			values, ok := scopes[scheme]
			if !ok || !containsString(values, route.scope) {
				t.Fatalf("expected %s %s to require %s scope on %s, got %#v", route.method, route.path, route.scope, scheme, scopes)
			}
		}
	}
}

func operationSecurityScopes(t *testing.T, operation map[string]any) map[string][]string {
	t.Helper()
	result := map[string][]string{}
	security, ok := operation["security"].([]any)
	if !ok {
		return result
	}
	for _, item := range security {
		scheme, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("expected security requirement object, got %T", item)
		}
		for name, rawScopes := range scheme {
			values, ok := rawScopes.([]any)
			if !ok {
				t.Fatalf("expected security scopes for %s, got %T", name, rawScopes)
			}
			for _, value := range values {
				scope, ok := value.(string)
				if !ok {
					t.Fatalf("expected security scope string for %s, got %T", name, value)
				}
				result[name] = append(result[name], scope)
			}
		}
	}
	return result
}

func operationHasScope(operation map[string]any, scope string) bool {
	security, ok := operation["security"].([]any)
	if !ok {
		return false
	}
	for _, item := range security {
		scheme, ok := item.(map[string]any)
		if !ok {
			continue
		}
		for _, scopes := range scheme {
			values, ok := scopes.([]any)
			if !ok {
				continue
			}
			for _, value := range values {
				if value == scope {
					return true
				}
			}
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func schemaProperties(t *testing.T, schema any) map[string]any {
	t.Helper()
	body, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("expected schema object, got %T", schema)
	}
	properties, ok := body["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected schema properties, got %T", body["properties"])
	}
	return properties
}

func TestManualUptimeCheck(t *testing.T) {
	data := store.New()
	data.CreateToken(store.TokenInput{Name: "admin", Token: "admin-token", Scope: "admin"})
	data.CreateUptimeMonitor(store.UptimeMonitor{ID: "upt-test", Name: "test target", URL: "://bad-url", ExpectedStatus: http.StatusOK})
	app := New(Config{IngestToken: "ingest-token"}, data, slog.Default())

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/uptime-monitors/upt-test/check", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
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
	app := New(Config{AllowPublicRead: true, Dependencies: []platform.DependencyCheck{{ID: "bad", Name: "Bad TCP", Kind: "tcp", Target: "127.0.0.1:1"}}}, store.New(), slog.Default())

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

	postProto(t, app, "/v1/logs", &logspb.LogsData{
		ResourceLogs: []*logspb.ResourceLogs{{
			Resource: &resourcepb.Resource{Attributes: []*commonpb.KeyValue{
				{Key: "service.name", Value: stringValue("agent-checkout-api")},
			}},
			ScopeLogs: []*logspb.ScopeLogs{{
				LogRecords: []*logspb.LogRecord{{
					TimeUnixNano:   now + 1,
					SeverityNumber: logspb.SeverityNumber_SEVERITY_NUMBER_INFO,
					Body:           stringValue("agent filelog record"),
					Attributes: []*commonpb.KeyValue{
						{Key: "traceId", Value: stringValue("00000000000000000000000000000001")},
						{Key: "spanId", Value: stringValue("0000000000000001")},
					},
				}},
			}},
		}},
	})
	agentLogs := data.Logs(10, "agent-checkout-api", "info", "agent filelog")
	if len(agentLogs) != 1 {
		t.Fatalf("expected one agent log, got %d", len(agentLogs))
	}
	if agentLogs[0].TraceID != "00000000000000000000000000000001" || agentLogs[0].SpanID != "0000000000000001" {
		t.Fatalf("expected agent trace/span promotion, got trace=%q span=%q", agentLogs[0].TraceID, agentLogs[0].SpanID)
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
