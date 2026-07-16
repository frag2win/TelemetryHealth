# TelemetryHealth — MCP Implementation Production Audit

**Date:** 2026-07-17
**Method:** Traced every import, call site, route registration, transport binding, and protocol compliance element across the entire `internal/mcp/` package, all `cmd/*/main.go` entrypoints, and the REST server route table.

---

## Executive Verdict

> [!CAUTION]
> **No external MCP client can connect to this implementation.** The MCP "server" has no transport layer, no startup path, no protocol handler, no route registration, and no running process. It is a Go struct with two methods that are never called by anything at runtime.

---

## Audit Matrix: 8 Production Requirements

| # | Requirement | Status | Evidence |
|---|---|---|---|
| 1 | **Transport** | ❌ MISSING | No stdio, HTTP/SSE, or WebSocket transport exists |
| 2 | **Startup** | ❌ MISSING | No `main()` or goroutine starts the MCP server |
| 3 | **Registration** | ❌ MISSING | No tool manifest, no `tools/list` handler |
| 4 | **Discovery** | ❌ MISSING | No `initialize`/`initialized` handshake |
| 5 | **Tool Execution** | ⚠️ PARTIAL | Logic exists but is unreachable |
| 6 | **Request Routing** | ⚠️ PARTIAL | `switch` on `tool_name` exists but uses non-MCP schema |
| 7 | **Response Serialization** | ❌ WRONG | Uses custom `ToolResponse`, not JSON-RPC 2.0 |
| 8 | **Error Handling** | ⚠️ PARTIAL | Application errors handled, but no protocol-level errors |

---

## Detailed Analysis

---

### 1. Transport — ❌ MISSING

**The MCP specification defines two transport mechanisms:**
- **stdio** (standard I/O): Process reads JSON-RPC from `stdin`, writes to `stdout`
- **Streamable HTTP** (HTTP + SSE): Server exposes an HTTP endpoint that accepts JSON-RPC over POST and returns Server-Sent Events

**What exists in the codebase:**

```
internal/mcp/
├── server.go    — 66 lines, no transport
├── client.go    — 44 lines, no transport
└── tools.go     — 104 lines, no transport
```

**Searched for and NOT found:**

| Transport Element | Search Pattern | Result |
|---|---|---|
| stdio reader | `os.Stdin`, `bufio.Scanner`, `bufio.Reader` | Not found |
| stdio writer | `os.Stdout`, `bufio.Writer` | Not found |
| HTTP listener | `http.ListenAndServe`, `http.Server` | Not found in `mcp/` |
| SSE handler | `text/event-stream`, `server-sent`, `SSE` | Not found |
| WebSocket | `websocket`, `gorilla/websocket` | Not found |
| JSON-RPC framing | `jsonrpc`, `json-rpc`, `Content-Length` | Not found |
| Any `net` import | `net`, `net/http` (as server) | Not found in `server.go` |

**Imports of `server.go`:**
```go
import (
    "context"
    "encoding/json"
    "fmt"
)
```

Three imports. No networking. No I/O. The `Server` struct has no way to receive bytes from any external source.

---

### 2. Startup — ❌ MISSING

**For a production MCP server to function, something must:**
1. Instantiate the `Server` struct
2. Bind it to a transport
3. Start a listen/read loop

**Evidence that none of this happens:**

| Check | Result |
|---|---|
| `mcp.NewServer()` called anywhere? | **No.** Zero call sites across entire codebase |
| `mcp.NewToolset()` called anywhere? | **No.** Zero call sites across entire codebase |
| `HandleToolCall()` called anywhere? | **No.** Only defined, never invoked |
| `internal/mcp` imported by any `cmd/*/main.go`? | **No.** Zero imports in any executable entrypoint |
| `internal/mcp` imported by anything? | **Only** by `rest/server.go` — and only for DTO types (`mcp.HealthResponse`, `mcp.MetricsPayload`, etc.) |

**The REST server uses `mcp` as a DTO package, not as an MCP server.**

