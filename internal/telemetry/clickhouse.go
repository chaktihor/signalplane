package telemetry

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/chaktihor/signalplane/internal/store"
)

type ClickHouseSink struct {
	baseURL      string
	database     string
	organization string
	username     string
	password     string
	client       *http.Client
}

type ClickHouseOptions struct {
	URL          string
	Database     string
	Organization string
	Username     string
	Password     string
	Timeout      time.Duration
}

func NewClickHouseSink(options ClickHouseOptions) (*ClickHouseSink, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(options.URL), "/")
	baseURL = strings.TrimSuffix(baseURL, "/ping")
	if baseURL == "" {
		return nil, errors.New("clickhouse url is required")
	}
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil {
		return nil, err
	}
	username := options.Username
	password := options.Password
	if parsed.User != nil {
		username = parsed.User.Username()
		if value, ok := parsed.User.Password(); ok {
			password = value
		}
		parsed.User = nil
		baseURL = strings.TrimRight(parsed.String(), "/")
	}
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	database := options.Database
	if database == "" {
		database = "signalplane"
	}
	organization := options.Organization
	if organization == "" {
		organization = "org-default"
	}
	return &ClickHouseSink{
		baseURL:      baseURL,
		database:     database,
		organization: organization,
		username:     username,
		password:     password,
		client:       &http.Client{Timeout: timeout},
	}, nil
}

func (sink *ClickHouseSink) WriteMetrics(metrics []store.Metric) error {
	rows := make([]metricRow, 0, len(metrics))
	for _, metric := range metrics {
		rows = append(rows, metricRow{
			Timestamp:          timestamp(metric.Timestamp),
			OrganizationID:     sink.organization,
			Environment:        metric.Resource.Environment,
			Service:            metric.Resource.Service,
			Host:               metric.Resource.Host,
			Region:             metric.Resource.Region,
			MetricName:         metric.Name,
			MetricType:         metric.Type,
			Unit:               metric.Unit,
			Value:              metric.Value,
			Labels:             safeMap(metric.Labels),
			ResourceAttributes: safeMap(metric.Resource.Attributes),
		})
	}
	return sink.insert("metrics", rows)
}

func (sink *ClickHouseSink) WriteLogs(logs []store.Log) error {
	rows := make([]logRow, 0, len(logs))
	for _, log := range logs {
		rows = append(rows, logRow{
			Timestamp:          timestamp(log.Timestamp),
			OrganizationID:     sink.organization,
			Environment:        log.Resource.Environment,
			Service:            log.Resource.Service,
			Host:               log.Resource.Host,
			Region:             log.Resource.Region,
			Severity:           log.Severity,
			Message:            log.Message,
			TraceID:            log.TraceID,
			SpanID:             log.SpanID,
			Fields:             safeMap(log.Fields),
			ResourceAttributes: safeMap(log.Resource.Attributes),
		})
	}
	return sink.insert("logs", rows)
}

func (sink *ClickHouseSink) WriteTraces(traces []store.Trace) error {
	traceRows := make([]traceRow, 0, len(traces))
	spanRows := make([]spanRow, 0)
	for _, trace := range traces {
		attributes := map[string]string{}
		for _, resource := range trace.Resources {
			for key, value := range resource.Attributes {
				attributes[key] = value
			}
		}
		traceRows = append(traceRows, traceRow{
			Timestamp:          timestamp(trace.Timestamp),
			OrganizationID:     sink.organization,
			Environment:        environment(trace.Resources),
			TraceID:            trace.TraceID,
			RootService:        trace.RootService,
			Operation:          trace.Operation,
			DurationMS:         trace.DurationMS,
			Status:             trace.Status,
			ResourceAttributes: attributes,
		})
		for _, span := range trace.Spans {
			spanRows = append(spanRows, spanRow{
				Timestamp:      timestamp(trace.Timestamp),
				OrganizationID: sink.organization,
				Environment:    environment(trace.Resources),
				TraceID:        trace.TraceID,
				SpanID:         span.ID,
				ParentSpanID:   span.ParentID,
				Service:        span.Service,
				Name:           span.Name,
				DurationMS:     span.DurationMS,
				Status:         span.Status,
				Attributes:     safeMap(span.Attributes),
			})
		}
	}
	if err := sink.insert("traces", traceRows); err != nil {
		return err
	}
	return sink.insert("spans", spanRows)
}

