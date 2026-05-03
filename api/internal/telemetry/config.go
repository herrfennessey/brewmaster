// Package telemetry provides OpenTelemetry tracing and metrics for the application.
package telemetry

// Axiom configuration constants (EU region).
const (
	axiomEndpoint       = "eu-central-1.aws.edge.axiom.co"
	axiomTracesDataset  = "brewmaster-traces"
	axiomMetricsDataset = "brewmaster-metrics"
)

// Config holds the telemetry configuration.
type Config struct {
	AxiomAPIToken string // AXIOM_API_TOKEN — Bearer token for Axiom
	Environment   string // Deployment environment (e.g., "prod", "staging", "dev")
	Enabled       bool   // Controls whether tracing and metrics are enabled
}

// tracesHeaders returns the headers for the traces exporter.
func (c Config) tracesHeaders() map[string]string {
	return map[string]string{
		"Authorization":   "Bearer " + c.AxiomAPIToken,
		"X-AXIOM-DATASET": axiomTracesDataset,
	}
}

// metricsHeaders returns the headers for the metrics exporter.
func (c Config) metricsHeaders() map[string]string {
	return map[string]string{
		"Authorization":   "Bearer " + c.AxiomAPIToken,
		"X-AXIOM-DATASET": axiomMetricsDataset,
	}
}
