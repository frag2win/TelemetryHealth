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
	if s.toolset == nil {
		return ToolResponse{Success: false, Error: "toolset is not initialized"}
	}
	switch req.ToolName {
	case "get_telemetry_health":
		if s.toolset.HealthRepo == nil {
			return ToolResponse{Success: false, Error: "health repository not configured — ClickHouse unavailable"}
		}
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
		respBytes, err := json.Marshal(resp)
		if err != nil {
			return ToolResponse{Success: false, Error: fmt.Sprintf("failed to marshal response: %v", err)}
		}
		return ToolResponse{Success: true, Data: string(respBytes)}

	case "generate_remediation":
		if s.toolset.Generator == nil {
			return ToolResponse{Success: false, Error: "remediation generator not configured"}
		}
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

// JSON-RPC 2.0 structures for MCP integration

type jsonrpcRequest struct {
	JsonRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type jsonrpcResponse struct {
	JsonRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// HandleJSONRPCMessage processes raw JSON-RPC 2.0 messages from MCP clients
// and returns the serialized JSON response.
func (s *Server) HandleJSONRPCMessage(ctx context.Context, messageBytes []byte) ([]byte, error) {
	var req jsonrpcRequest
	if err := json.Unmarshal(messageBytes, &req); err != nil {
		resp := jsonrpcResponse{
			JsonRPC: "2.0",
			Error: &rpcError{
				Code:    -32700,
				Message: "Parse error",
			},
			ID: nil,
		}
		return json.Marshal(resp)
	}

	if req.JsonRPC != "2.0" {
		resp := jsonrpcResponse{
			JsonRPC: "2.0",
			Error: &rpcError{
				Code:    -32600,
				Message: "Invalid Request: expected jsonrpc v2.0",
			},
			ID: req.ID,
		}
		return json.Marshal(resp)
	}

	isNotification := req.ID == nil

	var result interface{}
	var rpcErr *rpcError

	switch req.Method {
	case "initialize":
		result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "TelemetryHealth",
				"version": "v1.1.0-mcp",
			},
		}

	case "notifications/initialized":
		return nil, nil

	case "ping":
		result = map[string]interface{}{}

	case "tools/list":
		result = map[string]interface{}{
			"tools": []map[string]interface{}{
				{
					"name":        "get_telemetry_health",
					"description": "Queries real-time composite health score, cardinality metrics, and orphan span rates for a tenant. Requires tenant_id.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"tenant_id": map[string]interface{}{
								"type":        "string",
								"description": "The unique identifier of the tenant",
							},
						},
						"required": []string{"tenant_id"},
					},
				},
				{
					"name":        "generate_remediation",
					"description": "Generates a verified, ready-to-deploy OTel Collector YAML remediation patch for a given issue type.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"issue_type": map[string]interface{}{
								"type":        "string",
								"description": "The type of issue (e.g. High Cardinality (user_id on checkout_service))",
							},
						},
						"required": []string{"issue_type"},
					},
				},
			},
		}

	case "tools/call":
		var callArgs struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &callArgs); err != nil {
			rpcErr = &rpcError{
				Code:    -32602,
				Message: "Invalid params: " + err.Error(),
			}
			break
		}

		toolReq := ToolRequest{
			ToolName:  callArgs.Name,
			Arguments: callArgs.Arguments,
		}
		toolResp := s.HandleToolCall(ctx, toolReq)

		if !toolResp.Success {
			result = map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": fmt.Sprintf("Error: %s", toolResp.Error),
					},
				},
				"isError": true,
			}
		} else {
			result = map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": toolResp.Data,
					},
				},
			}
		}

	default:
		rpcErr = &rpcError{
			Code:    -32601,
			Message: "Method not found: " + req.Method,
		}
	}

	if isNotification {
		return nil, nil
	}

	resp := jsonrpcResponse{
		JsonRPC: "2.0",
		ID:      req.ID,
	}
	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		resp.Result = result
	}

	return json.Marshal(resp)
}
