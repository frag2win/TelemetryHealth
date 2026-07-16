# TelemetryHealth Reality Check

**Date:** 2026-07-17
**Perspective:** Senior Distributed Systems Engineer Review
**Methodology:** Zero-trust code analysis. Documentation, PRDs, architecture diagrams, and comments were ignored. Only statically verifiable source code and execution call graphs were evaluated.

---

## Subsystem Execution Matrix

| Subsystem | Exists | Compiles | Executes | Reachable | Tested | Complete |
| :--- | :---: | :---: | :---: | :---: | :---: | :--- |
| **REST API Server** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ (Serves mocks for complex data) |
| **Dashboard (React)** | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ (Relies on mock API responses) |
| **ClickHouse Storage** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ (Violates separation of concerns) |
| **OTel gRPC Ingestion** | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ |
| **Remediation Generator**| ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Kafka Stream Workers** | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ (Missing alert wiring) |
| **AI Behavior Engine** | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ (Dead code, bypassed by API) |
| **AI Decision Engine** | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ (Dead code, bypassed by API) |
| **AI Root Cause Engine**| ✅ | ✅ | ❌ | ❌ | ✅ | ❌ (Dead code, bypassed by API) |
| **MCP Server** | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ (No transport or JSON-RPC) |
| **SigNoz Query API** | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ (Bypasses API, uses direct SQL) |
| **Slack / PagerDuty** | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ (HTTP clients never instantiated) |
| **SigNoz Alert Bridge** | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ (Fake implementation, logs only) |

---

## 1. Working
*These components actually do what the code says they do.*
*   **API Routing:** `api/rest/server.go` correctly wires chi middleware (CORS, JWT validation) and handles basic HTTP semantics.
*   **Basic Storage:** `client.go` establishes real ClickHouse connections. `HealthRepository` executes real SQL against custom `telemetry_health.*` tables.
*   **Ingestion:** `cmd/ingest-gateway` uses actual `go.opentelemetry.io/collector/pdata` receivers to accept OTLP traffic.
*   **Remediation Templates:** `internal/remediation/generator.go` genuinely evaluates Go text/templates.

## 2. Partially Working
*These execute but fall short of structural integrity or correctness.*
*   **Kafka Processing:** The `worker` loop consumes sarama messages successfully, but the pipeline ends there. It never triggers the alerting system.
*   **Trace Queries:** `QueryAgentTraces` runs real SQL, but against an unstable internal SigNoz table (`signoz_index_v2`) rather than an API, and instantly falls back to a hardcoded `[SIMULATED]` mock array if the DB is empty.

## 3. Broken
*Code that exists in the critical path but cannot function.*
*   **MCP Server (`internal/mcp/server.go`):** It is a Go struct with a `switch` statement. It lacks standard I/O readers, HTTP listeners, WebSocket bindings, or JSON-RPC 2.0 framing. An external MCP client cannot establish a connection to it.

## 4. Dead Code
*Fully implemented, well-tested logic that is mathematically impossible to reach at runtime.*
*   **The "Intelligence" Core:** `internal/behavior`, `internal/decision`, and `internal/rootcause` represent the core hackathon innovation. They have complex logic and tests. However, the REST handlers bypass them entirely, opting to return static JSON strings to the frontend.
*   **The Alert Transports:** `SlackBridge` and `PagerDutyBridge` contain real HTTP POST logic, but they are never instantiated in `cmd/worker/main.go` or wired into the stream processor.

## 5. Fake Completion Claims
*Code that actively deceives the reader.*
*   **SigNoz Alertmanager:** `SigNozBridge.FireAlert()` contains no `net/http` client. It simply prints `logger.Info("Firing alert to SigNoz")` and returns `nil`.
*   **SigNoz Dashboard JSON:** Claimed as "native dashboards," these files use incompatible schemas and cannot actually be imported into a running SigNoz instance.
*   **MCP Client:** `mcp.QueryAgentTraces` constructs a SQL string, throws it away (`_ = query`), ignores the `serverURL` argument entirely, and returns `[SIMULATED] t1`.
*   **OTel SDK:** `telemetry.InitOTelSDK` claims to initialize a global tracer, but the actual exporter initialization code is commented out.

