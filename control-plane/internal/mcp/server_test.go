package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestHandleJSONRPCMessage(t *testing.T) {
	logger := zap.NewNop()
	toolset := NewToolset(nil, nil, nil, logger)
	server := NewServer(toolset)
	ctx := context.Background()

	t.Run("initialize", func(t *testing.T) {
		req := `{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}`
		respBytes, err := server.HandleJSONRPCMessage(ctx, []byte(req))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp jsonrpcResponse
		if err := json.Unmarshal(respBytes, &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.JsonRPC != "2.0" {
			t.Errorf("expected jsonrpc 2.0, got %s", resp.JsonRPC)
		}
		if resp.ID.(float64) != 1 {
			t.Errorf("expected ID 1, got %v", resp.ID)
		}
		if resp.Error != nil {
			t.Errorf("expected no error, got %v", resp.Error)
		}

		resultMap, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map result, got %T", resp.Result)
		}
		if resultMap["protocolVersion"] != "2024-11-05" {
			t.Errorf("expected protocolVersion 2024-11-05, got %v", resultMap["protocolVersion"])
		}
	})

	t.Run("notifications/initialized", func(t *testing.T) {
		req := `{"jsonrpc": "2.0", "method": "notifications/initialized"}`
		respBytes, err := server.HandleJSONRPCMessage(ctx, []byte(req))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if respBytes != nil {
			t.Errorf("expected nil response for notification, got %s", string(respBytes))
		}
	})

	t.Run("tools/list", func(t *testing.T) {
		req := `{"jsonrpc": "2.0", "id": 2, "method": "tools/list"}`
		respBytes, err := server.HandleJSONRPCMessage(ctx, []byte(req))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp jsonrpcResponse
		if err := json.Unmarshal(respBytes, &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Error != nil {
			t.Errorf("expected no error, got %v", resp.Error)
		}

		resultMap, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map result, got %T", resp.Result)
		}
		toolsList, ok := resultMap["tools"].([]interface{})
		if !ok {
			t.Fatalf("expected tools list, got %T", resultMap["tools"])
		}
		if len(toolsList) != 2 {
			t.Errorf("expected 2 tools, got %d", len(toolsList))
		}
	})

	t.Run("tools/call get_telemetry_health no repo", func(t *testing.T) {
		req := `{
			"jsonrpc": "2.0",
			"id": 3,
			"method": "tools/call",
			"params": {
				"name": "get_telemetry_health",
				"arguments": {
					"tenant_id": "tenant-123"
				}
			}
		}`
		respBytes, err := server.HandleJSONRPCMessage(ctx, []byte(req))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp jsonrpcResponse
		if err := json.Unmarshal(respBytes, &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		resultMap, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map result, got %T", resp.Result)
		}

		isErrorVal, exists := resultMap["isError"]
		if !exists || isErrorVal != true {
			t.Errorf("expected isError to be true, got %v", isErrorVal)
		}

		contentList, ok := resultMap["content"].([]interface{})
		if !ok || len(contentList) == 0 {
			t.Fatalf("expected content list, got %T", resultMap["content"])
		}
		contentMap := contentList[0].(map[string]interface{})
		textVal := contentMap["text"].(string)
		if !strings.Contains(textVal, "health repository not configured") {
			t.Errorf("expected repository not configured error, got: %s", textVal)
		}
	})

	t.Run("parse error", func(t *testing.T) {
		req := `{invalid-json}`
		respBytes, err := server.HandleJSONRPCMessage(ctx, []byte(req))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp jsonrpcResponse
		if err := json.Unmarshal(respBytes, &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Error == nil {
			t.Fatal("expected error, got nil")
		}
		if resp.Error.Code != -32700 {
			t.Errorf("expected error code -32700, got %d", resp.Error.Code)
		}
	})

	t.Run("method not found", func(t *testing.T) {
		req := `{"jsonrpc": "2.0", "id": 4, "method": "invalid_method"}`
		respBytes, err := server.HandleJSONRPCMessage(ctx, []byte(req))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp jsonrpcResponse
		if err := json.Unmarshal(respBytes, &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Error == nil {
			t.Fatal("expected error, got nil")
		}
		if resp.Error.Code != -32601 {
			t.Errorf("expected error code -32601, got %d", resp.Error.Code)
		}
	})
}
