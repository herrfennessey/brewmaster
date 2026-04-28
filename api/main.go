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

	"github.com/herrfennessey/brewmaster/api/internal/ai"
	"github.com/herrfennessey/brewmaster/api/internal/router"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	Port string
}

func loadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{Port: port}
}

func main() {
	cfg := loadConfig()

	provider, err := ai.NewOpenAIProvider()
	if err != nil {
		slog.Error("Failed to initialize AI provider", "error", err)
		os.Exit(1)
	}

	r := router.New(provider)

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
	}()

	slog.Info("Starting server", "port", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped")
}
