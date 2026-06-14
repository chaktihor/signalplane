package server

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chaktihor/signalplane/internal/platform"
	"github.com/chaktihor/signalplane/internal/store"
)

//go:embed web/*
var webFS embed.FS

type Config struct {
	Addr         string
	IngestToken  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	Dependencies []platform.DependencyCheck
}

type Server struct {
	cfg    Config
	store  *store.Store
	logger *slog.Logger
	mux    *http.ServeMux
}

func New(cfg Config, data *store.Store, logger *slog.Logger) *Server {
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:4318"
	}
	if cfg.IngestToken == "" {
		cfg.IngestToken = "dev-token"
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 5 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 10 * time.Second
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 60 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{cfg: cfg, store: data, logger: logger, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) HTTPServer() *http.Server {
	return &http.Server{
		Addr: s.cfg.Addr, Handler: s.mux, ReadTimeout: s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout, IdleTimeout: s.cfg.IdleTimeout,
	}
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) StartBackground(ctx context.Context) {
	go s.uptimeLoop(ctx)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("GET /api/bootstrap", s.bootstrap)
	s.mux.HandleFunc("GET /api/services", s.services)
	s.mux.HandleFunc("GET /api/hosts", s.hosts)
	s.mux.HandleFunc("GET /api/metrics", s.metrics)
	s.mux.HandleFunc("GET /api/logs", s.logs)
	s.mux.HandleFunc("GET /api/traces", s.traces)
	s.mux.HandleFunc("GET /api/alerts", s.alerts)
	s.mux.HandleFunc("PATCH /api/alerts/{id}", s.updateAlert)
	s.mux.HandleFunc("GET /api/incidents", s.incidents)
	s.mux.HandleFunc("POST /api/incidents", s.createIncident)
	s.mux.HandleFunc("GET /api/uptime-monitors", s.uptimeMonitors)
	s.mux.HandleFunc("POST /api/uptime-monitors", s.createUptimeMonitor)
	s.mux.HandleFunc("POST /api/uptime-monitors/{id}/check", s.checkUptimeMonitor)
	s.mux.HandleFunc("GET /api/system/dependencies", s.dependencies)
	s.mux.HandleFunc("GET /api/tokens", s.tokens)
	s.mux.HandleFunc("POST /api/tokens", s.createToken)
	s.mux.HandleFunc("POST /api/ingest/hosts", s.ingestHost)
	s.mux.HandleFunc("POST /api/ingest/metrics", s.ingestMetrics)
	s.mux.HandleFunc("POST /api/ingest/logs", s.ingestLogs)
	s.mux.HandleFunc("POST /api/ingest/traces", s.ingestTraces)
	s.mux.HandleFunc("GET /api/openapi", s.openapi)

	sub, _ := fs.Sub(webFS, "web")
	s.mux.Handle("/", http.FileServer(http.FS(sub)))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "signalplane", "timestamp": time.Now().UTC().Format(time.RFC3339Nano)})
}
func (s *Server) bootstrap(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Summary())
}
func (s *Server) services(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"services": s.store.Services(limitParam(r))})
}
func (s *Server) hosts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"hosts": s.store.Hosts(limitParam(r))})
}
func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"metrics": s.store.Metrics(limitParam(r))})
}
func (s *Server) logs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	writeJSON(w, http.StatusOK, map[string]any{"logs": s.store.Logs(limitParam(r), q.Get("service"), q.Get("severity"), q.Get("q"))})
}
func (s *Server) traces(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	writeJSON(w, http.StatusOK, map[string]any{"traces": s.store.Traces(limitParam(r), q.Get("service"), q.Get("status"))})
}
func (s *Server) alerts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"alerts": s.store.Alerts(limitParam(r))})
}
func (s *Server) incidents(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"incidents": s.store.Incidents(limitParam(r))})
}
func (s *Server) uptimeMonitors(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"uptimeMonitors": s.store.UptimeMonitors(limitParam(r))})
}
func (s *Server) dependencies(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	writeJSON(w, http.StatusOK, map[string]any{"dependencies": platform.CheckAll(ctx, s.cfg.Dependencies)})
}
func (s *Server) tokens(w http.ResponseWriter, r *http.Request) {
	if !s.authorizedScope(w, r, "admin") {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": s.store.Tokens(limitParam(r))})
}
func (s *Server) createToken(w http.ResponseWriter, r *http.Request) {
	if !s.authorizedScope(w, r, "admin") {
		return
	}
	var input store.TokenInput
	if !decodeJSON(w, r, &input) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"token": s.store.CreateToken(input)})
}
func (s *Server) createIncident(w http.ResponseWriter, r *http.Request) {
	if !s.authorizedScope(w, r, "admin") {
		return
	}
	var input store.Incident
	if !decodeJSON(w, r, &input) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"incident": s.store.CreateIncident(input)})
}
func (s *Server) createUptimeMonitor(w http.ResponseWriter, r *http.Request) {
	if !s.authorizedScope(w, r, "admin") {
		return
	}
	var input store.UptimeMonitor
	if !decodeJSON(w, r, &input) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"uptimeMonitor": s.store.CreateUptimeMonitor(input)})
}
func (s *Server) checkUptimeMonitor(w http.ResponseWriter, r *http.Request) {
	if !s.authorizedScope(w, r, "admin") {
		return
	}
	monitor, ok := s.store.UptimeMonitor(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "uptime monitor not found")
		return
	}
	result := s.checkUptime(r.Context(), monitor)
	updated, _ := s.store.RecordUptimeResult(result)
	writeJSON(w, http.StatusOK, map[string]any{"uptimeMonitor": updated})
}
func (s *Server) updateAlert(w http.ResponseWriter, r *http.Request) {
	if !s.authorizedScope(w, r, "admin") {
		return
	}
	var input struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	alert, ok := s.store.UpdateAlert(r.PathValue("id"), input.Status)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "alert not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"alert": alert})
}

