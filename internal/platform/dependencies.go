package platform

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type DependencyCheck struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Target string `json:"target"`
}

type DependencyStatus struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Kind        string  `json:"kind"`
	Target      string  `json:"target"`
	Status      string  `json:"status"`
	LatencyMS   float64 `json:"latencyMs"`
	LastChecked string  `json:"lastChecked"`
	Error       string  `json:"error,omitempty"`
}

func ChecksFromEnv() []DependencyCheck {
	checks := []DependencyCheck{
		{ID: "postgres", Name: "PostgreSQL metadata store", Kind: "tcp", Target: env("SIGNALPLANE_POSTGRES_ADDR", "")},
		{ID: "clickhouse", Name: "ClickHouse telemetry store", Kind: "http", Target: env("SIGNALPLANE_CLICKHOUSE_HTTP_URL", "")},
		{ID: "otel-grpc", Name: "OTLP gRPC receiver", Kind: "tcp", Target: env("SIGNALPLANE_OTEL_GRPC_ADDR", "")},
		{ID: "otel-http", Name: "OTLP HTTP receiver", Kind: "tcp", Target: env("SIGNALPLANE_OTEL_HTTP_ADDR", "")},
		{ID: "smtp", Name: "SMTP notification sink", Kind: "tcp", Target: env("SIGNALPLANE_SMTP_ADDR", "")},
		{ID: "mailpit", Name: "Mailpit web UI", Kind: "http", Target: env("SIGNALPLANE_MAILPIT_URL", "")},
	}
	out := make([]DependencyCheck, 0, len(checks))
	for _, check := range checks {
		if strings.TrimSpace(check.Target) == "" {
			continue
		}
		out = append(out, check)
	}
	return out
}

func CheckAll(ctx context.Context, checks []DependencyCheck) []DependencyStatus {
	statuses := make([]DependencyStatus, 0, len(checks))
	for _, check := range checks {
		statuses = append(statuses, Check(ctx, check))
	}
	return statuses
}

func Check(ctx context.Context, check DependencyCheck) DependencyStatus {
	start := time.Now()
	status := DependencyStatus{
		ID:          check.ID,
		Name:        check.Name,
		Kind:        check.Kind,
		Target:      check.Target,
		Status:      "up",
		LastChecked: time.Now().UTC().Format(time.RFC3339Nano),
	}

	var err error
	switch check.Kind {
	case "http":
		err = checkHTTP(ctx, check.Target)
	default:
		err = checkTCP(ctx, check.Target)
	}
	status.LatencyMS = float64(time.Since(start).Microseconds()) / 1000
	if err != nil {
		status.Status = "down"
		status.Error = err.Error()
	}
	return status
}

func checkTCP(ctx context.Context, target string) error {
	checkCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(checkCtx, "tcp", target)
	if err != nil {
		return err
	}
	return conn.Close()
}

func checkHTTP(ctx context.Context, target string) error {
	checkCtx, cancel := context.WithTimeout(ctx, 1200*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return &statusError{status: resp.Status}
	}
	return nil
}

type statusError struct {
	status string
}

func (err *statusError) Error() string {
	return "unexpected status " + err.status
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
