package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chaktihor/signalplane/internal/store"
)

func (s *Server) ingestOTLPLogs(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r) {
		return
	}
	var payload map[string]any
	if !decodeJSON(w, r, &payload) {
		return
	}
	logs := parseOTLPLogs(payload)
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": len(logs), "logs": s.store.IngestLogs(logs)})
}

func (s *Server) ingestOTLPMetrics(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r) {
		return
	}
	var payload map[string]any
	if !decodeJSON(w, r, &payload) {
		return
	}
	metrics := parseOTLPMetrics(payload)
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": len(metrics), "metrics": s.store.IngestMetrics(metrics)})
}

func (s *Server) ingestOTLPTraces(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r) {
		return
	}
	var payload map[string]any
	if !decodeJSON(w, r, &payload) {
		return
	}
	traces := parseOTLPTraces(payload)
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": len(traces), "traces": s.store.IngestTraces(traces)})
}

func parseOTLPLogs(payload map[string]any) []store.LogInput {
	var out []store.LogInput
	for _, resourceLog := range objectList(payload["resourceLogs"]) {
		resource := otlpResource(object(resourceLog["resource"]))
		for _, scopeLog := range objectList(resourceLog["scopeLogs"]) {
			for _, record := range objectList(scopeLog["logRecords"]) {
				out = append(out, store.LogInput{
					Timestamp: otlpUnixNano(record["timeUnixNano"]),
					Severity:  firstNonEmptyString(textField(record["severityText"]), strings.ToLower(textField(record["severityNumber"]))),
					Message:   otlpValue(record["body"]),
					TraceID:   hexField(record["traceId"]),
					SpanID:    hexField(record["spanId"]),
					Fields:    otlpAttributes(record["attributes"]),
					Resource:  resource,
				})
			}
		}
	}
	return out
}

func parseOTLPMetrics(payload map[string]any) []store.MetricInput {
	var out []store.MetricInput
	for _, resourceMetric := range objectList(payload["resourceMetrics"]) {
		resource := otlpResource(object(resourceMetric["resource"]))
		for _, scopeMetric := range objectList(resourceMetric["scopeMetrics"]) {
			for _, metric := range objectList(scopeMetric["metrics"]) {
				name := textField(metric["name"])
				unit := textField(metric["unit"])
				for _, point := range otlpDataPoints(metric) {
					out = append(out, store.MetricInput{
						Timestamp: otlpUnixNano(firstNonNil(point["timeUnixNano"], point["startTimeUnixNano"])),
						Name:      name,
						Value:     otlpNumber(firstNonNil(point["asDouble"], point["asInt"], point["value"])),
						Unit:      unit,
						Type:      otlpMetricType(metric),
						Labels:    otlpAttributes(point["attributes"]),
						Resource:  resource,
					})
				}
			}
		}
	}
	return out
}

func parseOTLPTraces(payload map[string]any) []store.TraceInput {
	grouped := map[string][]store.SpanInput{}
	for _, resourceSpan := range objectList(payload["resourceSpans"]) {
		resource := otlpResource(object(resourceSpan["resource"]))
		for _, scopeSpan := range objectList(resourceSpan["scopeSpans"]) {
			for _, span := range objectList(scopeSpan["spans"]) {
				traceID := hexField(span["traceId"])
				if traceID == "" {
					traceID = "trace-" + newTextID()
				}
				grouped[traceID] = append(grouped[traceID], store.SpanInput{
					SpanID:     firstNonEmptyString(hexField(span["spanId"]), "span-"+newTextID()),
					ParentID:   hexField(span["parentSpanId"]),
					Name:       textField(span["name"]),
					DurationMS: spanDurationMS(span),
					Status:     otlpSpanStatus(object(span["status"])),
					Resource:   resource,
					Attributes: otlpAttributes(span["attributes"]),
					Timestamp:  otlpUnixNano(span["startTimeUnixNano"]),
				})
			}
		}
	}
	out := make([]store.TraceInput, 0, len(grouped))
	for traceID, spans := range grouped {
		out = append(out, store.TraceInput{TraceID: traceID, Spans: spans})
	}
	return out
}

func otlpResource(resource map[string]any) store.Resource {
	attributes := otlpAttributes(resource["attributes"])
	out := store.Resource{
		Service:     firstNonEmptyString(attributes["service.name"], attributes["service"]),
		Host:        firstNonEmptyString(attributes["host.name"], attributes["host"], attributes["container.name"], attributes["k8s.pod.name"]),
		Environment: firstNonEmptyString(attributes["deployment.environment"], attributes["deployment.environment.name"], attributes["environment"]),
		Region:      firstNonEmptyString(attributes["cloud.region"], attributes["region"]),
		Version:     attributes["service.version"],
		Attributes:  attributes,
	}
	return out
}

func otlpDataPoints(metric map[string]any) []map[string]any {
	for _, key := range []string{"gauge", "sum", "histogram"} {
		container := object(metric[key])
		if points := objectList(container["dataPoints"]); len(points) > 0 {
			return points
		}
	}
	return nil
}

func otlpMetricType(metric map[string]any) string {
	for _, key := range []string{"gauge", "sum", "histogram"} {
		if _, ok := metric[key]; ok {
			if key == "sum" {
				return "counter"
			}
			return key
		}
	}
	return "gauge"
}

func otlpAttributes(value any) map[string]string {
	out := map[string]string{}
	for _, attribute := range objectList(value) {
		key := textField(attribute["key"])
		if key == "" {
			continue
		}
		out[key] = otlpValue(attribute["value"])
	}
	return out
}

func otlpValue(value any) string {
	obj := object(value)
	for _, key := range []string{"stringValue", "intValue", "doubleValue", "boolValue", "bytesValue"} {
		if raw, ok := obj[key]; ok {
			return textField(raw)
		}
	}
	if value == nil {
		return ""
	}
	return textField(value)
}

func otlpNumber(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case string:
		parsed, _ := strconv.ParseFloat(typed, 64)
		return parsed
	default:
		parsed, _ := strconv.ParseFloat(fmt.Sprint(value), 64)
		return parsed
	}
}

func spanDurationMS(span map[string]any) float64 {
	start := otlpUnixNanoTime(span["startTimeUnixNano"])
	end := otlpUnixNanoTime(span["endTimeUnixNano"])
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return float64(end.Sub(start).Microseconds()) / 1000
}

func otlpSpanStatus(status map[string]any) string {
	code := strings.ToLower(textField(status["code"]))
	if code == "2" || code == "error" || strings.Contains(strings.ToLower(textField(status["message"])), "error") {
		return "error"
	}
	return "ok"
}

func otlpUnixNano(value any) string {
	parsed := otlpUnixNanoTime(value)
	if parsed.IsZero() {
		return ""
	}
	return parsed.UTC().Format(time.RFC3339Nano)
}

func otlpUnixNanoTime(value any) time.Time {
	textValue := textField(value)
	if textValue == "" {
		return time.Time{}
	}
	nanos, err := strconv.ParseInt(textValue, 10, 64)
	if err != nil || nanos <= 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos).UTC()
}

func object(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func objectList(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, object(item))
	}
	return out
}

func textField(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

func hexField(value any) string {
	textValue := textField(value)
	if textValue == "" {
		return ""
	}
	if _, err := hex.DecodeString(textValue); err == nil {
		return strings.ToLower(textValue)
	}
	return textValue
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func newTextID() string {
	return strconv.FormatInt(time.Now().UTC().UnixNano(), 16)
}