func (s *Server) ingestHost(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r) {
		return
	}
	var input store.HostInput
	if !decodeJSON(w, r, &input) {
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"host": s.store.UpsertHost(input)})
}
func (s *Server) ingestMetrics(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r) {
		return
	}
	var input []store.MetricInput
	if !decodeJSONList(w, r, &input, "metrics") {
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": len(input), "metrics": s.store.IngestMetrics(input)})
}
func (s *Server) ingestLogs(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r) {
		return
	}
	var input []store.LogInput
	if !decodeJSONList(w, r, &input, "logs") {
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": len(input), "logs": s.store.IngestLogs(input)})
}
func (s *Server) ingestTraces(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r) {
		return
	}
	var input []store.TraceInput
	if !decodeJSONList(w, r, &input, "traces") {
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": len(input), "traces": s.store.IngestTraces(input)})
}

func (s *Server) openapi(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"openapi": "3.1.0",
		"info":    map[string]string{"title": "SignalPlane Silver API", "version": "0.1.0"},
		"paths":   []string{"/healthz", "/api/bootstrap", "/api/services", "/api/hosts", "/api/metrics", "/api/logs", "/api/traces", "/api/alerts", "/api/incidents", "/api/uptime-monitors", "/api/uptime-monitors/{id}/check", "/api/system/dependencies", "/api/tokens", "/api/ingest/hosts", "/api/ingest/metrics", "/api/ingest/logs", "/api/ingest/traces"},
	})
}

func (s *Server) authorized(w http.ResponseWriter, r *http.Request) bool {
	return s.authorizedScope(w, r, "ingest")
}

func (s *Server) authorizedScope(w http.ResponseWriter, r *http.Request, scope string) bool {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	token := strings.TrimSpace(r.Header.Get("X-SignalPlane-Token"))
	if strings.HasPrefix(auth, "Bearer ") {
		token = strings.TrimPrefix(auth, "Bearer ")
	}
	if s.store.ValidToken(token, scope) || token == s.cfg.IngestToken {
		return true
	}
	writeError(w, http.StatusUnauthorized, "unauthorized", "provide API token with Authorization: Bearer <token> or X-SignalPlane-Token")
	return false
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", "request body must be valid JSON: "+err.Error())
		return false
	}
	return true
}
func decodeJSONList[T any](w http.ResponseWriter, r *http.Request, target *[]T, wrapper string) bool {
	defer r.Body.Close()
	var raw json.RawMessage
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20))
	if err := decoder.Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", "request body must be valid JSON: "+err.Error())
		return false
	}
	if len(raw) > 0 && raw[0] == '[' {
		if err := json.Unmarshal(raw, target); err != nil {
			writeError(w, http.StatusBadRequest, "bad_json", err.Error())
			return false
		}
		return true
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err == nil {
		if payload, ok := envelope[wrapper]; ok {
			if err := json.Unmarshal(payload, target); err != nil {
				writeError(w, http.StatusBadRequest, "bad_json", err.Error())
				return false
			}
			return true
		}
	}
	var single T
	if err := json.Unmarshal(raw, &single); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", err.Error())
		return false
	}
	*target = []T{single}
	return true
}
func limitParam(r *http.Request) int {
	value, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || value <= 0 {
		return 100
	}
	return value
}

func (s *Server) uptimeLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runDueUptimeChecks(ctx)
		}
	}
}

func (s *Server) runDueUptimeChecks(ctx context.Context) {
	for _, monitor := range s.store.UptimeMonitors(500) {
		if !uptimeCheckDue(monitor) {
			continue
		}
		result := s.checkUptime(ctx, monitor)
		if _, ok := s.store.RecordUptimeResult(result); !ok {
			s.logger.Warn("uptime monitor disappeared before result could be recorded", "id", monitor.ID)
		}
	}
}

func (s *Server) checkUptime(ctx context.Context, monitor store.UptimeMonitor) store.UptimeResult {
	method := monitor.Method
	if method == "" {
		method = http.MethodGet
	}
	timeout := time.Duration(monitor.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(checkCtx, method, monitor.URL, nil)
	if err != nil {
		return store.UptimeResult{ID: monitor.ID, Status: "down", Error: err.Error(), CheckedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	}
	resp, err := http.DefaultClient.Do(req)
	elapsed := float64(time.Since(start).Microseconds()) / 1000
	if err != nil {
		return store.UptimeResult{ID: monitor.ID, Status: "down", ResponseMS: elapsed, Error: err.Error(), CheckedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	}
	defer resp.Body.Close()

	status := "down"
	errorMessage := ""
	if resp.StatusCode == monitor.ExpectedStatus {
		status = "up"
	} else {
		errorMessage = "expected status " + strconv.Itoa(monitor.ExpectedStatus) + ", got " + strconv.Itoa(resp.StatusCode)
	}
	return store.UptimeResult{
		ID:         monitor.ID,
		Status:     status,
		StatusCode: resp.StatusCode,
		ResponseMS: elapsed,
		Error:      errorMessage,
		CheckedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func uptimeCheckDue(monitor store.UptimeMonitor) bool {
	if monitor.URL == "" {
		return false
	}
	if monitor.LastCheckedAt == "" {
		return true
	}
	interval := time.Duration(monitor.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}
	last, err := time.Parse(time.RFC3339Nano, monitor.LastCheckedAt)
	if err != nil {
		return true
	}
	return time.Since(last) >= interval
}
