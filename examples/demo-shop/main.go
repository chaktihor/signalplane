package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	mathrand "math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	defaultAddr       = "127.0.0.1:8088"
	defaultSignalURL  = "http://127.0.0.1:4318"
	defaultToken      = "dev-token"
	serviceName       = "demo-checkout-api"
	hostName          = "demo-checkout-1"
	environment       = "production"
	region            = "local"
	version           = "demo-0.1.0"
	autoTrafficPeriod = 5 * time.Second
	heartbeatPeriod   = 10 * time.Second
)

type app struct {
	signalURL string
	token     string
	addr      string
	client    *http.Client
	requests  atomic.Uint64
	errors    atomic.Uint64
}

func main() {
	a := &app{
		signalURL: strings.TrimRight(env("SIGNALPLANE_URL", defaultSignalURL), "/"),
		token:     env("SIGNALPLANE_TOKEN", defaultToken),
		addr:      env("DEMO_SHOP_ADDR", defaultAddr),
		client:    &http.Client{Timeout: 5 * time.Second},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a.emitHostHeartbeat()
	a.registerUptimeMonitor()
	go a.heartbeatLoop(ctx)
	if envBool("DEMO_SHOP_AUTO_TRAFFIC", true) {
		go a.autoTrafficLoop(ctx)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", a.home)
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /checkout", a.checkout)
	mux.HandleFunc("POST /checkout", a.checkout)
	mux.HandleFunc("GET /traffic", a.traffic)

	log.Printf("demo shop listening on http://%s and sending telemetry to %s", a.addr, a.signalURL)
	if err := http.ListenAndServe(a.addr, mux); err != nil {
		log.Fatal(err)
	}
}

func (a *app) home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := map[string]any{
		"SignalURL": a.signalURL,
		"Requests":  a.requests.Load(),
		"Errors":    a.errors.Load(),
	}
	if err := homeTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *app) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"service":   serviceName,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func (a *app) checkout(w http.ResponseWriter, r *http.Request) {
	fail := r.URL.Query().Get("fail") == "true"
	result := a.simulateCheckout(r.Context(), fail, "manual")
	status := http.StatusOK
	if result.Failed {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, result)
}

func (a *app) traffic(w http.ResponseWriter, r *http.Request) {
	count := intParam(r, "count", 8)
	failEvery := intParam(r, "failEvery", 4)
	if count < 1 {
		count = 1
	}
	if count > 50 {
		count = 50
	}

	results := make([]checkoutResult, 0, count)
	for i := 1; i <= count; i++ {
		fail := failEvery > 0 && i%failEvery == 0
		results = append(results, a.simulateCheckout(r.Context(), fail, "traffic"))
	}
	writeJSON(w, http.StatusOK, map[string]any{"generated": len(results), "results": results})
}

func (a *app) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(heartbeatPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.emitHostHeartbeat()
		}
	}
}

func (a *app) autoTrafficLoop(ctx context.Context) {
	ticker := time.NewTicker(autoTrafficPeriod)
	defer ticker.Stop()
	for i := 1; ; i++ {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.simulateCheckout(ctx, i%5 == 0, "auto")
		}
	}
}

type checkoutResult struct {
	OrderID    string  `json:"orderId"`
	TraceID    string  `json:"traceId"`
	DurationMS float64 `json:"durationMs"`
	Amount     float64 `json:"amount"`
	Failed     bool    `json:"failed"`
	Message    string  `json:"message"`
}

