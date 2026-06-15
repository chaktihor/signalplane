package server

func signalPlaneOpenAPI() map[string]any {
	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "SignalPlane Silver API",
			"version":     "0.1.0",
			"description": "Authenticated ingestion, read, configuration, and platform-health APIs for SignalPlane Silver.",
		},
		"servers": []map[string]string{{"url": "/"}},
		"security": []map[string][]string{
			{"bearerAuth": {}},
			{"signalplaneToken": {}},
			{"sessionCookie": {}},
		},
		"paths": map[string]any{
			"/healthz":                             pathItem(getOp("Health", "Unauthenticated liveness check.", "", refResponse("200", "HealthResponse"))),
			"/api/auth/login":                      pathItem(postOp("Login", "Create a browser session.", "", "LoginRequest", refResponse("200", "LoginResponse"), emptyResponse("401"))),
			"/api/auth/logout":                     pathItem(postOp("Logout", "Revoke the current browser session.", "", "", emptyResponse("204"), emptyResponse("200"))),
			"/api/me":                              pathItem(getOp("Current user", "Return the authenticated user for the current session.", "read", refResponse("200", "MeResponse"), emptyResponse("401"))),
			"/api/bootstrap":                       pathItem(getOp("Bootstrap", "Return dashboard bootstrap data and dependency health.", "read", refResponse("200", "BootstrapResponse"), emptyResponse("401"))),
			"/api/services":                        pathItem(getOp("Services", "List inferred services.", "read", arrayResponse("200", "services", "Service"), emptyResponse("401"))),
			"/api/hosts":                           pathItem(getOp("Hosts", "List inferred hosts.", "read", arrayResponse("200", "hosts", "Host"), emptyResponse("401"))),
			"/api/metrics":                         pathItem(getOp("Metrics", "List recent metrics.", "read", arrayResponse("200", "metrics", "Metric"), emptyResponse("401"))),
			"/api/logs":                            pathItem(getOp("Logs", "List recent logs.", "read", arrayResponse("200", "logs", "Log"), emptyResponse("401"))),
			"/api/traces":                          pathItem(getOp("Traces", "List recent traces.", "read", arrayResponse("200", "traces", "Trace"), emptyResponse("401"))),
			"/api/alerts":                          pathItem(getOp("Alerts", "List alerts.", "read", arrayResponse("200", "alerts", "Alert"), emptyResponse("401"))),
			"/api/alerts/{id}":                     pathItem(patchOp("Update alert", "Acknowledge or resolve an alert.", "admin", "AlertStatusInput", refResponse("200", "AlertResponse"), emptyResponse("401"), emptyResponse("404"))),
			"/api/alert-rules":                     pathItem(getOp("Alert rules", "List alert rules.", "admin", arrayResponse("200", "alertRules", "AlertRule"), emptyResponse("401")), postOp("Create alert rule", "Create a metric, log, or trace alert rule.", "admin", "AlertRuleInput", refResponse("201", "AlertRuleResponse"), emptyResponse("401"))),
			"/api/incidents":                       pathItem(getOp("Incidents", "List incidents.", "read", arrayResponse("200", "incidents", "Incident"), emptyResponse("401")), postOp("Create incident", "Create a manual incident.", "admin", "IncidentInput", refResponse("201", "IncidentResponse"), emptyResponse("401"))),
			"/api/uptime-monitors":                 pathItem(getOp("Uptime monitors", "List uptime monitors.", "read", arrayResponse("200", "uptimeMonitors", "UptimeMonitor"), emptyResponse("401")), postOp("Create uptime monitor", "Register a synthetic uptime check.", "admin", "UptimeMonitorInput", refResponse("201", "UptimeMonitorResponse"), emptyResponse("401"))),
			"/api/uptime-monitors/{id}/check":      pathItem(postOp("Run uptime monitor", "Run a registered uptime monitor immediately.", "admin", "", refResponse("200", "UptimeMonitorResponse"), emptyResponse("401"), emptyResponse("404"))),
			"/api/notification-channels":           pathItem(getOp("Notification channels", "List notification channels.", "admin", arrayResponse("200", "notificationChannels", "NotificationChannel"), emptyResponse("401")), postOp("Create notification channel", "Create an email, webhook, or Slack-compatible notification channel.", "admin", "NotificationChannelInput", refResponse("201", "NotificationChannelResponse"), emptyResponse("401"))),
			"/api/notification-channels/{id}/test": pathItem(postOp("Test notification channel", "Send a test message through a configured notification channel.", "admin", "", refResponse("200", "NotificationTestResponse"), emptyResponse("401"), emptyResponse("404"))),
			"/api/system/dependencies":             pathItem(getOp("Dependencies", "Check PostgreSQL, ClickHouse, OTLP, SMTP, and Mailpit dependencies.", "read", arrayResponse("200", "dependencies", "DependencyStatus"), emptyResponse("401"))),
			"/api/tokens":                          pathItem(getOp("Tokens", "List API tokens with masked token values.", "admin", arrayResponse("200", "tokens", "Token"), emptyResponse("401")), postOp("Create token", "Create an ingest, read, or admin API token.", "admin", "TokenInput", refResponse("201", "TokenResponse"), emptyResponse("401"))),
			"/api/users":                           pathItem(getOp("Users", "List users.", "admin", arrayResponse("200", "users", "User"), emptyResponse("401")), postOp("Create user", "Create a user account.", "admin", "UserInput", refResponse("201", "UserResponse"), emptyResponse("401"))),
			"/api/ingest/hosts":                    pathItem(postOp("Ingest host", "Upsert a host from an agent or collector.", "ingest", "HostInput", refResponse("202", "HostResponse"), emptyResponse("401"))),
			"/api/ingest/metrics":                  pathItem(postArrayOp("Ingest metrics", "Ingest SignalPlane JSON metrics.", "ingest", "MetricInput", refResponse("202", "MetricIngestResponse"), emptyResponse("401"))),
			"/api/ingest/logs":                     pathItem(postArrayOp("Ingest logs", "Ingest SignalPlane JSON logs.", "ingest", "LogInput", refResponse("202", "LogIngestResponse"), emptyResponse("401"))),
			"/api/ingest/traces":                   pathItem(postArrayOp("Ingest traces", "Ingest SignalPlane JSON traces.", "ingest", "TraceInput", refResponse("202", "TraceIngestResponse"), emptyResponse("401"))),
			"/v1/metrics":                          pathItem(otlpOp("OTLP metrics", "Ingest OTLP HTTP JSON or protobuf metrics.")),
			"/v1/logs":                             pathItem(otlpOp("OTLP logs", "Ingest OTLP HTTP JSON or protobuf logs.")),
			"/v1/traces":                           pathItem(otlpOp("OTLP traces", "Ingest OTLP HTTP JSON or protobuf traces.")),
			"/api/openapi":                         pathItem(getOp("OpenAPI", "Return this OpenAPI 3.1 contract.", "read", schemaResponse("200", map[string]any{"type": "object"}), emptyResponse("401"))),
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"bearerAuth":       map[string]any{"type": "http", "scheme": "bearer"},
				"signalplaneToken": map[string]any{"type": "apiKey", "in": "header", "name": "X-SignalPlane-Token"},
				"sessionCookie":    map[string]any{"type": "apiKey", "in": "cookie", "name": "signalplane_session"},
			},
			"schemas": openAPISchemas(),
		},
	}
}

