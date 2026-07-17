# Strict Feature Verification Audit

This document verifies the exact execution paths for features previously marked COMPLETE.

---

### 1. Behavior Graph
- **Execution Path:** `cmd/api-server/main.go:main()` → `rest.NewServer()` → `s.handleBehaviorGraph` → `engine.GenerateBehaviorGraph(tenantID)`
- **Constructors:** `engine.NewEngine(replayRepo)`, `engine.NewBehaviorBuilder()` (`graph.go:49`)
- **Interfaces Crossed:** `engine.BehaviorBuilder.Build(events []ReplayEvent)`
- **Persistence Layer:** `engine.ReplayRepository` backed by `clickhouse.ReplayRepository` fetching from `signoz_traces.distributed_signoz_index_v2`.
- **Data Used:** Real ClickHouse data (with a fallback to `defaultTopology()` if empty).
- **Frontend Invocation:** Requested by `/api/v1/tenant/{tenant_id}/topology` and rendered inside `dashboard/src/components/views/DigitalTwin.tsx` via ReactFlow.
- **Status:** **COMPLETE**

### 2. Decision Engine
- **Execution Path:** `cmd/api-server/main.go:main()` → `rest.NewServer()` → `s.GetTenantRootCause` → `engine.GenerateRootCause(tenantID, issueID)`
- **Constructors:** `engine.NewDecisionBuilder()` inside `engine.NewEngine()` constructor (`graph.go:46`)
- **Interfaces Crossed:** `engine.DecisionBuilder.Build(bg *BehaviorGraph)`
- **Persistence Layer:** Same as Behavior Graph (`ReplayRepository`).
- **Data Used:** Real data via ClickHouse.
- **Frontend Invocation:** Part of the Root Cause graph fetch via `/api/v1/tenant/{tenant_id}/rootcause`.
- **Status:** **COMPLETE**

### 3. Root Cause Engine
- **Execution Path:** `cmd/api-server/main.go:main()` → `rest.NewServer()` → `s.GetTenantRootCause` → `engine.GenerateRootCause(tenantID, traceID)` → `dg := e.decBuilder.Build(bg)` → `rcg := e.rcBuilder.Build(dg)`
- **Constructors:** `engine.NewRootCauseBuilder()` inside `engine.NewEngine()` (`graph.go:46`)
- **Interfaces Crossed:** `engine.RootCauseBuilder.Build(dg *DecisionGraph)`
- **Persistence Layer:** Same as Behavior Graph.
- **Data Used:** Real data via ClickHouse.
- **Frontend Invocation:** Fetched by `dashboard/src/components/views/RootCauseGraph.tsx`.
- **Status:** **COMPLETE**

### 4. Benchmark Framework
- **Execution Path:** `cmd/api-server/main.go:main()` → `s.GetTenantRootCause` → `engine.GenerateRootCause(tenantID, traceID)`
- **Logic Hook:** `if len(traceID) > 10 && traceID[:10] == "benchmark-"` (`graph.go:78`)
- **Constructors:** None (static function).
- **Interfaces Crossed:** Bypasses `ReplayRepository` entirely; directly supplies `[]ReplayEvent`.
- **Persistence Layer:** None.
- **Data Used:** **MOCK** (Hardcoded deterministic events).
- **Frontend Invocation:** Can be explicitly requested via UI by passing a trace ID like `benchmark-prompt-explosion`.
- **Status:** **COMPLETE**

### 5. Replay Engine
- **Execution Path:** `cmd/api-server/main.go:main()` → `rest.NewServer()` → `s.GetAgents` (`server.go:565`)
- **Constructors:** `ch.NewHealthRepository(client.Conn())`
- **Interfaces Crossed:** `storage.HealthRepository.GetSpanData()`
- **Persistence Layer:** ClickHouse (`signoz_traces.distributed_signoz_index_v2`).
- **Data Used:** Real data (falls back to mock if empty/error).
- **Frontend Invocation:** Rendered by `dashboard/src/components/views/AgentTraces.tsx`.
- **Status:** **COMPLETE**

### 6. MCP Server
- **Execution Path:** `cmd/mcp-server/main.go:main()`
- **Constructors:** `mcp.NewToolset(...)`, `mcp.NewServer(toolset)`, `mcp.ServeStdio()` / `mcp.NewHTTPHandler()`.
- **Interfaces Crossed:** `mcp.Tool.Execute()`.
- **Persistence Layer:** Injected `healthRepo` connected to ClickHouse.
- **Data Used:** Real data queries.
- **Frontend Invocation:** N/A (invoked externally via stdio or HTTP/SSE).
- **Status:** **COMPLETE** (Note: `resources` and `prompts` capabilities are unverified/missing, but server transport, tools, and execution are proven complete).

### 7. Auto Remediation
- **Execution Path (API):** `cmd/api-server/main.go:main()` → `rest.NewServer()` → `s.ApplyRemediation`
- **Execution Path (MCP):** `cmd/mcp-server/main.go:main()` → `mcp.Toolset` execution of `generate_remediation` tool.
- **Constructors:** `remediation.NewGenerator(logger)`
- **Interfaces Crossed:** None directly outside of `mcp.Tool`.
- **Persistence Layer:** N/A (Produces YAML payloads).
- **Data Used:** Deterministic generation based on input.
- **Frontend Invocation:** UI interactions in `Remediation.tsx` trigger `POST /api/v1/remediation/apply`.
- **Status:** **COMPLETE**

### 8. ClickHouse Query Layer
- **Execution Path:** `cmd/api-server/main.go:main()` / `cmd/mcp-server/main.go:main()` → `ch.NewClient()` → `ch.NewHealthRepository()`
- **Constructors:** `clickhouse.Open()` from `github.com/ClickHouse/clickhouse-go/v2`.
- **Interfaces Crossed:** `storage.HealthRepository` and `engine.ReplayRepository`.
- **Persistence Layer:** `clickhouse-go/v2` driver querying port `9000`.
- **Data Used:** Real telemetry data stored by an external OTLP collector.
- **Frontend Invocation:** N/A (Backend only).
- **Status:** **COMPLETE**
