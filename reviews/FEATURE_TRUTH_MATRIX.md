# TelemetryHealth — Feature Truth Matrix

**Date:** 2026-07-17
**Methodology:** All claims verified strictly via source code analysis, import graphs, and AST parsing. Documentation and PRD claims were entirely ignored.

### Legend
*   **Claimed**: Does the project claim this feature exists?
*   **Implemented**: Does Go source code for this logic actually exist?
*   **Reachable**: Is there an uninterrupted call chain from a `main()` entrypoint to this code?
*   **Tested**: Does a corresponding `_test.go` file exist?
*   **Demo Ready**: Can this feature be visually demonstrated to a judge (even if faked)?
*   **Evidence**: The structural proof from the source code.

---

| Feature | Claimed | Implemented | Reachable | Tested | Demo Ready | Evidence (Source Truth) |
| :--- | :---: | :---: | :---: | :---: | :---: | :--- |
| **REST API Server** | ✅ | ✅ | ✅ | ✅ | ✅ | `api-server` binds a `chi` router on port 8080. Tested via `server_test.go`. |
| **React Dashboard** | ✅ | ✅ | ✅ | ❌ | ✅ | `dashboard/src/` contains 8 React views. Connects to REST API endpoints. |
| **OTel gRPC Ingestion** | ✅ | ✅ | ✅ | ❌ | ✅ | `cmd/ingest-gateway` wires up `pdata/ptraceotlp` receivers. No tests exist in `internal/ingest/`. |
| **ClickHouse Storage** | ✅ | ✅ | ✅ | ✅ | ✅ | `clickhouse-go/v2` driver used directly in `health_repository.go`. Tested in `health_repository_test.go`. |
| **Remediation Generator** | ✅ | ✅ | ✅ | ✅ | ✅ | `internal/remediation/generator.go` renders templates. Called directly by REST handler. |
| **Kafka Stream Processing** | ✅ | ✅ | ✅ | ✅ | ❌ | `cmd/worker` initializes Sarama consumers and reads topics. Tested in `kafka_test.go`. |
| **AI Behavior Reconstruction** | ✅ | ✅ | ❌ | ✅ | ✅ | Rich domain logic in `internal/behavior/`. Never called. API serves hardcoded `[SIMULATED]` data instead. |
| **AI Decision Reconstruction** | ✅ | ✅ | ❌ | ✅ | ✅ | Exists in `internal/decision/`. Completely isolated dead code. UI shows fake API data. |
| **AI Root Cause Analysis** | ✅ | ✅ | ❌ | ✅ | ✅ | Exists in `internal/rootcause/`. Completely isolated dead code. UI shows fake API data. |
| **Slack / PagerDuty Alerts** | ✅ | ✅ | ❌ | ❌ | ❌ | HTTP clients exist in `internal/alerting/`, but are never instantiated or invoked by any binary. |
| **Model Context Protocol (MCP)** | ✅ | ⚠️ Partial | ❌ | ❌ | ❌ | Logic exists, but lacks transport (stdio/HTTP) and JSON-RPC framing. `mcp.NewServer` has zero callers. |
| **SigNoz Trace Integration** | ✅ | ❌ | ❌ | ❌ | ✅ | Skips SigNoz API. Runs raw SQL against internal `signoz_traces` table, then falls back to mocks if it fails. |
| **SigNoz Alert Bridge** | ✅ | ❌ | ❌ | ❌ | ❌ | `internal/alerting/signoz_bridge.go` is a fake implementation. It logs a string and returns `nil`. |

---

## 🔍 Forensic Summary

1.  **The "Demo" Layer is Fully Reachable:** Everything required to make a YouTube video (Dashboard, REST API, Remediation templates, and basic ClickHouse connections) is implemented, reachable, and works.
2.  **The "Intelligence" Layer is Dead Code:** The entire AI reconstruction suite (Behavior, Decision, Root Cause) is implemented and well-tested, but the developers never wired it into the runtime. The API returns hardcoded arrays to keep the dashboard working.
3.  **The "Ecosystem" Layer is Vaporware:** Every external integration (SigNoz APIs, Alertmanager, Slack, PagerDuty, MCP) is either entirely faked (dummy functions) or partially implemented but unreachable.
