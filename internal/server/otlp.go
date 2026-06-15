package server

import (
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chaktihor/signalplane/internal/store"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const maxOTLPProtobufBodyBytes = 16 << 20

func (s *Server) ingestOTLPLogs(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r) {
		return
	}
	if isOTLPProtobuf(r) {
		var payload logspb.LogsData
		if !decodeOTLPProtobuf(w, r, &payload) {
			return
		}
		s.store.IngestLogs(parseOTLPLogsProto(&payload))
		writeOTLPSuccess(w)
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
	if isOTLPProtobuf(r) {
		var payload metricspb.MetricsData
		if !decodeOTLPProtobuf(w, r, &payload) {
			return
		}
		s.store.IngestMetrics(parseOTLPMetricsProto(&payload))
		writeOTLPSuccess(w)
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
	if isOTLPProtobuf(r) {
		var payload tracepb.TracesData
		if !decodeOTLPProtobuf(w, r, &payload) {
			return
		}
		s.store.IngestTraces(parseOTLPTracesProto(&payload))
		writeOTLPSuccess(w)
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
				fields := otlpAttributes(record["attributes"])
				out = append(out, store.LogInput{
					Timestamp: otlpUnixNano(record["timeUnixNano"]),
					Severity:  firstNonEmptyString(textField(record["severityText"]), strings.ToLower(textField(record["severityNumber"]))),
					Message:   otlpValue(record["body"]),
					TraceID:   firstNonEmptyString(hexField(record["traceId"]), fields["traceId"], fields["trace_id"]),
					SpanID:    firstNonEmptyString(hexField(record["spanId"]), fields["spanId"], fields["span_id"]),
					Fields:    fields,
					Resource:  resource,
				})
			}
		}
	}
	return out
}

func parseOTLPLogsProto(payload *logspb.LogsData) []store.LogInput {
	var out []store.LogInput
	for _, resourceLog := range payload.GetResourceLogs() {
		resource := otlpResourceProto(resourceLog.GetResource().GetAttributes())
		for _, scopeLog := range resourceLog.GetScopeLogs() {
			for _, record := range scopeLog.GetLogRecords() {
				fields := otlpAttributesProto(record.GetAttributes())
				out = append(out, store.LogInput{
					Timestamp: otlpUnixNanoUint(firstNonZeroUint64(record.GetTimeUnixNano(), record.GetObservedTimeUnixNano())),
					Severity:  otlpLogSeverity(record),
					Message:   otlpValueProto(record.GetBody()),
					TraceID:   firstNonEmptyString(bytesHex(record.GetTraceId()), fields["traceId"], fields["trace_id"]),
					SpanID:    firstNonEmptyString(bytesHex(record.GetSpanId()), fields["spanId"], fields["span_id"]),
					Fields:    fields,
					Resource:  resource,
				})
			}
		}
	}
	return out
}

