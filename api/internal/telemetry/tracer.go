package telemetry

import (
	"context"
	"crypto/tls"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	serviceName         = "brewmaster-api"
	serviceVersion      = "v1"
	instrumentationName = "github.com/herrfennessey/brewmaster"
)

// InitTracer initializes the OpenTelemetry tracer provider for Axiom.
// Returns the tracer, trace provider, and a shutdown function.
func InitTracer(ctx context.Context, cfg Config) (trace.Tracer, trace.TracerProvider, func(context.Context) error, error) {
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(axiomEndpoint),
		otlptracehttp.WithHeaders(cfg.tracesHeaders()),
		otlptracehttp.WithTLSClientConfig(&tls.Config{MinVersion: tls.VersionTLS12}),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
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
		return nil, nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	tracer := tp.Tracer(instrumentationName)

	shutdown := func(ctx context.Context) error {
		return tp.Shutdown(ctx) //nolint:wrapcheck
	}

	return tracer, tp, shutdown, nil
}
