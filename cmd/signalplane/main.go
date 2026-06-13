package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/chaktihor/signalplane/internal/server"
	"github.com/chaktihor/signalplane/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg := server.Config{
		Addr:         envString("SIGNALPLANE_ADDR", "127.0.0.1:4318"),
		IngestToken:  envString("SIGNALPLANE_INGEST_TOKEN", "dev-token"),
		ReadTimeout:  envDurationSeconds("SIGNALPLANE_READ_TIMEOUT_SECONDS", 5),
		WriteTimeout: envDurationSeconds("SIGNALPLANE_WRITE_TIMEOUT_SECONDS", 10),
		IdleTimeout:  envDurationSeconds("SIGNALPLANE_IDLE_TIMEOUT_SECONDS", 60),
	}

	data, err := store.Open(store.Options{
		Path:           envString("SIGNALPLANE_DATA_PATH", "data/signalplane.json"),
		Seed:           envBool("SIGNALPLANE_SEED_DEMO_DATA", true),
		BootstrapToken: cfg.IngestToken,
	})
	if err != nil {
		logger.Error("failed to open data store", "error", err)
		os.Exit(1)
	}

	app := server.New(cfg, data, logger)
	httpServer := app.HTTPServer()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	app.StartBackground(ctx)

	go func() {
		logger.Info("signalplane listening", "addr", cfg.Addr, "ingest_token", cfg.IngestToken)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("signalplane stopped")
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envDurationSeconds(key string, fallback int) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return time.Duration(fallback) * time.Second
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return time.Duration(fallback) * time.Second
	}
	return time.Duration(parsed) * time.Second
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "TRUE", "yes", "YES":
		return true
	case "0", "false", "FALSE", "no", "NO":
		return false
	default:
		return fallback
	}
}
