package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
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
