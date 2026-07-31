# 15 — Bug Report

### BUG-01: Kafka rawspan Writer Not Closed
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/kafka/producer.go` L123-141
- **Type**: Resource leak
- **Explanation**: `Close()` closes cardinality, orphan, and coverage writers but NOT the `rawspan` writer, leaking its Kafka connection and goroutines on shutdown.
- **Fix**: Add `if err := p.rawspan.Close(); err != nil { ... }` to `Close()`.

### BUG-02: Rate Limiter Includes Port in Key
- **Severity**: 🟠 High
- **File**: `control-plane/internal/api/rest/server.go` L204
- **Type**: Logic error
- **Explanation**: `r.RemoteAddr` includes the port (e.g., `192.168.1.1:54321`). Each TCP connection gets a unique port, so each request gets its own rate limiter bucket — completely defeating rate limiting.
- **Fix**: Use `net.SplitHostPort(r.RemoteAddr)` to extract only the IP.

### BUG-03: ClickHouse Rows Not Deferred in SigNoz Query Path
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/storage/clickhouse/health_repository.go` L643
- **Type**: Resource leak risk
- **Explanation**: In `QuerySpansByTraceID`, the SigNoz fallback query path uses manual `rows.Close()` at L665 instead of `defer`. If `rows.Next()` or `rows.Scan()` panics, rows won't be closed, leaking a database connection.
- **Fix**: Replace `rows.Close()` with `defer rows.Close()` immediately after the nil check.

### BUG-04: `simulateTimeRangeMetrics` Overwrites Real Data
- **Severity**: 🟠 High
- **File**: `dashboard/src/App.tsx` L236-275
- **Type**: Data corruption (client-side)
- **Explanation**: When the user changes the time range dropdown, real API data (cardinality, orphans) is replaced with hardcoded strings like `"1.1M"` and `"8.4%"`. The dashboard displays fabricated data even when connected to a live backend.
- **Fix**: Move time-range filtering to the backend API, or at minimum, only apply simulation when `dataSource === 'mock'`.

### BUG-05: `GetCoverage` Godoc on Wrong Function
- **Severity**: 🟢 Low
- **File**: `control-plane/internal/api/rest/server.go` L603
- **Type**: Documentation bug
- **Explanation**: The comment `// GetCoverage returns service coverage status.` appears above `handleBehaviorGraph` instead of the actual `GetCoverage` function.

### BUG-06: `Promise.all().catch()` Returns Wrong Type
- **Severity**: 🟡 Medium
- **File**: `dashboard/src/App.tsx` L283-287
- **Type**: Runtime error risk
- **Explanation**: `.catch(() => [null, null, null])` on `Promise.all` returns an array as the catch value, but the destructured result `[agentsRes, orphansRes, coverageRes]` expects the Promise.all resolution shape. If the catch fires, the destructured variables will be `undefined`.
- **Fix**: Use `Promise.allSettled` or wrap each fetch in its own `.catch()`.

### BUG-07: MCP Server `sseFlag` Parsed But Never Read
- **Severity**: 🟢 Low
- **File**: `control-plane/cmd/mcp-server/main.go`
- **Type**: Dead code / logic error
- **Explanation**: `sseFlag` is declared and parsed as a CLI flag but the code only checks `stdioFlag`. If the user passes `--sse`, nothing happens.
- **Fix**: Add the SSE transport logic or remove the flag.

### BUG-08: Shutdown Typo
- **Severity**: 🟢 Low
- **File**: `control-plane/internal/api/rest/server.go` L314
- **Type**: Typo
- **Explanation**: `"Shutding down API Server"` → should be `"Shutting down API Server"`.