func (sink *ClickHouseSink) WriteUptimeResult(monitor store.UptimeMonitor) error {
	rows := []uptimeRow{{
		Timestamp:      timestamp(monitor.LastCheckedAt),
		OrganizationID: sink.organization,
		MonitorID:      monitor.ID,
		Name:           monitor.Name,
		URL:            monitor.URL,
		Status:         monitor.Status,
		ExpectedStatus: monitor.ExpectedStatus,
		StatusCode:     monitor.LastStatusCode,
		ResponseMS:     monitor.LastResponseMS,
		Error:          monitor.LastError,
	}}
	return sink.insert("uptime_results", rows)
}

func (sink *ClickHouseSink) Metrics(ctx context.Context, limit int) ([]store.Metric, error) {
	rows, err := sink.query(ctx, fmt.Sprintf(`
SELECT
  timestamp,
  metric_name,
  value,
  unit,
  metric_type,
  labels,
  environment,
  service,
  host,
  region,
  resource_attributes
FROM %s.metrics
ORDER BY timestamp DESC
LIMIT %d
FORMAT JSONEachRow`, sink.database, queryLimit(limit)))
	if err != nil {
		return nil, err
	}
	out := make([]store.Metric, 0, len(rows))
	for _, raw := range rows {
		var row metricQueryRow
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, err
		}
		out = append(out, store.Metric{
			ID:        "ch-metric-" + row.Timestamp + "-" + row.MetricName,
			Timestamp: apiTime(row.Timestamp),
			Name:      row.MetricName,
			Value:     row.Value,
			Unit:      row.Unit,
			Type:      row.MetricType,
			Labels:    safeMap(row.Labels),
			Resource: store.Resource{
				Service:     row.Service,
				Host:        row.Host,
				Environment: row.Environment,
				Region:      row.Region,
				Attributes:  safeMap(row.ResourceAttributes),
			},
		})
	}
	return out, nil
}

func (sink *ClickHouseSink) Logs(ctx context.Context, limit int, service, severity, search string) ([]store.Log, error) {
	filters := []string{}
	if service != "" {
		filters = append(filters, "service = "+sqlString(service))
	}
	if severity != "" {
		filters = append(filters, "severity = "+sqlString(severity))
	}
	if search != "" {
		filters = append(filters, "positionCaseInsensitive(message, "+sqlString(search)+") > 0")
	}
	where := whereClause(filters)
	rows, err := sink.query(ctx, fmt.Sprintf(`
SELECT
  timestamp,
  severity,
  message,
  trace_id,
  span_id,
  fields,
  environment,
  service,
  host,
  region,
  resource_attributes
FROM %s.logs
%s
ORDER BY timestamp DESC
LIMIT %d
FORMAT JSONEachRow`, sink.database, where, queryLimit(limit)))
	if err != nil {
		return nil, err
	}
	out := make([]store.Log, 0, len(rows))
	for _, raw := range rows {
		var row logQueryRow
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, err
		}
		out = append(out, store.Log{
			ID:        "ch-log-" + row.Timestamp + "-" + row.TraceID + "-" + row.SpanID,
			Timestamp: apiTime(row.Timestamp),
			Severity:  row.Severity,
			Message:   row.Message,
			TraceID:   row.TraceID,
			SpanID:    row.SpanID,
			Fields:    safeMap(row.Fields),
			Resource: store.Resource{
				Service:     row.Service,
				Host:        row.Host,
				Environment: row.Environment,
				Region:      row.Region,
				Attributes:  safeMap(row.ResourceAttributes),
			},
		})
	}
	return out, nil
}

func (sink *ClickHouseSink) Traces(ctx context.Context, limit int, service, status string) ([]store.Trace, error) {
	filters := []string{}
	if service != "" {
		filters = append(filters, "root_service = "+sqlString(service))
	}
	if status != "" {
		filters = append(filters, "status = "+sqlString(status))
	}
	rows, err := sink.query(ctx, fmt.Sprintf(`
SELECT
  timestamp,
  trace_id,
  root_service,
  operation,
  duration_ms,
  status,
  environment,
  resource_attributes
FROM %s.traces
%s
ORDER BY timestamp DESC
LIMIT %d
FORMAT JSONEachRow`, sink.database, whereClause(filters), queryLimit(limit)))
	if err != nil {
		return nil, err
	}
	out := make([]store.Trace, 0, len(rows))
	traceIDs := make([]string, 0, len(rows))
	seen := map[string]bool{}
	for _, raw := range rows {
		var row traceQueryRow
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, err
		}
		trace := store.Trace{
			ID:          "ch-trace-" + row.TraceID,
			TraceID:     row.TraceID,
			Timestamp:   apiTime(row.Timestamp),
			RootService: row.RootService,
			Operation:   row.Operation,
			DurationMS:  row.DurationMS,
			Status:      row.Status,
			Spans:       []store.Span{},
			Resources: []store.Resource{{
				Service:     row.RootService,
				Environment: row.Environment,
				Attributes:  safeMap(row.ResourceAttributes),
			}},
		}
		out = append(out, trace)
		if row.TraceID != "" && !seen[row.TraceID] {
			traceIDs = append(traceIDs, row.TraceID)
			seen[row.TraceID] = true
		}
	}
	if len(traceIDs) == 0 {
		return out, nil
	}
	spans, err := sink.spans(ctx, traceIDs)
	if err != nil {
		return out, nil
	}
	for i := range out {
		out[i].Spans = spans[out[i].TraceID]
	}
	return out, nil
}

