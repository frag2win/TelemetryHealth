# TelemetryHealth — Repository Health Audit

**Date:** 2026-07-17
**Scope:** `control-plane/` Go codebase

---

## 1. Architectural Smells

### 🚨 God Objects
*   `*clickhouse.HealthRepository`: At 555 lines, this struct violates the Single Responsibility Principle entirely. It acts as a God Object handling:
    1.  Raw OpenTelemetry span querying
    2.  Cardinality math
    3.  Mock data generation (falling back to hardcoded traces)
    4.  ClickHouse connection state
    5.  Business logic for health scoring
    *This object forces the entire system to couple to ClickHouse.*

### 🚨 Poor Abstractions
*   **No Dependency Inversion:** The `api-server` does not depend on a `DomainRepository` interface. It depends directly on `*clickhouse.HealthRepository`. If you want to test the REST server, you must spin up a ClickHouse instance or heavily modify the server code.
*   **Leaky DTOs:** The REST server (`api/rest/server.go`) imports `mcp.HealthResponse` from the `internal/mcp` package and uses it as its HTTP JSON response shape, tightly coupling the REST API to the MCP tool schema.

### ⚠️ Duplicated Code
*   **Remediation Logic:** Both `rest/server.go` (`GetTenantHealth`) and `mcp/tools.go` (`GetTelemetryHealth`) duplicate the exact same 15-line block of logic to generate and validate YAML remediation snippets.
*   **Type Systems:** The project maintains redundant type structures between `internal/engine` and `pkg/models`.

### ⚠️ Cyclic Imports (Avoided via Coupling)
*   Go physically prevents cyclic imports. However, to avoid cyclic imports between `internal/engine` and `internal/api`, the developers crammed domain business logic directly into the ClickHouse storage layer, resulting in architectural collapse.

---

## 2. Dead Weight

### 🧟 Unused Packages
*   `internal/mcp`: The MCP server is a Go struct with no transport listener, no caller, and no route registration. It is 100% dead weight.
*   `internal/alerting`: The `SlackBridge`, `PagerDutyBridge`, and `SigNozBridge` are fully implemented but never instantiated or invoked by any `main()` binary.
*   `internal/decision`, `internal/behavior`, `internal/rootcause`: The core AI domain logic engines are completely bypassed at runtime in favor of hardcoded mocks.

### 📦 Unused Dependencies
*   `segmentio/kafka-go`: Used only by the dead-end `worker` binary.
*   `go.opentelemetry.io/otel/sdk`: Imported, but the `InitOTelSDK` function (`internal/telemetry/otel.go`) is a stub where the actual exporter initialization is commented out.

---

## 3. Code Quality Indicators

### 📏 Large Files & Long Methods
*   `api/rest/server.go`: 851 lines. Contains routing, middleware, structural JWT parsing, CORS, and massive 100+ line handler methods (`GetTenantHealth`, `ApplyRemediation`) that mix HTTP parsing with domain logic.
*   `storage/clickhouse/health_repository.go`: 555 lines. The `QueryHealthMetrics` method is a massive block of inline SQL strings mixed with business logic.

### 🔮 Magic Strings
*   **Hardcoded Tenant Bypass:** `rest/server.go` contains a magic string bypass for authentication: `tenantID != "acme-prod" && tenantID != "acme-staging" ...`.
*   **Database Bypasses:** `health_repository.go` hardcodes `"signoz_traces.signoz_index_v2"`, assuming internal knowledge of an external product's database schema.
*   **Hardcoded Data:** `mcp/client.go` hardcodes `"[SIMULATED] t1"` for trace IDs.

---

## 4. Technical Debt & Incomplete Implementations

### 📝 TODOs, FIXMEs, and XXXs
The codebase is littered with comments acknowledging incomplete or faked implementations for the sake of the hackathon demo:

*   **`mcp/client.go:26`**
    ```go
    // Mocking the MCP query for the hackathon demo
    query := `SELECT * FROM traces WHERE attributes['service.name'] = 'ai-agent'`
    _ = query // keep compiler happy
    ```
*   **`mcp/client.go:29`**
    ```go
    // In a real implementation:
    // import "github.com/signoz/mcp-go"
    ```
*   **`engine/rootcause.go:103`**
    ```go
    // For the Hackathon Demo, if we detect no explicit errors but want to show a propagation:
    ```
*   **`rest/server.go:344`**
    ```go
    // Could add a lightweight DB ping here when health repository exposes Ping().
    ```
*   **`api/rest/server.go` (OIDC Middleware)**
    ```go
    // Fallback: structural JWT parse only (no signature check).
    // This path is acceptable only in non-production environments without an IdP yet.
    ```
