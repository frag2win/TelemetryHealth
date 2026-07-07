package processor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

const (
	// typeStr is the identifier for the processor
	typeStr   = "telemetryhealth"
	stability = component.StabilityLevelAlpha
)

// NewFactory creates a factory for the telemetryhealth processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType(typeStr),
		createDefaultConfig,
		processor.WithTraces(createTracesProcessor, stability),
		processor.WithMetrics(createMetricsProcessor, stability),
		processor.WithLogs(createLogsProcessor, stability),
	)
}

// createTracesProcessor is required by the OpenTelemetry Collector factory pattern to instantiate the traces consumer.
func createTracesProcessor(ctx context.Context, set processor.Settings, cfg component.Config, nextConsumer consumer.Traces) (processor.Traces, error) {
	_ = ctx // Required by interface
	return newTracesConsumer(set, cfg, nextConsumer)
}

// createMetricsProcessor is required by the OpenTelemetry Collector factory pattern to instantiate the metrics consumer.
func createMetricsProcessor(ctx context.Context, set processor.Settings, cfg component.Config, nextConsumer consumer.Metrics) (processor.Metrics, error) {
	_ = ctx // Required by interface
	return newMetricsConsumer(set, cfg, nextConsumer)
}

// createLogsProcessor is required by the OpenTelemetry Collector factory pattern to instantiate the logs consumer.
func createLogsProcessor(ctx context.Context, set processor.Settings, cfg component.Config, nextConsumer consumer.Logs) (processor.Logs, error) {
	_ = ctx // Required by interface
	return newLogsConsumer(set, cfg, nextConsumer)
}
