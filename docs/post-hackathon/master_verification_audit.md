# TelemetryHealth Master Verification Audit (Final)

*This audit was conducted by strictly tracing actual execution paths, imports, test suites, and binaries. Documentation, PRDs, and comments were actively disregarded in favor of proven code reality.*

---

## Phase 1 — Build Verification

| Module | Build Command | Status | Evidence / Notes |
| :--- | :--- | :--- | :--- |
| **Frontend (React)** | `npm run build` | **PASS** | `vite build` completed cleanly, generating `dist/` with 1936 modules transformed. |
| **Backend (Go)** | `go build ./...` | **PASS** | All `cmd/` targets compile cleanly. |
| **Backend Tests** | `go test ./...` | **PASS** | Suite passes. Tests exist for MCP transport (`transport_test.go`), server (`server_test.go`), and API. |

---

## Phase 2 — Runtime Execution Graph

**Graph A: `cmd/api-server/main.go`**
```text
main()
 ↓
 authz.ValidateStartupConfig()
 ↓
 telemetry.InitOTelSDK("api-server")
 ↓
 ClickHouse Client (`ch.NewClient`)
 ↓
 storage.HealthRepository / storage.ReplayRepository
 ↓
 rest.NewServer(logger, healthRepo, replayRepo)
 ↓
 http.Server (port 8080)
```
**Conclusion:** `api-server` strictly serves REST. No OTLP ingest endpoints are active in this binary. `internal/kafka` is completely bypassed and is dead code in this runtime.

**Graph B: `cmd/mcp-server/main.go`**
```text
main()
 ↓
 ClickHouse Client
 ↓
 remediation.NewGenerator() -> remediation.NewValidator()
 ↓
 mcp.NewToolset(healthRepo, generator, validator)
 ↓
 mcp.NewServer()
 ↓
 [Branch] if --stdio: mcp.ServeStdio()
 [Branch] default: mcp.NewHTTPHandler() -> http.Server (port 8081)
```
**Conclusion:** MCP server is fully decoupled from the API server and runs its own transport.

---

## Phase 3 — Feature Truth Matrix

| Feature | Status | Evidence (Source File) |
| :--- | :--- | :--- |
| **ClickHouse Storage** | COMPLETE | `internal/storage/clickhouse/health_repository.go` executing queries against `signoz_traces`. |
| **Behavior / Decision Graph** | COMPLETE | `internal/engine/decision.go` and `rootcause.go` actively mapping states. |
| **Replay Engine UI** | COMPLETE | `dashboard/src/components/views/AgentTraces.tsx` (fully implemented with CSS animations). |
| **Auto Remediation** | COMPLETE | `internal/remediation/generator.go` |
| **Benchmark Framework** | COMPLETE | `internal/engine/benchmark.go` (TraceID hooking via `benchmark-` prefix). |
| **MCP Server (stdio / HTTP)** | COMPLETE | `internal/mcp/transport.go` & `cmd/mcp-server/main.go` |
| **Slack / PagerDuty / SigNoz Bridge** | STUB / DEAD | `internal/alerting/signoz_bridge.go` (Stubbed prints, not actively routed in main logic). |
| **Kafka Pipeline** | DEAD | `internal/kafka/producer.go` is completely unreachable from any `main.go`. |
| **OTLP Ingestion** | NOT IMPLEMENTED | Relies entirely on an external collector writing to ClickHouse. The platform only *reads* traces. |

---

## Phase 4 — Clean Architecture Verification

**Violation 1: Infrastructure Leakage in Presentation Layer**
- **Current:** `internal/api/rest/server.go` at Line 887 contains `generateMockSpans()`.
- **Why it violates:** The HTTP delivery mechanism contains hardcoded domain/mock logic, bypassing the storage repository abstraction.
- **Recommendation:** Move all mock generation into a `mock_repository.go` that implements the `ReplayRepository` interface.

---

## Phase 5 — Runtime Dependency Verification

| Integration | Classification | Note |
| :--- | :--- | :--- |
| **React / Vite** | Official SDK | Production-ready configuration. |
| **ClickHouse** | Raw Database | Directly queries DB bypassing OTel/SigNoz APIs. |
| **MCP** | Vendor Neutral | Adheres to standardized MCP JSON-RPC protocol. |
| **OpenTelemetry** | Official SDK | Used for self-instrumentation (`telemetry.InitOTelSDK`). |

---

## Phase 6 — MCP Compliance

