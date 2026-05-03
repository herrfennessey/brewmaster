// Package main is the entry point for the brewmaster API service.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/herrfennessey/brewmaster/api/internal/ai"
	"github.com/herrfennessey/brewmaster/api/internal/router"
	"github.com/herrfennessey/brewmaster/api/internal/telemetry"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	Port             string
	Environment      string
	AxiomAPIToken    string
	TelemetryEnabled bool
}

func loadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "dev"
	}
	return Config{
		Port:             port,
		Environment:      env,
		AxiomAPIToken:    os.Getenv("AXIOM_API_TOKEN"),
		TelemetryEnabled: os.Getenv("TELEMETRY_ENABLED") != "false",
	}
}

// telemetryResult holds the initialized telemetry components.
type telemetryResult struct {
	tracer          trace.Tracer
	tracerProvider  trace.TracerProvider
	tracerShutdown  func(context.Context) error
	metricsShutdown func(context.Context) error
}

// initTelemetry initializes tracing and metrics if configured.
func initTelemetry(ctx context.Context, cfg *Config) telemetryResult {
	var result telemetryResult

	if !cfg.TelemetryEnabled || cfg.AxiomAPIToken == "" {
		slog.Info("Telemetry disabled or AXIOM_API_TOKEN not set; running without traces/metrics")
		return result
	}

	telemetryCfg := telemetry.Config{
		Enabled:       cfg.TelemetryEnabled,
		AxiomAPIToken: cfg.AxiomAPIToken,
		Environment:   cfg.Environment,
	}

	var err error
	result.tracer, result.tracerProvider, result.tracerShutdown, err = telemetry.InitTracer(ctx, telemetryCfg)
	if err != nil {
		slog.Error("Failed to initialize tracer", "error", err)
	} else {
		slog.Info("Tracing initialized", "environment", cfg.Environment)
	}

	result.metricsShutdown, err = telemetry.InitMetrics(ctx, telemetryCfg)
	if err != nil {
		slog.Error("Failed to initialize metrics", "error", err)
	} else {
		slog.Info("Metrics initialized", "environment", cfg.Environment)
	}

	return result
}

func shutdownTelemetry(ctx context.Context, tel telemetryResult) {
	if tel.tracerShutdown != nil {
		if err := tel.tracerShutdown(ctx); err != nil {
			slog.Error("Tracer shutdown error", "error", err)
		}
	}
	if tel.metricsShutdown != nil {
		if err := tel.metricsShutdown(ctx); err != nil {
			slog.Error("Metrics shutdown error", "error", err)
		}
	}
}

func main() {
	cfg := loadConfig()
	ctx := context.Background()

	tel := initTelemetry(ctx, &cfg)

	provider, err := ai.NewOpenAIProvider(tel.tracer)
	if err != nil {
		slog.Error("Failed to initialize AI provider", "error", err)
		os.Exit(1)
	}

	r := router.New(provider, tel.tracerProvider)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdownChan
		slog.Info("Shutting down gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Server shutdown error", "error", err)
		}
		shutdownTelemetry(shutdownCtx, tel)
	}()

	slog.Info("Starting server", "port", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped")
}
