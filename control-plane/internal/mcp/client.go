package mcp

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// InjectTraceContext injects OTel trace context into the outgoing request headers.
func InjectTraceContext(ctx context.Context, req *http.Request) *http.Request {
	req = req.WithContext(ctx)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	return req
}