func openAPISchemas() map[string]any {
	return map[string]any{
		"Error":                       schemaObject(required("error", "message"), prop("error", stringSchema()), prop("message", stringSchema())),
		"HealthResponse":              schemaObject(nil, prop("status", stringSchema())),
		"LoginRequest":                schemaObject(required("email", "password"), prop("email", stringSchema()), prop("password", stringSchema())),
		"LoginResponse":               schemaObject(nil, prop("session", schemaObject(nil)), prop("user", ref("User"))),
		"MeResponse":                  schemaObject(nil, prop("user", ref("User"))),
		"BootstrapResponse":           schemaObject(nil, prop("health", stringSchema()), prop("counts", schemaObject(nil))),
		"Resource":                    schemaObject(nil, prop("service", stringSchema()), prop("host", stringSchema()), prop("environment", stringSchema()), prop("region", stringSchema()), prop("version", stringSchema()), prop("attributes", stringMap())),
		"Service":                     schemaObject(nil, prop("name", stringSchema()), prop("status", stringSchema()), prop("owner", stringSchema()), prop("environment", stringSchema()), prop("region", stringSchema()), prop("version", stringSchema()), prop("rate", numberSchema()), prop("p95", numberSchema()), prop("errorRate", numberSchema())),
		"Host":                        schemaObject(nil, prop("id", stringSchema()), prop("name", stringSchema()), prop("environment", stringSchema()), prop("region", stringSchema()), prop("agentVersion", stringSchema()), prop("status", stringSchema()), prop("lastSeenAt", dateTimeSchema()), prop("tags", arrayOf(stringSchema())), prop("metrics", numberMap()), prop("createdAt", dateTimeSchema()), prop("updatedAt", dateTimeSchema())),
		"HostInput":                   schemaObject(required("name"), prop("id", stringSchema()), prop("name", stringSchema()), prop("environment", stringSchema()), prop("region", stringSchema()), prop("agentVersion", stringSchema()), prop("status", stringSchema()), prop("metrics", numberMap()), prop("tags", arrayOf(stringSchema()))),
		"Metric":                      schemaObject(nil, prop("id", stringSchema()), prop("timestamp", dateTimeSchema()), prop("name", stringSchema()), prop("value", numberSchema()), prop("unit", stringSchema()), prop("type", stringSchema()), prop("labels", stringMap()), prop("resource", ref("Resource"))),
		"MetricInput":                 schemaObject(required("name", "value"), prop("timestamp", dateTimeSchema()), prop("name", stringSchema()), prop("value", numberSchema()), prop("unit", stringSchema()), prop("type", stringSchema()), prop("labels", stringMap()), prop("resource", ref("Resource"))),
		"Log":                         schemaObject(nil, prop("id", stringSchema()), prop("timestamp", dateTimeSchema()), prop("severity", stringSchema()), prop("message", stringSchema()), prop("traceId", stringSchema()), prop("spanId", stringSchema()), prop("fields", stringMap()), prop("resource", ref("Resource"))),
		"LogInput":                    schemaObject(required("message"), prop("timestamp", dateTimeSchema()), prop("severity", stringSchema()), prop("message", stringSchema()), prop("traceId", stringSchema()), prop("spanId", stringSchema()), prop("fields", stringMap()), prop("resource", ref("Resource"))),
		"Span":                        schemaObject(nil, prop("id", stringSchema()), prop("parentId", stringSchema()), prop("name", stringSchema()), prop("service", stringSchema()), prop("durationMs", numberSchema()), prop("status", stringSchema()), prop("attributes", stringMap())),
		"SpanInput":                   schemaObject(required("spanId", "name"), prop("spanId", stringSchema()), prop("parentId", stringSchema()), prop("name", stringSchema()), prop("durationMs", numberSchema()), prop("status", stringSchema()), prop("resource", ref("Resource")), prop("attributes", stringMap()), prop("timestamp", dateTimeSchema())),
		"Trace":                       schemaObject(nil, prop("id", stringSchema()), prop("traceId", stringSchema()), prop("timestamp", dateTimeSchema()), prop("rootService", stringSchema()), prop("operation", stringSchema()), prop("durationMs", numberSchema()), prop("status", stringSchema()), prop("spans", arrayRef("Span")), prop("resources", arrayRef("Resource"))),
		"TraceInput":                  schemaObject(required("traceId", "spans"), prop("traceId", stringSchema()), prop("spans", arrayRef("SpanInput"))),
		"Alert":                       schemaObject(nil, prop("id", stringSchema()), prop("title", stringSchema()), prop("severity", stringSchema()), prop("status", stringSchema()), prop("source", stringSchema()), prop("resource", ref("Resource"))),
		"AlertStatusInput":            schemaObject(required("status"), prop("status", stringSchema())),
		"AlertRule":                   schemaObject(nil, prop("id", stringSchema()), prop("name", stringSchema()), prop("signalType", stringSchema()), prop("enabled", boolSchema()), prop("metricName", stringSchema()), prop("logSeverity", stringSchema()), prop("query", stringSchema()), prop("operator", stringSchema()), prop("threshold", numberSchema()), prop("severity", stringSchema()), prop("labels", stringMap()), prop("createdAt", dateTimeSchema()), prop("updatedAt", dateTimeSchema())),
		"AlertRuleInput":              schemaObject(required("name", "signalType"), prop("name", stringSchema()), prop("signalType", stringSchema()), prop("enabled", boolSchema()), prop("metricName", stringSchema()), prop("logSeverity", stringSchema()), prop("query", stringSchema()), prop("operator", stringSchema()), prop("threshold", numberSchema()), prop("severity", stringSchema()), prop("labels", stringMap())),
		"Incident":                    schemaObject(nil, prop("id", stringSchema()), prop("timestamp", dateTimeSchema()), prop("title", stringSchema()), prop("severity", stringSchema()), prop("status", stringSchema()), prop("owner", stringSchema()), prop("affectedServices", arrayOf(stringSchema())), prop("affectedHosts", arrayOf(stringSchema())), prop("alertIds", arrayOf(stringSchema())), prop("notes", arrayOf(stringSchema())), prop("labels", stringMap())),
		"IncidentInput":               schemaObject(required("title"), prop("title", stringSchema()), prop("severity", stringSchema()), prop("owner", stringSchema()), prop("affectedServices", arrayOf(stringSchema())), prop("affectedHosts", arrayOf(stringSchema())), prop("alertIds", arrayOf(stringSchema())), prop("notes", arrayOf(stringSchema())), prop("labels", stringMap())),
		"UptimeMonitor":               schemaObject(nil, prop("id", stringSchema()), prop("name", stringSchema()), prop("url", stringSchema()), prop("method", stringSchema()), prop("expectedStatus", integerSchema()), prop("intervalSeconds", integerSchema()), prop("timeoutSeconds", integerSchema()), prop("status", stringSchema()), prop("lastCheckedAt", dateTimeSchema()), prop("lastStatusCode", integerSchema()), prop("lastResponseMs", numberSchema()), prop("lastError", stringSchema()), prop("consecutiveFailures", integerSchema()), prop("createdAt", dateTimeSchema())),
		"UptimeMonitorInput":          schemaObject(required("name", "url"), prop("name", stringSchema()), prop("url", stringSchema()), prop("method", stringSchema()), prop("expectedStatus", integerSchema()), prop("intervalSeconds", integerSchema()), prop("timeoutSeconds", integerSchema())),
		"NotificationChannel":         schemaObject(nil, prop("id", stringSchema()), prop("name", stringSchema()), prop("type", stringSchema()), prop("target", stringSchema()), prop("enabled", boolSchema()), prop("config", stringMap()), prop("createdAt", dateTimeSchema()), prop("updatedAt", dateTimeSchema())),
		"NotificationChannelInput":    schemaObject(required("name", "type"), prop("name", stringSchema()), prop("type", stringSchema()), prop("target", stringSchema()), prop("enabled", boolSchema()), prop("config", stringMap())),
		"DependencyStatus":            schemaObject(nil, prop("id", stringSchema()), prop("name", stringSchema()), prop("kind", stringSchema()), prop("target", stringSchema()), prop("status", stringSchema()), prop("latencyMs", numberSchema()), prop("lastChecked", dateTimeSchema())),
		"Token":                       schemaObject(nil, prop("id", stringSchema()), prop("name", stringSchema()), prop("token", stringSchema()), prop("scope", stringSchema()), prop("createdAt", dateTimeSchema()), prop("revokedAt", dateTimeSchema())),
		"TokenInput":                  schemaObject(required("name", "token", "scope"), prop("name", stringSchema()), prop("token", stringSchema()), prop("scope", stringSchema())),
		"User":                        schemaObject(nil, prop("id", stringSchema()), prop("email", stringSchema()), prop("name", stringSchema()), prop("role", stringSchema()), prop("createdAt", dateTimeSchema())),
		"UserInput":                   schemaObject(required("email", "password"), prop("email", stringSchema()), prop("password", stringSchema()), prop("name", stringSchema()), prop("role", stringSchema())),
		"AlertResponse":               wrapper("alert", "Alert"),
		"AlertRuleResponse":           wrapper("alertRule", "AlertRule"),
		"IncidentResponse":            wrapper("incident", "Incident"),
		"UptimeMonitorResponse":       wrapper("uptimeMonitor", "UptimeMonitor"),
		"NotificationChannelResponse": wrapper("notificationChannel", "NotificationChannel"),
		"NotificationTestResponse":    schemaObject(nil, prop("status", stringSchema())),
		"TokenResponse":               wrapper("token", "Token"),
		"UserResponse":                wrapper("user", "User"),
		"HostResponse":                wrapper("host", "Host"),
		"MetricIngestResponse":        ingestResponse("metrics", "Metric"),
		"LogIngestResponse":           ingestResponse("logs", "Log"),
		"TraceIngestResponse":         ingestResponse("traces", "Trace"),
		"OTLPAcceptedResponse":        schemaObject(nil, prop("accepted", integerSchema()), prop("signal", stringSchema())),
	}
}