func parseOTLPMetricsProto(payload *metricspb.MetricsData) []store.MetricInput {
	var out []store.MetricInput
	for _, resourceMetric := range payload.GetResourceMetrics() {
		resource := otlpResourceProto(resourceMetric.GetResource().GetAttributes())
		for _, scopeMetric := range resourceMetric.GetScopeMetrics() {
			for _, metric := range scopeMetric.GetMetrics() {
				name := metric.GetName()
				unit := metric.GetUnit()
				switch data := metric.GetData().(type) {
				case *metricspb.Metric_Gauge:
					for _, point := range data.Gauge.GetDataPoints() {
						out = append(out, store.MetricInput{
							Timestamp: otlpUnixNanoUint(firstNonZeroUint64(point.GetTimeUnixNano(), point.GetStartTimeUnixNano())),
							Name:      name,
							Value:     otlpNumberDataPointValue(point),
							Unit:      unit,
							Type:      "gauge",
							Labels:    otlpAttributesProto(point.GetAttributes()),
							Resource:  resource,
						})
					}
				case *metricspb.Metric_Sum:
					for _, point := range data.Sum.GetDataPoints() {
						out = append(out, store.MetricInput{
							Timestamp: otlpUnixNanoUint(firstNonZeroUint64(point.GetTimeUnixNano(), point.GetStartTimeUnixNano())),
							Name:      name,
							Value:     otlpNumberDataPointValue(point),
							Unit:      unit,
							Type:      "counter",
							Labels:    otlpAttributesProto(point.GetAttributes()),
							Resource:  resource,
						})
					}
				case *metricspb.Metric_Histogram:
					for _, point := range data.Histogram.GetDataPoints() {
						out = append(out, store.MetricInput{
							Timestamp: otlpUnixNanoUint(firstNonZeroUint64(point.GetTimeUnixNano(), point.GetStartTimeUnixNano())),
							Name:      name,
							Value:     otlpHistogramPointValue(point),
							Unit:      unit,
							Type:      "histogram",
							Labels:    otlpAttributesProto(point.GetAttributes()),
							Resource:  resource,
						})
					}
				case *metricspb.Metric_Summary:
					for _, point := range data.Summary.GetDataPoints() {
						out = append(out, store.MetricInput{
							Timestamp: otlpUnixNanoUint(firstNonZeroUint64(point.GetTimeUnixNano(), point.GetStartTimeUnixNano())),
							Name:      name,
							Value:     otlpSummaryPointValue(point),
							Unit:      unit,
							Type:      "summary",
							Labels:    otlpAttributesProto(point.GetAttributes()),
							Resource:  resource,
						})
					}
				}
			}
		}
	}
	return out
}

