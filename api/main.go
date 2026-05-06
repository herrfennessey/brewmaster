// Package main is the entry point for the brewmaster API service.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"go.opentelemetry.io/otel/trace"

	"github.com/herrfennessey/brewmaster/api/internal/ai"
	"github.com/herrfennessey/brewmaster/api/internal/auth"
	"github.com/herrfennessey/brewmaster/api/internal/router"
	"github.com/herrfennessey/brewmaster/api/internal/store"
	"github.com/herrfennessey/brewmaster/api/internal/telemetry"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	Port             string
	Environment      string
	AxiomAPIToken    string
	GCPProjectID     string
	TelemetryEnabled bool
	DisableAuth      bool
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
		GCPProjectID:     os.Getenv("GCP_PROJECT_ID"),
		TelemetryEnabled: os.Getenv("TELEMETRY_ENABLED") != "false",
		DisableAuth:      os.Getenv("DISABLE_AUTH") == "true",
	}
}

// initStorage initializes Firestore and the Firebase auth verifier. Returns
// nil components when the project id is unset (local dev without storage) so
// the rest of the service still boots. The shutdown closure releases the
// Firestore client.
func initStorage(ctx context.Context, cfg *Config) (store.Repo, auth.Verifier, func(), error) {
	if cfg.GCPProjectID == "" {
		if cfg.Environment == "production" {
			return nil, nil, nil, errors.New(
				"GCP_PROJECT_ID is required in production but is empty; protected /api/coffees/* routes would not register")
		}
		slog.Warn("GCP_PROJECT_ID not set; protected routes (/api/coffees/*) will return 404 — fine for local dev only")
		return nil, nil, func() {}, nil
	}

	fsClient, err := firestore.NewClient(ctx, cfg.GCPProjectID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("init firestore: %w", err)
	}
	repo := store.NewFirestoreRepo(fsClient)
	shutdown := func() {
		if closeErr := fsClient.Close(); closeErr != nil {
			slog.Error("firestore close error", "error", closeErr)
		}
	}

	if cfg.DisableAuth {
		slog.Info("DISABLE_AUTH=true; auth middleware will inject local-dev uid")
		return repo, nil, shutdown, nil
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.GCPProjectID})
	if err != nil {
		shutdown()
		return nil, nil, nil, fmt.Errorf("init firebase app: %w", err)
	}
	authClient, err := app.Auth(ctx)
	if err != nil {
		shutdown()
		return nil, nil, nil, fmt.Errorf("init firebase auth: %w", err)
	}
	return repo, verifierAdapter{authClient}, shutdown, nil
}

// verifierAdapter narrows *firebaseauth.Client to the auth.Verifier interface.
// It also wraps verify errors so wrapcheck stays happy.
type verifierAdapter struct{ c *firebaseauth.Client }

func (a verifierAdapter) VerifyIDToken(ctx context.Context, token string) (*firebaseauth.Token, error) {
	tok, err := a.c.VerifyIDToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("verify id token: %w", err)
	}
	return tok, nil
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

	repo, verifier, shutdownStore, err := initStorage(ctx, &cfg)
	if err != nil {
		slog.Error("Failed to initialize storage", "error", err)
		os.Exit(1)
	}
	defer shutdownStore()

	r := router.New(provider, repo, verifier, tel.tracerProvider)

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
		shutdownStore()
		os.Exit(1) //nolint:gocritic // shutdownStore already invoked above
	}

	slog.Info("Server stopped")
}