func pathItem(operations ...map[string]any) map[string]any {
	item := map[string]any{}
	for _, operation := range operations {
		item[operation["method"].(string)] = operation["operation"]
	}
	return item
}

func getOp(summary, description, scope string, responses ...map[string]any) map[string]any {
	return op("get", summary, description, scope, "", responses...)
}

func postOp(summary, description, scope, requestSchema string, responses ...map[string]any) map[string]any {
	return op("post", summary, description, scope, requestSchema, responses...)
}

func postArrayOp(summary, description, scope, itemSchema string, responses ...map[string]any) map[string]any {
	operation := op("post", summary, description, scope, "", responses...)
	operation["operation"].(map[string]any)["requestBody"] = requestBody(arrayRef(itemSchema))
	return operation
}

func patchOp(summary, description, scope, requestSchema string, responses ...map[string]any) map[string]any {
	return op("patch", summary, description, scope, requestSchema, responses...)
}

func otlpOp(summary, description string) map[string]any {
	operation := op("post", summary, description, "ingest", "", otlpProtobufResponse(), refResponse("202", "OTLPAcceptedResponse"), emptyResponse("400"), emptyResponse("401"), emptyResponse("415"))
	operation["operation"].(map[string]any)["requestBody"] = map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json":       map[string]any{"schema": map[string]any{"type": "object"}},
			"application/x-protobuf": map[string]any{"schema": map[string]any{"type": "string", "contentEncoding": "base64"}},
		},
	}
	return operation
}