func parseOTLPTracesProto(payload *tracepb.TracesData) []store.TraceInput {
	grouped := map[string][]store.SpanInput{}
	for _, resourceSpan := range payload.GetResourceSpans() {
		resource := otlpResourceProto(resourceSpan.GetResource().GetAttributes())
		for _, scopeSpan := range resourceSpan.GetScopeSpans() {
			for _, span := range scopeSpan.GetSpans() {
				traceID := bytesHex(span.GetTraceId())
				if traceID == "" {
					traceID = "trace-" + newTextID()
				}
				grouped[traceID] = append(grouped[traceID], store.SpanInput{
					SpanID:     firstNonEmptyString(bytesHex(span.GetSpanId()), "span-"+newTextID()),
					ParentID:   bytesHex(span.GetParentSpanId()),
					Name:       span.GetName(),
					DurationMS: spanDurationMSProto(span),
					Status:     otlpSpanStatusProto(span.GetStatus()),
					Resource:   resource,
					Attributes: otlpAttributesProto(span.GetAttributes()),
					Timestamp:  otlpUnixNanoUint(span.GetStartTimeUnixNano()),
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

func otlpResourceProto(attributes []*commonpb.KeyValue) store.Resource {
	values := otlpAttributesProto(attributes)
	return store.Resource{
		Service:     firstNonEmptyString(values["service.name"], values["service"]),
		Host:        firstNonEmptyString(values["host.name"], values["host"], values["container.name"], values["k8s.pod.name"]),
		Environment: firstNonEmptyString(values["deployment.environment"], values["deployment.environment.name"], values["environment"]),
		Region:      firstNonEmptyString(values["cloud.region"], values["region"]),
		Version:     values["service.version"],
		Attributes:  values,
	}
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

func otlpAttributesProto(attributes []*commonpb.KeyValue) map[string]string {
	out := map[string]string{}
	for _, attribute := range attributes {
		key := strings.TrimSpace(attribute.GetKey())
		if key == "" {
			continue
		}
		out[key] = otlpValueProto(attribute.GetValue())
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

func otlpValueProto(value *commonpb.AnyValue) string {
	if value == nil {
		return ""
	}
	switch typed := value.GetValue().(type) {
	case *commonpb.AnyValue_StringValue:
		return typed.StringValue
	case *commonpb.AnyValue_IntValue:
		return strconv.FormatInt(typed.IntValue, 10)
	case *commonpb.AnyValue_DoubleValue:
		return strconv.FormatFloat(typed.DoubleValue, 'f', -1, 64)
	case *commonpb.AnyValue_BoolValue:
		return strconv.FormatBool(typed.BoolValue)
	case *commonpb.AnyValue_BytesValue:
		return bytesHex(typed.BytesValue)
	case *commonpb.AnyValue_ArrayValue, *commonpb.AnyValue_KvlistValue:
		raw, err := protojson.Marshal(value)
		if err == nil {
			return string(raw)
		}
	}
	return value.String()
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

func otlpNumberDataPointValue(point *metricspb.NumberDataPoint) float64 {
	switch typed := point.GetValue().(type) {
	case *metricspb.NumberDataPoint_AsDouble:
		return typed.AsDouble
	case *metricspb.NumberDataPoint_AsInt:
		return float64(typed.AsInt)
	default:
		return 0
	}
}

func otlpHistogramPointValue(point *metricspb.HistogramDataPoint) float64 {
	if point.GetCount() > 0 {
		return point.GetSum() / float64(point.GetCount())
	}
	return point.GetSum()
}

func otlpSummaryPointValue(point *metricspb.SummaryDataPoint) float64 {
	if point.GetCount() > 0 {
		return point.GetSum() / float64(point.GetCount())
	}
	return point.GetSum()
}

func spanDurationMS(span map[string]any) float64 {
	start := otlpUnixNanoTime(span["startTimeUnixNano"])
	end := otlpUnixNanoTime(span["endTimeUnixNano"])
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return float64(end.Sub(start).Microseconds()) / 1000
}

func spanDurationMSProto(span *tracepb.Span) float64 {
	start := time.Unix(0, int64(span.GetStartTimeUnixNano())).UTC()
	end := time.Unix(0, int64(span.GetEndTimeUnixNano())).UTC()
	if span.GetStartTimeUnixNano() == 0 || span.GetEndTimeUnixNano() == 0 || end.Before(start) {
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

func otlpSpanStatusProto(status *tracepb.Status) string {
	if status.GetCode() == tracepb.Status_STATUS_CODE_ERROR || strings.Contains(strings.ToLower(status.GetMessage()), "error") {
		return "error"
	}
	return "ok"
}

func otlpLogSeverity(record *logspb.LogRecord) string {
	if severity := strings.ToLower(strings.TrimSpace(record.GetSeverityText())); severity != "" {
		return severity
	}
	switch number := int32(record.GetSeverityNumber()); {
	case number >= 21:
		return "fatal"
	case number >= 17:
		return "error"
	case number >= 13:
		return "warning"
	case number >= 9:
		return "info"
	case number >= 5:
		return "debug"
	case number >= 1:
		return "trace"
	default:
		return "info"
	}
}

func otlpUnixNano(value any) string {
	parsed := otlpUnixNanoTime(value)
	if parsed.IsZero() {
		return ""
	}
	return parsed.UTC().Format(time.RFC3339Nano)
}

func otlpUnixNanoUint(value uint64) string {
	if value == 0 {
		return ""
	}
	return time.Unix(0, int64(value)).UTC().Format(time.RFC3339Nano)
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

func firstNonZeroUint64(values ...uint64) uint64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func newTextID() string {
	return strconv.FormatInt(time.Now().UTC().UnixNano(), 16)
}

func isOTLPProtobuf(r *http.Request) bool {
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	return strings.Contains(contentType, "application/x-protobuf") || strings.Contains(contentType, "application/protobuf")
}

func decodeOTLPProtobuf(w http.ResponseWriter, r *http.Request, target proto.Message) bool {
	defer r.Body.Close()
	var reader io.Reader = r.Body
	switch encoding := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Encoding"))); encoding {
	case "", "identity":
	case "gzip":
		gzipReader, err := gzip.NewReader(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_protobuf", "gzip request body is invalid: "+err.Error())
			return false
		}
		defer gzipReader.Close()
		reader = gzipReader
	default:
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_encoding", "unsupported OTLP content encoding: "+encoding)
		return false
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, io.NopCloser(reader), maxOTLPProtobufBodyBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_protobuf", "request body must be valid OTLP protobuf: "+err.Error())
		return false
	}
	if err := proto.Unmarshal(body, target); err != nil {
		writeError(w, http.StatusBadRequest, "bad_protobuf", "request body must be valid OTLP protobuf: "+err.Error())
		return false
	}
	return true
}

func writeOTLPSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
}

func bytesHex(value []byte) string {
	if len(value) == 0 {
		return ""
	}
	return hex.EncodeToString(value)
}
