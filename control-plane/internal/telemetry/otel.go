package telemetry

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// InitOTelSDK initializes OTel propagation and telemetry exporting to the meta-pipeline (PRD §10, Improvement #16).
func InitOTelSDK(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	// Configure global text map propagator (PRD §10 Observability)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	endpoint := os.Getenv("TELEMETRYHEALTH_META_OTLP_ENDPOINT")
	var tp *sdktrace.TracerProvider
	var exporter sdktrace.SpanExporter
	var err error

	if endpoint != "" {
		// In production, instantiate OTLP exporter pointing to the isolated meta-pipeline:
		exporter, err = otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}

		res, err := resource.New(ctx,
			resource.WithAttributes(
				semconv.ServiceNameKey.String(serviceName),
			),
		)
		if err != nil {
			return nil, err
		}

		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
	}

	shutdown := func(shutdownCtx context.Context) error {
		if tp != nil {
			if err := tp.Shutdown(shutdownCtx); err != nil {
				return err
			}
		}
		if exporter != nil {
			if err := exporter.Shutdown(shutdownCtx); err != nil {
				return err
			}
		}
		return nil
	}
	return shutdown, nil
}