func otlpProtobufResponse() map[string]any {
	return map[string]any{
		"200": map[string]any{
			"description": "OTLP protobuf accepted. SignalPlane returns HTTP 200 with an OTLP protobuf response envelope for protobuf requests.",
			"content": map[string]any{
				"application/x-protobuf": map[string]any{
					"schema": map[string]any{"type": "string", "contentEncoding": "base64"},
				},
			},
		},
	}
}

func op(method, summary, description, scope, requestSchema string, responses ...map[string]any) map[string]any {
	operation := map[string]any{
		"summary":     summary,
		"description": description,
		"responses":   mergeResponses(responses...),
	}
	if scope != "" {
		operation["security"] = securityForScope(scope)
	}
	if requestSchema != "" {
		operation["requestBody"] = requestBody(ref(requestSchema))
	}
	return map[string]any{"method": method, "operation": operation}
}

func securityForScope(scope string) []map[string][]string {
	return []map[string][]string{
		{"bearerAuth": {scope}},
		{"signalplaneToken": {scope}},
		{"sessionCookie": {scope}},
	}
}

func mergeResponses(responses ...map[string]any) map[string]any {
	merged := map[string]any{}
	for _, response := range responses {
		for status, body := range response {
			merged[status] = body
		}
	}
	if _, ok := merged["default"]; !ok {
		merged["default"] = map[string]any{"description": "Error", "content": content(ref("Error"))}
	}
	return merged
}

