# TelemetryHealth — Repository Reachability Audit

**Date:** 2026-07-17
**Scope:** `control-plane/internal/` Package Ecosystem

---

## 1. Package Reachability Classification

Every package inside `internal/` has been classified according to direct call-graph analysis of runtime entrypoints (`cmd/`), test suites (`*_test.go`), and import matrices.

| Package | Classification | Reachability Path |
| :--- | :--- | :--- |
| **`internal/api`** | **Runtime Reachable** | `cmd/api-server/main.go` -> Starts the HTTP server router. |
| **`internal/authz`** | **Runtime Reachable** | `cmd/api-server/main.go` & `internal/ingest` -> Validates OIDC JWTs. |
| **`internal/behavior`** | **Runtime Reachable** | `rest/server.go` -> Reconstructs span behavior trees on-demand. |
| **`internal/decision`** | **Runtime Reachable** | `rest/server.go` -> Reconstructs agent decision paths on-demand. |
| **`internal/engine`** | **Runtime Reachable** | `rest/server.go` & `replay_repository.go` -> Shared engine structures. |
| **`internal/ingest`** | **Runtime Reachable** | `cmd/ingest-gateway/main.go` -> Exposes gRPC OTLP receiver ports. |
| **`internal/kafka`** | **Runtime Reachable** | `cmd/worker/main.go` & `cmd/ingest-gateway/main.go` -> Standard brokers. |
| **`internal/remediation`** | **Runtime Reachable** | `rest/server.go` -> Generates YAML remediation configurations. |
| **`internal/rootcause`** | **Runtime Reachable** | `rest/server.go` -> Analyzes trace root-cause profiles. |
| **`internal/simulator`** | **Runtime Reachable** | `cmd/simulator/main.go` -> Drives mock test-span injections. |
| **`internal/storage`** | **Runtime Reachable** | All executable binaries -> Raw database access layers. |
| **`internal/telemetry`** | **Runtime Reachable** | All binaries -> Registers Prometheus vectors and SDK hooks. |
| **`internal/streaming`** | **Test Reachable** | Bypassed at runtime. Only reached by `streaming/*_test.go` suites. |
| **`internal/mcp`** | **Library Only / Dead** | Structs are imported by `rest/server.go` as REST response types, but the MCP server and clients are unreachable. |
| **`internal/alerting`** | **Dead Code** | **Totally isolated.** No execution binary, test suite, or library imports this package. |

---

## 2. Gap Analysis for Unreachable Components

### 🚨 `internal/alerting` (Slack, PagerDuty, SigNoz Bridges)

* **Target Execution Path:** Needs to connect to the Kafka Stream Aggregator consumer loop in the worker binary (`cmd/worker/main.go` -> `workers.ProcessMessage`). When metrics drop below composite score thresholds, the loop should dispatch an alert payload.
* **Missing Constructor:** No package instantiates the bridges via `alerting.NewSlackBridge`, `alerting.NewPagerDutyBridge`, or `alerting.NewSigNozBridge`.
* **Missing Dependency Injection:** The alerting structures are not injected into the streaming worker lifecycle or handlers.
* **Missing Route:** There are no REST webhooks or routing endpoints that expose manual alert triggers.
* **Missing Worker:** No daemon handles alert deduplication or cool-down loops.
* **Missing Transport:** The alert bridges have no live endpoints or configured API secrets wired at runtime.

### 🚨 `internal/streaming` (AI Health, Cardinality, Coverage Jobs)

* **Target Execution Path:** Needs to run as background worker jobs within `cmd/worker/main.go` to compute running aggregations for ClickHouse tables.
* **Missing Constructor:** No Go code calls the job constructor functions (e.g. `streaming.NewCoverageJob`).
* **Missing Dependency Injection:** Jobs are not injected or started inside the Kafka consumer loops.
* **Missing Route:** No administration endpoints exist to manually trigger, check, or reset streaming jobs.
* **Missing Worker:** No execution threads run these jobs at runtime.
* **Missing Transport:** Database handles and Kafka configurations are never passed to the streaming jobs.

### 🚨 `internal/mcp` (MCP Server & Query Clients)

* **Target Execution Path:** Needs to expose a dedicated stdin/stdout reader daemon (`cmd/mcp-server`) or run as an HTTP/SSE endpoint sidecar inside `cmd/api-server/main.go`.
* **Missing Constructor:** `mcp.NewServer` has zero caller references.
* **Missing Dependency Injection:** The MCP server is not registered with the HTTP router or daemon controllers.
* **Missing Route:** No REST/WS routes handle MCP protocol handshakes.
* **Missing Worker:** Lacks a session worker thread to handle client context tracking.
* **Missing Transport:** Lacks stdio scanners or SSE writer streams to serialize JSON-RPC messages to external clients.