## 6. Architecture Drift
*Where the code violates standard distributed systems principles.*
*   **The God Object:** `clickhouse.HealthRepository` (555 lines) handles connection pooling, raw span querying, AI inference scoring, cardinality math, mock data generation, and acts as the REST API's direct dependency. There are no domain interfaces.
*   **Leaky DTOs:** The REST server imports `mcp.HealthResponse` to use as its HTTP JSON response shape, tightly coupling the HTTP API to the unimplemented MCP tool schemas.

## 7. Highest Risk Components
*If deployed, these will cause immediate failure.*
*   **`QueryAgentTraces` SQL Bypass:** Querying `signoz_traces.signoz_index_v2` bypasses all API rate limits, RBAC, and data governance. This table schema is internal to SigNoz; any upstream version bump will break TelemetryHealth silently.
*   **Auth Bypass:** `rest/server.go` contains a magic string `tenantID != "acme-prod"` allowing bypassing of strict UUID validation.

---

## 8. Top 20 Fixes Before Hackathon

If this repository is audited by judges, they will discover the deception. Execute these fixes immediately to transform the vaporware into a functioning system.

### Intelligence & Core Value
1.  **Wire up `internal/behavior`:** Modify `GetBehaviorGraph` in `rest/server.go` to invoke `engine.ReconstructBehavior()` instead of returning static strings.
2.  **Wire up `internal/decision`:** Modify `GetDecisionGraph` to invoke `engine.ReconstructDecision()`.
3.  **Wire up `internal/rootcause`:** Modify `GetRootCause` to invoke `engine.AnalyzeRootCause()`.
4.  **Remove Trace Mocks:** Delete the `[SIMULATED]` fallback block in `health_repository.go:183`.

### SigNoz Integration (Actual)
5.  **Use the API:** Replace the `signoz_index_v2` raw SQL query with a `net/http` call to the SigNoz Query Service API (`/api/v3/query_range`).
6.  **Real Alert Bridge:** Add an `*http.Client` to `SigNozBridge` and construct an actual Alertmanager webhook POST request.
7.  **Fix Dashboard Schemas:** Update the JSON files in `dashboard/signoz/` to match the official SigNoz `v1/dashboards` export format (adding `uuid`, `layout`, `panelMap`).
8.  **Fix Missing Metrics:** The dashboard JSON references `telemetryhealth_agent_health_score`, but `metrics.go` actually exports `telemetryhealth_pipeline_health_score`. Align these names.

### MCP Server
9.  **Build Transport:** Implement an `os.Stdin` / `os.Stdout` scanner loop in `cmd/api-server/main.go` to serve the MCP protocol.
10. **Implement JSON-RPC:** Replace `mcp.ToolRequest` with a compliant JSON-RPC 2.0 envelope parser.
11. **Tool Discovery:** Implement the `initialize` handshake and `tools/list` manifest handlers.
12. **Real Client Integration:** Replace the mocked `[SIMULATED]` return in `mcp/client.go` by integrating a real MCP SDK (e.g., `mark3labs/mcp-go`).

### Stream Processing & Alerting
13. **Instantiate Bridges:** In `cmd/worker/main.go`, instantiate `SlackBridge` and `PagerDutyBridge`.
14. **Wire Alerts:** In `workers.go`, add logic inside the Kafka consumer loop to invoke `bridge.FireAlert()` when scores drop below thresholds.

### Technical Debt & Architecture
15. **Break up the God Object:** Extract the cardinality math and health scoring logic out of `health_repository.go` into a new `internal/domain/health.go` service.
16. **Create Interfaces:** Define a `type Repository interface` in the REST package so `server.go` doesn't depend on the `clickhouse` package directly.
17. **Remove Magic Strings:** Delete the `acme-prod` UUID bypass in `server.go`.
18. **Uncomment OTel:** In `telemetry/otel.go`, uncomment the `otlptracegrpc.New` block so the global tracer actually exports data.
19. **DRY Remediation:** Delete the duplicate `Generator.Generate` invocation block in `mcp/tools.go` and have it call a unified domain service instead.
20. **Test Ingestion:** Write at least one unit test for `grpc_server.go` to prove the `ptraceotlp` unmarshaling functions correctly.