func (sink *ClickHouseSink) spans(ctx context.Context, traceIDs []string) (map[string][]store.Span, error) {
	sort.Strings(traceIDs)
	values := make([]string, 0, len(traceIDs))
	for _, id := range traceIDs {
		values = append(values, sqlString(id))
	}
	rows, err := sink.query(ctx, fmt.Sprintf(`
SELECT
  trace_id,
  span_id,
  parent_span_id,
  service,
  name,
  duration_ms,
  status,
  attributes
FROM %s.spans
WHERE trace_id IN (%s)
ORDER BY timestamp ASC, span_id ASC
FORMAT JSONEachRow`, sink.database, strings.Join(values, ",")))
	if err != nil {
		return nil, err
	}
	out := map[string][]store.Span{}
	for _, raw := range rows {
		var row spanQueryRow
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, err
		}
		out[row.TraceID] = append(out[row.TraceID], store.Span{
			ID:         row.SpanID,
			ParentID:   row.ParentSpanID,
			Name:       row.Name,
			Service:    row.Service,
			DurationMS: row.DurationMS,
			Status:     row.Status,
			Attributes: safeMap(row.Attributes),
		})
	}
	return out, nil
}

func (sink *ClickHouseSink) insert(table string, rows any) error {
	body, err := jsonLines(rows)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	query := "INSERT INTO " + sink.database + "." + table + " FORMAT JSONEachRow"
	ctx, cancel := context.WithTimeout(context.Background(), sink.client.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sink.baseURL+"/?query="+url.QueryEscape(query), bytes.NewReader(body))
	if err != nil {
		return err
	}
	if sink.username != "" {
		req.SetBasicAuth(sink.username, sink.password)
	}
	resp, err := sink.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return errors.New("clickhouse insert failed: " + resp.Status + ": " + strings.TrimSpace(string(data)))
	}
	return nil
}

func (sink *ClickHouseSink) query(ctx context.Context, query string) ([]json.RawMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, sink.client.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodPost, sink.baseURL+"/?query="+url.QueryEscape(query), nil)
	if err != nil {
		return nil, err
	}
	if sink.username != "" {
		req.SetBasicAuth(sink.username, sink.password)
	}
	resp, err := sink.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, errors.New("clickhouse query failed: " + resp.Status + ": " + strings.TrimSpace(string(data)))
	}
	var rows []json.RawMessage
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		rows = append(rows, append(json.RawMessage(nil), line...))
	}
	return rows, scanner.Err()
}

func jsonLines(rows any) ([]byte, error) {
	data, err := json.Marshal(rows)
	if err != nil {
		return nil, err
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	for _, row := range raw {
		out.Write(row)
		out.WriteByte('\n')
	}
	return out.Bytes(), nil
}

func timestamp(value string) string {
	if value == "" {
		return clickHouseTime(time.Now())
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return clickHouseTime(parsed)
		}
	}
	return value
}

func apiTime(value string) string {
	if value == "" {
		return ""
	}
	for _, layout := range []string{"2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05", time.RFC3339Nano} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC().Format(time.RFC3339Nano)
		}
	}
	return value
}

func queryLimit(value int) int {
	if value <= 0 {
		return 100
	}
	if value > 500 {
		return 500
	}
	return value
}

func whereClause(filters []string) string {
	if len(filters) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(filters, " AND ")
}

func sqlString(value string) string {
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "'", "\\'")
	return "'" + escaped + "'"
}

func clickHouseTime(value time.Time) string {
	return value.UTC().Format("2006-01-02 15:04:05.000000000")
}

func environment(resources []store.Resource) string {
	for _, resource := range resources {
		if resource.Environment != "" {
			return resource.Environment
		}
	}
	return "production"
}

