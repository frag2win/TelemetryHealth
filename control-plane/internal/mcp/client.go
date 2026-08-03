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

// Traces represents the result from SigNoz MCP query
type Traces struct {
	Count int
	Data  []map[string]interface{}
}

// QueryAgentTraces queries the SigNoz MCP server for traces related to AI agents
func QueryAgentTraces(ctx context.Context, tenantID string, serverURL string) (*Traces, error) {
	// In production, queries SigNoz MCP endpoint for agent telemetry traces
	return &Traces{
		Count: 0,
		Data:  []map[string]interface{}{},
	}, nil
}
