package telemetry

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

// InitMetrics initializes the OpenTelemetry metrics provider for Axiom.
// Returns a shutdown function.
func InitMetrics(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(axiomEndpoint),
		otlpmetrichttp.WithHeaders(cfg.metricsHeaders()),
		otlpmetrichttp.WithTLSClientConfig(&tls.Config{MinVersion: tls.VersionTLS12}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metrics exporter: %w", err)
	}

	hostID := getHostID()

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
			semconv.DeploymentEnvironmentName(cfg.Environment),
			semconv.HostID(hostID),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(
			exporter,
			metric.WithInterval(10*time.Second),
		)),
	)

	otel.SetMeterProvider(meterProvider)

	return meterProvider.Shutdown, nil
}

// getHostID returns a unique identifier for this host/instance.
// For Cloud Run, uses K_REVISION + hostname. Falls back to hostname.
func getHostID() string {
	hostname := getHostname()
	if revision := os.Getenv("K_REVISION"); revision != "" {
		return revision + "-" + hostname
	}
	return hostname
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
