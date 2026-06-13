package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	baseURL := env("SIGNALPLANE_URL", "http://127.0.0.1:4318")
	token := env("SIGNALPLANE_TOKEN", "dev-token")
	resource := map[string]any{
		"service":     "go-orders-api",
		"host":        "go-api-1",
		"environment": "production",
		"region":      "local",
		"version":     "0.1.0",
	}

	post(baseURL, token, "/api/ingest/metrics", map[string]any{
		"metrics": []map[string]any{
			{"name": "http.server.requests", "value": 1280, "unit": "requests", "type": "counter", "resource": resource},
			{"name": "http.server.duration", "value": 41, "unit": "ms", "type": "histogram", "resource": resource},
		},
	})
	post(baseURL, token, "/api/ingest/logs", map[string]any{
		"logs": []map[string]any{
			{"severity": "info", "message": "go orders API accepted checkout request", "traceId": "trace-go-orders-1", "resource": resource},
		},
	})
	post(baseURL, token, "/api/ingest/traces", map[string]any{
		"traceId": "trace-go-orders-1",
		"spans": []map[string]any{
			{"spanId": "span-go-root", "name": "POST /orders", "durationMs": 41, "status": "ok", "resource": resource, "timestamp": time.Now().UTC().Format(time.RFC3339Nano)},
		},
	})
}

func post(baseURL, token, path string, payload any) {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println(path, resp.Status)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