At [server.go L389](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/api/rest/server.go#L389):
```go
resp := mcp.HealthResponse{          // ← using MCP types as REST DTOs
    HealthScore: metrics.CompositeScore,
    Metrics: mcp.MetricsPayload{...},
    Remediation: mcp.RemediationPayload{...},
}
json.NewEncoder(w).Encode(resp)       // ← served over REST/HTTP, not MCP
```

The MCP package is a **struct library** being borrowed by the REST handler for its JSON response shapes. The `mcp.Server`, `mcp.ToolRequest`, and `mcp.HandleToolCall` are never used.

---

### 3. Registration — ❌ MISSING

**The MCP spec requires servers to respond to `tools/list` with a tool manifest describing available tools, their input schemas, and descriptions.**

**What should exist:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "get_telemetry_health",
        "description": "Returns composite health score and metrics for a tenant",
        "inputSchema": {
          "type": "object",
          "properties": {
            "tenant_id": { "type": "string", "description": "Tenant UUID" }
          },
          "required": ["tenant_id"]
        }
      }
    ]
  }
}
```

**What actually exists:** Nothing. No tool manifest. No input schema definition. No `tools/list` handler. No tool descriptions. An MCP client performing discovery would receive no response.

---

### 4. Discovery — ❌ MISSING

**The MCP spec requires an initialization handshake:**

```
Client → Server:  initialize (with client capabilities)
Server → Client:  initialize result (with server capabilities, protocol version)
Client → Server:  initialized (notification)
```

**What should exist:**
- Handler for `initialize` method
- `protocolVersion` field (e.g., `"2025-03-26"`)
- `capabilities` object declaring `tools` support
- Handler for `initialized` notification

**What actually exists:** None of the above. Zero references to `initialize`, `capabilities`, `protocolVersion`, or any MCP lifecycle method anywhere in the codebase.

---

### 5. Tool Execution — ⚠️ PARTIAL (Logic exists, unreachable)

**File:** [server.go L34-64](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/mcp/server.go#L34-L64)

```go
func (s *Server) HandleToolCall(ctx context.Context, req ToolRequest) ToolResponse {
    switch req.ToolName {
    case "get_telemetry_health":
        // ... unmarshal args, call toolset, return response
    case "generate_remediation":
        // ... unmarshal args, call toolset, return response
    default:
        return ToolResponse{Success: false, Error: "unsupported MCP tool: ..."}
    }
}
```

**Assessment:**
- ✅ Two tools are implemented with real business logic
- ✅ Argument deserialization from `json.RawMessage`
- ✅ Error propagation from downstream services
- ❌ **Never called.** Zero call sites exist. `HandleToolCall` is an exported method with no caller.
- ❌ Uses `ToolRequest` (custom struct), not the MCP spec's `tools/call` JSON-RPC method
- ❌ Returns `ToolResponse` (custom struct), not JSON-RPC 2.0 result

---

### 6. Request Routing — ⚠️ PARTIAL (Wrong protocol)

**MCP spec requires:** JSON-RPC 2.0 messages with `method` field routing:
```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "get_telemetry_health", "arguments": {"tenant_id": "..."}}}
```

**What exists:**
```go
type ToolRequest struct {
    ToolName  string          `json:"tool_name"`    // ← not "name"
    Arguments json.RawMessage `json:"arguments"`
}
```

| MCP Spec Field | Implementation Field | Match? |
|---|---|---|
| `method: "tools/call"` | Not implemented | ❌ |
| `params.name` | `tool_name` | ❌ Wrong field name |
| `params.arguments` | `arguments` | ✅ |
| `jsonrpc: "2.0"` | Not present | ❌ |
| `id` (request ID) | Not present | ❌ |

The routing uses a Go-level `switch` on a struct field, not a JSON-RPC method dispatcher.

---

### 7. Response Serialization — ❌ WRONG

**MCP spec requires JSON-RPC 2.0 responses:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [{"type": "text", "text": "..."}],
    "isError": false
  }
}
```