func requestBody(schema map[string]any) map[string]any {
	return map[string]any{
		"required": true,
		"content":  content(schema),
	}
}

func refResponse(status, schemaName string) map[string]any {
	return schemaResponse(status, ref(schemaName))
}

func arrayResponse(status, field, schemaName string) map[string]any {
	return schemaResponse(status, schemaObject(nil, prop(field, arrayRef(schemaName))))
}

func schemaResponse(status string, schema map[string]any) map[string]any {
	return map[string]any{status: map[string]any{"description": "OK", "content": content(schema)}}
}

func emptyResponse(status string) map[string]any {
	return map[string]any{status: map[string]any{"description": httpStatusDescription(status)}}
}

func httpStatusDescription(status string) string {
	switch status {
	case "200":
		return "OK"
	case "201":
		return "Created"
	case "202":
		return "Accepted"
	case "204":
		return "No Content"
	case "400":
		return "Bad Request"
	case "401":
		return "Unauthorized"
	case "404":
		return "Not Found"
	case "415":
		return "Unsupported Media Type"
	default:
		return "Response"
	}
}

func content(schema map[string]any) map[string]any {
	return map[string]any{"application/json": map[string]any{"schema": schema}}
}

func ingestResponse(field, schemaName string) map[string]any {
	return schemaObject(nil, prop("accepted", integerSchema()), prop(field, arrayRef(schemaName)))
}

