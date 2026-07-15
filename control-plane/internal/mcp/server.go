package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// Server implements SigNoz MCP server tools for AI agent interaction.
type Server struct {
	toolset *Toolset
}

func NewServer(toolset *Toolset) *Server {
	return &Server{
		toolset: toolset,
	}
}

// ToolRequest represents an incoming MCP tool invocation.
type ToolRequest struct {
	ToolName  string          `json:"tool_name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolResponse represents the MCP tool execution result.
type ToolResponse struct {
	Success bool   `json:"success"`
	Data    string `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// HandleToolCall routes generic MCP tool calls to the appropriate handler.
func (s *Server) HandleToolCall(ctx context.Context, req ToolRequest) ToolResponse {
	switch req.ToolName {
	case "get_telemetry_health":
		var args struct {
			TenantID string `json:"tenant_id"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return ToolResponse{Success: false, Error: err.Error()}
		}
		resp, err := s.toolset.GetTelemetryHealth(ctx, args.TenantID)
		if err != nil {
			return ToolResponse{Success: false, Error: err.Error()}
		}
		return ToolResponse{Success: true, Data: fmt.Sprintf(`{"health_score": %f}`, resp.HealthScore)}

	case "generate_remediation":
		var args struct {
			IssueType string `json:"issue_type"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return ToolResponse{Success: false, Error: err.Error()}
		}
		yamlSnippet, err := s.toolset.GenerateRemediation(ctx, args.IssueType)
		if err != nil {
			return ToolResponse{Success: false, Error: err.Error()}
		}
		return ToolResponse{Success: true, Data: yamlSnippet}

	default:
		return ToolResponse{Success: false, Error: fmt.Sprintf("unsupported MCP tool: %s", req.ToolName)}
	}
}