**What exists:**
```go
type ToolResponse struct {
    Success bool   `json:"success"`       // ← not "isError" or "content"
    Data    string `json:"data,omitempty"` // ← not content array
    Error   string `json:"error,omitempty"`
}
```

| MCP Spec Field | Implementation | Match? |
|---|---|---|
| `jsonrpc: "2.0"` | Not present | ❌ |
| `id` (echo request ID) | Not present | ❌ |
| `result.content` (array of content blocks) | `Data string` | ❌ |
| `result.isError` | `Success bool` (inverted semantics) | ❌ |
| Error responses with `error.code`, `error.message` | `Error string` | ❌ |

Additionally, the `get_telemetry_health` tool serializes its response with `fmt.Sprintf`:
```go
return ToolResponse{Success: true, Data: fmt.Sprintf(`{"health_score": %f}`, resp.HealthScore)}
```

This creates a JSON string **inside** a JSON field — double-encoded JSON. An MCP client would receive:
```json
{"success": true, "data": "{\"health_score\": 85.300000}"}
```

The `health_score` value uses `%f` (6 decimal places by default), producing `85.300000` instead of a clean numeric.

---

### 8. Error Handling — ⚠️ PARTIAL

**Application-level errors are handled:**
```go
if err := json.Unmarshal(req.Arguments, &args); err != nil {
    return ToolResponse{Success: false, Error: err.Error()}
}
```

**What's missing:**
- No JSON-RPC error codes (`-32700` Parse Error, `-32600` Invalid Request, `-32601` Method Not Found, `-32602` Invalid Params)
- No protocol-level error handling (malformed JSON-RPC envelope, missing `id`, wrong `jsonrpc` version)
- No timeout handling
- No request cancellation via `notifications/cancelled`
- No logging of tool execution failures (the `Toolset` has a `Logger` but `Server` does not)

---

## The Missing Execution Path

Here is the complete path that would need to exist for an external MCP client to connect and call a tool. Every ❌ step is completely absent from the codebase.

```
External MCP Client (e.g., Claude Desktop, Cursor, VS Code)
    │
    ▼
❌ Transport Layer (stdio read loop OR HTTP listener)
    │  server.go has no stdin reader, no http.Server, no SSE handler
    │  No net.Listener, no bufio.Scanner, no goroutine
    │
    ▼
❌ JSON-RPC 2.0 Frame Parser
    │  No jsonrpc library imported
    │  No Content-Length header parsing (stdio)
    │  No POST body parsing (HTTP)
    │
    ▼
❌ Method Dispatcher
    │  No routing for "initialize", "tools/list", "tools/call"
    │  No notification handling ("notifications/cancelled", "initialized")
    │
    ▼
❌ initialize / initialized Handshake
    │  No protocol version negotiation
    │  No capabilities exchange
    │
    ▼
❌ tools/list Handler
    │  No tool manifest
    │  No input schema definitions
    │  No tool descriptions
    │
    ▼
❌ tools/call Dispatcher
    │  No extraction of params.name and params.arguments
    │  No mapping to HandleToolCall
    │
    ▼
⚠️ HandleToolCall (EXISTS but unreachable)
    │  switch req.ToolName {
    │    case "get_telemetry_health": ...  ← logic exists
    │    case "generate_remediation": ...  ← logic exists
    │  }
    │
    ▼
⚠️ Toolset Business Logic (EXISTS but unreachable)
    │  GetTelemetryHealth → ClickHouse query  ← logic exists
    │  GenerateRemediation → template render  ← logic exists
    │
    ▼
❌ JSON-RPC 2.0 Response Serializer
    │  No {"jsonrpc":"2.0","id":...,"result":{...}} envelope
    │  No content block array format
    │
    ▼
❌ Transport Writer
    │  No stdout writer (stdio)
    │  No HTTP response writer (streamable HTTP)
    │  No SSE event emitter
    │
    ▼
  ∅  Client receives nothing
```

**9 steps are required. 7 are completely missing. 2 exist but are unreachable.**

---

## The MCP Client (`client.go`) Is Equally Non-Functional