func safeMap(values map[string]string) map[string]string {
	if values == nil {
		return map[string]string{}
	}
	return values
}

type metricRow struct {
	Timestamp          string            `json:"timestamp"`
	OrganizationID     string            `json:"organization_id"`
	Environment        string            `json:"environment"`
	Service            string            `json:"service"`
	Host               string            `json:"host"`
	Region             string            `json:"region"`
	MetricName         string            `json:"metric_name"`
	MetricType         string            `json:"metric_type"`
	Unit               string            `json:"unit"`
	Value              float64           `json:"value"`
	Labels             map[string]string `json:"labels"`
	ResourceAttributes map[string]string `json:"resource_attributes"`
}

type logRow struct {
	Timestamp          string            `json:"timestamp"`
	OrganizationID     string            `json:"organization_id"`
	Environment        string            `json:"environment"`
	Service            string            `json:"service"`
	Host               string            `json:"host"`
	Region             string            `json:"region"`
	Severity           string            `json:"severity"`
	Message            string            `json:"message"`
	TraceID            string            `json:"trace_id"`
	SpanID             string            `json:"span_id"`
	Fields             map[string]string `json:"fields"`
	ResourceAttributes map[string]string `json:"resource_attributes"`
}

type traceRow struct {
	Timestamp          string            `json:"timestamp"`
	OrganizationID     string            `json:"organization_id"`
	Environment        string            `json:"environment"`
	TraceID            string            `json:"trace_id"`
	RootService        string            `json:"root_service"`
	Operation          string            `json:"operation"`
	DurationMS         float64           `json:"duration_ms"`
	Status             string            `json:"status"`
	ResourceAttributes map[string]string `json:"resource_attributes"`
}

type spanRow struct {
	Timestamp      string            `json:"timestamp"`
	OrganizationID string            `json:"organization_id"`
	Environment    string            `json:"environment"`
	TraceID        string            `json:"trace_id"`
	SpanID         string            `json:"span_id"`
	ParentSpanID   string            `json:"parent_span_id"`
	Service        string            `json:"service"`
	Name           string            `json:"name"`
	DurationMS     float64           `json:"duration_ms"`
	Status         string            `json:"status"`
	Attributes     map[string]string `json:"attributes"`
}

type uptimeRow struct {
	Timestamp      string  `json:"timestamp"`
	OrganizationID string  `json:"organization_id"`
	MonitorID      string  `json:"monitor_id"`
	Name           string  `json:"name"`
	URL            string  `json:"url"`
	Status         string  `json:"status"`
	ExpectedStatus int     `json:"expected_status"`
	StatusCode     int     `json:"status_code"`
	ResponseMS     float64 `json:"response_ms"`
	Error          string  `json:"error"`
}

type metricQueryRow struct {
	Timestamp          string            `json:"timestamp"`
	MetricName         string            `json:"metric_name"`
	Value              float64           `json:"value"`
	Unit               string            `json:"unit"`
	MetricType         string            `json:"metric_type"`
	Labels             map[string]string `json:"labels"`
	Environment        string            `json:"environment"`
	Service            string            `json:"service"`
	Host               string            `json:"host"`
	Region             string            `json:"region"`
	ResourceAttributes map[string]string `json:"resource_attributes"`
}

type logQueryRow struct {
	Timestamp          string            `json:"timestamp"`
	Severity           string            `json:"severity"`
	Message            string            `json:"message"`
	TraceID            string            `json:"trace_id"`
	SpanID             string            `json:"span_id"`
	Fields             map[string]string `json:"fields"`
	Environment        string            `json:"environment"`
	Service            string            `json:"service"`
	Host               string            `json:"host"`
	Region             string            `json:"region"`
	ResourceAttributes map[string]string `json:"resource_attributes"`
}

type traceQueryRow struct {
	Timestamp          string            `json:"timestamp"`
	TraceID            string            `json:"trace_id"`
	RootService        string            `json:"root_service"`
	Operation          string            `json:"operation"`
	DurationMS         float64           `json:"duration_ms"`
	Status             string            `json:"status"`
	Environment        string            `json:"environment"`
	ResourceAttributes map[string]string `json:"resource_attributes"`
}

type spanQueryRow struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id"`
	Service      string            `json:"service"`
	Name         string            `json:"name"`
	DurationMS   float64           `json:"duration_ms"`
	Status       string            `json:"status"`
	Attributes   map[string]string `json:"attributes"`
}