func (a *app) simulateCheckout(ctx context.Context, fail bool, source string) checkoutResult {
	start := time.Now()
	traceID := "trace-" + shortID()
	rootSpan := "span-" + shortID()
	paymentSpan := "span-" + shortID()
	inventorySpan := "span-" + shortID()
	orderID := "ord-" + shortID()
	amount := roundMoney(45 + mathrand.Float64()*125)

	inventoryMS := float64(18 + mathrand.IntN(28))
	paymentMS := float64(80 + mathrand.IntN(280))
	dbMS := float64(16 + mathrand.IntN(45))
	if fail {
		paymentMS += 420
	}
	durationMS := float64(time.Since(start).Milliseconds()) + inventoryMS + paymentMS + dbMS + float64(20+mathrand.IntN(60))

	a.requests.Add(1)
	message := fmt.Sprintf("checkout completed order=%s amount=%.2f source=%s", orderID, amount, source)
	severity := "info"
	status := "ok"
	errorRate := 0.5
	if fail {
		a.errors.Add(1)
		message = fmt.Sprintf("checkout failed order=%s reason=payment_gateway_timeout source=%s", orderID, source)
		severity = "error"
		status = "error"
		errorRate = 12.5
	}

	a.emitMetrics(ctx, durationMS, amount, fail, errorRate)
	a.emitLog(ctx, severity, message, traceID, rootSpan, source, orderID)
	a.emitTrace(ctx, traceID, rootSpan, inventorySpan, paymentSpan, durationMS, inventoryMS, paymentMS, dbMS, status, orderID)

	return checkoutResult{
		OrderID:    orderID,
		TraceID:    traceID,
		DurationMS: round(durationMS, 1),
		Amount:     amount,
		Failed:     fail,
		Message:    message,
	}
}

func (a *app) emitHostHeartbeat() {
	payload := map[string]any{
		"id":           hostName,
		"name":         hostName,
		"environment":  environment,
		"region":       region,
		"status":       "online",
		"agentVersion": "demo-shop-emitter",
		"tags":         []string{"demo", "silver", "checkout"},
		"metrics": map[string]float64{
			"cpu":    round(22+mathrand.Float64()*58, 1),
			"memory": round(45+mathrand.Float64()*32, 1),
			"disk":   41.2,
		},
	}
	a.post("/api/ingest/hosts", payload)
}

func (a *app) registerUptimeMonitor() {
	payload := map[string]any{
		"id":              "upt-demo-shop",
		"name":            "Demo shop health",
		"url":             "http://" + a.addr + "/healthz",
		"method":          "GET",
		"expectedStatus":  200,
		"intervalSeconds": 15,
		"timeoutSeconds":  3,
	}
	a.post("/api/uptime-monitors", payload)
}

func (a *app) emitMetrics(ctx context.Context, durationMS, amount float64, failed bool, errorRate float64) {
	metrics := []map[string]any{
		{"name": "http.server.requests", "value": 1, "unit": "requests", "type": "counter", "labels": map[string]string{"route": "/checkout", "method": "POST"}, "resource": resource()},
		{"name": "http.server.request_rate", "value": round(18+mathrand.Float64()*34, 1), "unit": "requests/s", "type": "gauge", "labels": map[string]string{"route": "/checkout"}, "resource": resource()},
		{"name": "http.server.duration", "value": round(durationMS, 1), "unit": "ms", "type": "histogram", "labels": map[string]string{"route": "/checkout"}, "resource": resource()},
		{"name": "http.server.error_rate", "value": errorRate, "unit": "percent", "type": "gauge", "resource": resource()},
		{"name": "checkout.revenue", "value": amount, "unit": "usd", "type": "counter", "resource": resource()},
	}
	if failed {
		metrics = append(metrics, map[string]any{"name": "checkout.failures", "value": 1, "unit": "errors", "type": "counter", "resource": resource()})
	} else {
		metrics = append(metrics, map[string]any{"name": "checkout.orders.created", "value": 1, "unit": "orders", "type": "counter", "resource": resource()})
	}
	a.postWithContext(ctx, "/api/ingest/metrics", map[string]any{"metrics": metrics})
}

func (a *app) emitLog(ctx context.Context, severity, message, traceID, spanID, source, orderID string) {
	payload := map[string]any{
		"severity": severity,
		"message":  message,
		"traceId":  traceID,
		"spanId":   spanID,
		"fields": map[string]string{
			"source":  source,
			"orderId": orderID,
		},
		"resource": resource(),
	}
	a.postWithContext(ctx, "/api/ingest/logs", payload)
}