func wrapper(field, schemaName string) map[string]any {
	return schemaObject(nil, prop(field, ref(schemaName)))
}

func schemaObject(requiredFields []string, properties ...map[string]any) map[string]any {
	schema := map[string]any{"type": "object"}
	props := map[string]any{}
	for _, property := range properties {
		for name, value := range property {
			props[name] = value
		}
	}
	if len(props) > 0 {
		schema["properties"] = props
	}
	if len(requiredFields) > 0 {
		schema["required"] = requiredFields
	}
	return schema
}

func prop(name string, schema map[string]any) map[string]any {
	return map[string]any{name: schema}
}

func ref(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

func arrayRef(name string) map[string]any {
	return arrayOf(ref(name))
}

func arrayOf(schema map[string]any) map[string]any {
	return map[string]any{"type": "array", "items": schema}
}

func stringMap() map[string]any {
	return map[string]any{"type": "object", "additionalProperties": stringSchema()}
}

func numberMap() map[string]any {
	return map[string]any{"type": "object", "additionalProperties": numberSchema()}
}

func required(fields ...string) []string {
	return fields
}

func stringSchema() map[string]any {
	return map[string]any{"type": "string"}
}

func dateTimeSchema() map[string]any {
	return map[string]any{"type": "string", "format": "date-time"}
}

func numberSchema() map[string]any {
	return map[string]any{"type": "number"}
}

func integerSchema() map[string]any {
	return map[string]any{"type": "integer"}
}

func boolSchema() map[string]any {
	return map[string]any{"type": "boolean"}
}
