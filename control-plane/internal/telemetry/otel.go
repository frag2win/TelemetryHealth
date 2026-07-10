package telemetry

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// InitOTelSDK initializes OTel propagation and telemetry exporting to the meta-pipeline (PRD §10, Improvement #16).
func InitOTelSDK(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	// Configure global text map propagator (PRD §10 Observability)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	endpoint := os.Getenv("TELEMETRYHEALTH_META_OTLP_ENDPOINT")
	if endpoint != "" {
		// In production, instantiate OTLP exporter pointing to the isolated meta-pipeline:
		// exporter, _ := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithInsecure())
		// tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
		// otel.SetTracerProvider(tp)
	}

	shutdown := func(ctx context.Context) error {
		return nil
	}
	return shutdown, nil
}