func (a *app) emitTrace(ctx context.Context, traceID, rootSpan, inventorySpan, paymentSpan string, durationMS, inventoryMS, paymentMS, dbMS float64, status, orderID string) {
	paymentStatus := "ok"
	if status == "error" {
		paymentStatus = "error"
	}
	payload := map[string]any{
		"traceId": traceID,
		"spans": []map[string]any{
			{"spanId": rootSpan, "name": "POST /checkout", "durationMs": round(durationMS, 1), "status": status, "resource": resource(), "attributes": map[string]string{"orderId": orderID}},
			{"spanId": inventorySpan, "parentId": rootSpan, "name": "GET /inventory/reserve", "durationMs": round(inventoryMS, 1), "status": "ok", "resource": dependencyResource("inventory-service")},
			{"spanId": paymentSpan, "parentId": rootSpan, "name": "POST /payment/authorize", "durationMs": round(paymentMS, 1), "status": paymentStatus, "resource": dependencyResource("payment-gateway")},
			{"spanId": "span-" + shortID(), "parentId": rootSpan, "name": "INSERT orders", "durationMs": round(dbMS, 1), "status": "ok", "resource": dependencyResource("orders-postgres")},
		},
	}
	a.postWithContext(ctx, "/api/ingest/traces", payload)
}

func (a *app) post(path string, payload any) {
	a.postWithContext(context.Background(), path, payload)
}

func (a *app) postWithContext(ctx context.Context, path string, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("encode telemetry payload: %v", err)
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.signalURL+path, bytes.NewReader(body))
	if err != nil {
		log.Printf("create telemetry request: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		log.Printf("send telemetry %s: %v", path, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("send telemetry %s: status %s", path, resp.Status)
	}
}

func resource() map[string]any {
	return map[string]any{
		"service":     serviceName,
		"host":        hostName,
		"environment": environment,
		"region":      region,
		"version":     version,
		"attributes": map[string]string{
			"team":    "checkout",
			"runtime": "go",
			"app":     "demo-shop",
		},
	}
}

func dependencyResource(service string) map[string]any {
	return map[string]any{
		"service":     service,
		"environment": environment,
		"region":      region,
		"version":     version,
		"attributes": map[string]string{
			"team": "checkout",
			"app":  "demo-shop",
		},
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func intParam(r *http.Request, key string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return fallback
	}
	return value
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func shortID() string {
	var bytes [4]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(bytes[:])
}

func round(value float64, places int) float64 {
	scale := math.Pow(10, float64(places))
	return math.Round(value*scale) / scale
}

func roundMoney(value float64) float64 {
	return round(value, 2)
}

var homeTemplate = template.Must(template.New("home").Parse(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Demo Shop</title>
    <style>
      body { margin: 0; font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #f5f7f8; color: #172026; }
      main { max-width: 880px; margin: 48px auto; padding: 0 24px; }
      h1 { font-size: 34px; margin: 0 0 8px; letter-spacing: 0; }
      p { color: #52616b; line-height: 1.5; }
      .grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin: 24px 0; }
      .card { background: #fff; border: 1px solid #dde4e8; border-radius: 8px; padding: 16px; }
      .card strong { display: block; font-size: 24px; margin-top: 6px; }
      .actions { display: flex; gap: 10px; flex-wrap: wrap; margin-top: 20px; }
      a { display: inline-flex; align-items: center; height: 38px; padding: 0 14px; border-radius: 6px; text-decoration: none; background: #172026; color: #fff; }
      a.secondary { background: #e7ecef; color: #172026; }
      code { background: #e7ecef; border-radius: 4px; padding: 2px 5px; }
      @media (max-width: 720px) { .grid { grid-template-columns: 1fr; } }
    </style>
  </head>
  <body>
    <main>
      <h1>Demo Shop Checkout</h1>
      <p>This small application sends logs, metrics, traces, host heartbeats, and uptime monitor registration to <code>{{.SignalURL}}</code>.</p>
      <div class="grid">
        <div class="card">Requests observed<strong>{{.Requests}}</strong></div>
        <div class="card">Errors emitted<strong>{{.Errors}}</strong></div>
        <div class="card">Service<strong>` + serviceName + `</strong></div>
      </div>
      <div class="actions">
        <a href="/checkout">Successful checkout</a>
        <a href="/checkout?fail=true">Fail checkout</a>
        <a class="secondary" href="/traffic?count=12&failEvery=4">Generate traffic</a>
        <a class="secondary" href="/healthz">Health</a>
      </div>
    </main>
  </body>
</html>`))
