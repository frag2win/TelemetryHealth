package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestServeStdio(t *testing.T) {
	logger := zap.NewNop()
	toolset := NewToolset(nil, nil, nil, logger)
	server := NewServer(toolset)

	input := strings.Join([]string{
		`{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}`,
		`{"jsonrpc": "2.0", "method": "notifications/initialized"}`,
		`{"jsonrpc": "2.0", "id": 2, "method": "tools/list"}`,
	}, "\n") + "\n"

	reader := strings.NewReader(input)
	var writer bytes.Buffer
	ctx := context.Background()

	err := ServeStdio(ctx, reader, &writer, server, logger)
	if err != nil {
		t.Fatalf("ServeStdio failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(writer.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 response lines (notification should have no response), got %d: %v", len(lines), lines)
	}

	var initResp jsonrpcResponse
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("failed to unmarshal init response: %v", err)
	}
	if initResp.ID.(float64) != 1 {
		t.Errorf("expected ID 1 for init response, got %v", initResp.ID)
	}

	var listResp jsonrpcResponse
	if err := json.Unmarshal([]byte(lines[1]), &listResp); err != nil {
		t.Fatalf("failed to unmarshal list response: %v", err)
	}
	if listResp.ID.(float64) != 2 {
		t.Errorf("expected ID 2 for list response, got %v", listResp.ID)
	}
}

func TestHTTPHandlerDirectAndSSE(t *testing.T) {
	logger := zap.NewNop()
	toolset := NewToolset(nil, nil, nil, logger)
	server := NewServer(toolset)

	handler := NewHTTPHandler(server, logger)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Run("healthz", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/healthz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("direct HTTP JSON-RPC POST", func(t *testing.T) {
		reqBody := `{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}`
		resp, err := http.Post(ts.URL+"/jsonrpc", "application/json", strings.NewReader(reqBody))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var rpcResp jsonrpcResponse
		if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if rpcResp.ID.(float64) != 1 {
			t.Errorf("expected ID 1, got %v", rpcResp.ID)
		}
	})

	t.Run("SSE session and message delivery", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/sse", nil)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK for /sse, got %d", resp.StatusCode)
		}

		// Read first line/event from SSE to get session endpoint
		buf := make([]byte, 1024)
		n, err := resp.Body.Read(buf)
		if err != nil && n == 0 {
			t.Fatalf("failed to read SSE endpoint event: %v", err)
		}
		output := string(buf[:n])
		if !strings.Contains(output, "event: endpoint") || !strings.Contains(output, "data: /message?session=") {
			t.Fatalf("unexpected SSE endpoint handshake: %s", output)
		}

		// Extract session ID
		parts := strings.Split(output, "session=")
		if len(parts) < 2 {
			t.Fatalf("could not parse session id from: %s", output)
		}
		sessionID := strings.Split(parts[1], "\n")[0]

		// Post a message to the session endpoint
		msgBody := `{"jsonrpc": "2.0", "id": 42, "method": "tools/list"}`
		postResp, err := http.Post(ts.URL+"/message?session="+sessionID, "application/json", strings.NewReader(msgBody))
		if err != nil {
			t.Fatalf("failed to post message: %v", err)
		}
		defer postResp.Body.Close()
		if postResp.StatusCode != http.StatusAccepted {
			t.Errorf("expected 202 Accepted, got %d", postResp.StatusCode)
		}
	})
}
