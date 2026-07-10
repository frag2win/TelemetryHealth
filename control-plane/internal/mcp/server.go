package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/remediation"
	"go.uber.org/zap"
)

// Server implements SigNoz MCP server tools for AI agent interaction.
type Server struct {
	logger    *zap.Logger
	generator *remediation.Generator
}

func NewServer(logger *zap.Logger) *Server {
	return &Server{
		logger:    logger,
		generator: remediation.NewGenerator(logger),
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

// GetTelemetryHealth returns the pipeline health score for a tenant via MCP.
func (s *Server) GetTelemetryHealth(ctx context.Context, tenantID string) (int, error) {
	s.logger.Info("MCP tool invoked: get_telemetry_health", zap.String("tenant_id", tenantID))
	// In production, queries ClickHouse health repository
	return 85, nil
}

// GenerateRemediation returns an auto-generated OTel collector YAML snippet via MCP.
func (s *Server) GenerateRemediation(ctx context.Context, issueType string) (string, error) {
	s.logger.Info("MCP tool invoked: generate_remediation", zap.String("issue_type", issueType))
	return s.generator.Generate(ctx, issueType)
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
		score, err := s.GetTelemetryHealth(ctx, args.TenantID)
		if err != nil {
			return ToolResponse{Success: false, Error: err.Error()}
		}
		return ToolResponse{Success: true, Data: fmt.Sprintf(`{"health_score": %d}`, score)}

	case "generate_remediation":
		var args struct {
			IssueType string `json:"issue_type"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return ToolResponse{Success: false, Error: err.Error()}
		}
		yamlSnippet, err := s.GenerateRemediation(ctx, args.IssueType)
		if err != nil {
			return ToolResponse{Success: false, Error: err.Error()}
		}
		return ToolResponse{Success: true, Data: yamlSnippet}

	default:
		return ToolResponse{Success: false, Error: fmt.Sprintf("unsupported MCP tool: %s", req.ToolName)}
	}
}