**File:** [client.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/mcp/client.go)

| Function | What It Claims | What It Does |
|---|---|---|
| `InjectTraceContext` | Inject OTel context into outgoing requests | ✅ Works, but never called at runtime (only tested in `server_test.go`) |
| `QueryAgentTraces` | "queries the SigNoz MCP server" | Returns hardcoded `[SIMULATED]` data. Ignores `serverURL`. Never called by any runtime path. |

```go
func QueryAgentTraces(ctx context.Context, tenantID string, serverURL string) (*Traces, error) {
    query := `SELECT * FROM traces WHERE attributes['service.name'] = 'ai-agent'`
    _ = query // keep compiler happy       ← DISCARDS the query
    return &Traces{
        Count: 2,
        Data: []map[string]interface{}{
            {"trace_id": "[SIMULATED] t1", ...},    ← HARDCODED
        },
    }, nil
}
```

The code even contains the aspirational comment:
```go
// In a real implementation:
// import "github.com/signoz/mcp-go"
// mcpClient := mcp.NewClient(serverURL)
// return mcpClient.Query(ctx, query)
```

`github.com/signoz/mcp-go` does not exist as a public Go package.

---

## What Production MCP Would Require

For reference, here is what a minimal working Go MCP server looks like (using `github.com/mark3labs/mcp-go` or similar):

```go
package main

import (
    "context"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

func main() {
    // 1. Create server with capabilities
    s := server.NewMCPServer("TelemetryHealth", "1.0.0",
        server.WithToolCapabilities(true),
    )

    // 2. Register tools with schemas
    s.AddTool(mcp.NewTool("get_telemetry_health",
        mcp.WithDescription("Get health score for a tenant"),
        mcp.WithString("tenant_id", mcp.Required(), mcp.Description("Tenant UUID")),
    ), handleGetHealth)

    // 3. Start transport (stdio)
    if err := server.ServeStdio(s); err != nil {
        log.Fatal(err)
    }
}
```

**Comparing with the current implementation:**

| Required Component | Production MCP | Current Code |
|---|---|---|
| MCP SDK dependency | `github.com/mark3labs/mcp-go` | ❌ None |
| Server constructor | `server.NewMCPServer(name, version)` | `mcp.NewServer(toolset)` — no name, no version |
| Tool registration | `s.AddTool(schema, handler)` | ❌ None |
| Input schema | `mcp.WithString("tenant_id", Required())` | ❌ None |
| Transport startup | `server.ServeStdio(s)` | ❌ None |
| JSON-RPC handling | Handled by SDK | ❌ None |
| Lifecycle (init) | Handled by SDK | ❌ None |
| Total lines needed | ~30 lines | 0 lines exist |

---

## Test Coverage

| Test Category | Exists? |
|---|---|
| Unit tests for `mcp.Server` | ❌ No |
| Unit tests for `mcp.Toolset` | ❌ No |
| Unit tests for `mcp.HandleToolCall` | ❌ No |
| Integration test (client → server) | ❌ No |
| Any test file in `internal/mcp/` | ❌ No |
| Any test referencing `mcp` package | ❌ No (only `TestInjectTraceContext` in `rest/server_test.go`) |

---

## Conclusion

The `internal/mcp/` package is **not an MCP implementation**. It is:

1. **A DTO library** — `HealthResponse`, `MetricsPayload`, `RemediationPayload` are used by the REST handler as JSON response shapes
2. **A dead internal API** — `Server.HandleToolCall` implements application logic for two tools but has zero callers, zero transport, and zero protocol compliance
3. **An aspirational stub** — `client.go` contains comments describing what a real implementation would look like and returns simulated data

**An external MCP client cannot connect because:**
- No process listens for MCP connections (no transport)
- No process advertises MCP capabilities (no discovery)
- No tool manifest exists (no registration)
- No JSON-RPC 2.0 protocol handling exists (no framing)
- The 7 missing components in the execution path would require approximately 200-400 lines of new code plus an MCP SDK dependency to implement