| Capability | Status | Evidence |
| :--- | :--- | :--- |
| **stdio Transport** | Implemented | `mcp.ServeStdio` actively routing stdin/stdout. |
| **HTTP/SSE Transport** | Implemented | `mcp.NewHTTPHandler` routing SSE events. |
| **tools/list & tools/call** | Implemented | `mcp.Server` actively registering Toolsets. |
| **prompts / resources** | Missing | No endpoints or handlers exist for these capabilities. |

**Compatibility:** 
- **Cursor / VS Code:** CAN connect via `stdio` for tool execution.
- **Claude Desktop:** CAN connect via `stdio` for tool execution.
- *Cannot* pull resources or prompt templates as those endpoints are missing.

---

## Phase 7 — SigNoz Integration

- **Classification:** **Raw ClickHouse / Internal SigNoz Schema**
- **Evidence:** `health_repository.go` executes raw SQL: `SELECT ... FROM signoz_traces.distributed_signoz_index_v2`.
- **Portability:** This will **NOT** work with Jaeger, Tempo, or Datadog without heavy modification.
- **Risks:** High schema coupling risk. If SigNoz updates its indexing schema (e.g., to `v3`), this application breaks instantly.

---

## Phase 8 — Mock Detection

| File | Mock Snippet | Classification |
| :--- | :--- | :--- |
| `health_repository.go` | `Fallback to rich, realistic traces if database returned nothing` | Demo Fallback |
| `api/rest/server.go` | `ClickHouse unavailable, using mock trace data` | Demo Fallback |
| `engine/benchmark.go` | `GetBenchmarkScenario()` | Production Feature (Benchmark Testing) |

**Conclusion:** The platform *can* operate on real telemetry if the DB is populated, but heavily relies on demo fallbacks to guarantee the UI populates for hackathon judging.

---

## Phase 9 — Dead Code Audit

- `internal/kafka/` - **Dead**. Missing any constructor or injection in `cmd/`.
- `internal/alerting/` - **Prototype / Dead**. The `signoz_bridge` is reachable in theory but not wired to a live alerting pipeline.

---

## Phase 10 — Documentation Drift

- `smallfi.md` claims "MCP Server Transport... needs stdio/HTTP" -> **Outdated**. `cmd/mcp-server/main.go` clearly implements both.
- "Behavior Intelligence Engine (BIE)" PRD claims real-time Kafka stream processing -> **Overstated**. The pipeline is entirely poll-based REST over ClickHouse.

---

## Phase 11 — Repository Health

- **Package Coupling:** High. (Engines are coupled to specific clickhouse schema structures).
- **God Objects:** `server.go` is bloated (800+ lines) containing route definitions, handlers, and mock data generators.
- **Technical Debt:** High, due to bypassing standard interfaces for speed, but highly acceptable for a hackathon boundary.

---

## Phase 12 — Production Readiness

**Status: PROTOTYPE**
**Why:** While the architecture (Repository Pattern, Dependency Injection) is solid, the reliance on raw SQL tied to a specific third-party table schema (`signoz_traces.v2`), the lack of auth in the frontend, and the massive mock-fallbacks push this strictly into the "Prototype / Demo" category.

---

## Phase 13 — Hackathon Readiness

- **Presentation Risks:** **Low**. The heavy use of `generateMockSpans` ensures that the demo will function flawlessly even if the database crashes or is empty.
- **Judge-visible Issues:** **Low**. The UI (`AgentTraces.tsx`, `Remediation.tsx`) is heavily polished, functional, and visually striking.
- **Architecture Risks:** **Medium**. If judges inspect the source, they will see hardcoded mock arrays inside `server.go`. 

---

## Phase 14 — Final Reality Check

### Working
- React Flow Intelligence Pipelines (Behavior, Decision, Root Cause)
- MCP stdio / SSE server integration
- ClickHouse raw telemetry querying
- Remediation Patch UI

### Partially Working
- Role-based Access (Tenant isolation works, but JWTs are unverified structural parses).

### Mock-backed / Fallbacks
- Trace fetching (falls back to mock if empty).
- Benchmark traces.

### Dead / Not Implemented
- Kafka Ingestion.
- OTLP receiver (relying purely on external SigNoz collector).
- Automated Alert Triggering.

### Top Remaining Tasks (Post-Hackathon)
1. **Priority High:** Extract `generateMockSpans` from `server.go` into a `MockReplayRepository`. (1 hour)
2. **Priority High:** Refactor `health_repository.go` to use standardized OTEL APIs instead of raw SigNoz SQL tables to become vendor-neutral. (12 hours)
3. **Priority Medium:** Implement MCP `resources` and `prompts` capabilities. (4 hours)
