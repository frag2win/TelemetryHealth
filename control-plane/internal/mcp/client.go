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
	// Mocking the MCP query for the hackathon demo
	query := `SELECT * FROM traces WHERE attributes['service.name'] = 'ai-agent'`
	
	// In a real implementation:
	// import "github.com/signoz/mcp-go"
	// mcpClient := mcp.NewClient(serverURL)
	// return mcpClient.Query(ctx, query)

	_ = query // keep compiler happy
	
	return &Traces{
		Count: 2,
		Data: []map[string]interface{}{
			{"trace_id": "t1", "attributes": map[string]string{"llm.token_usage": "1500"}},
			{"trace_id": "t2", "attributes": map[string]string{"llm.tool_call.error": "TimeoutError"}},
		},
	}, nil
}
