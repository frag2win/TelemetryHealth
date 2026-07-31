# 06 — Backend Audit (Go Code Quality)

## Go Idiomatic Issues

### BE-01: `os.Setenv` Called from Production Code
- **Severity**: 🟠 High
- **File**: `control-plane/cmd/api-server/main.go` L25, `cmd/ingest-gateway/main.go` L23
- **Explanation**: `os.Setenv("INSECURE_DEV_MODE", "true")` is called at startup. This mutates process-level environment, is not goroutine-safe, and will appear in `/proc/environ` for any child process.
- **How to fix**: Use a configuration struct with explicit fields, not environment variable mutation.

### BE-02: Context Handling Anti-patterns
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/api/rest/server.go` L739
- **Explanation**: `SimulateFailure` creates a new `context.Background()` instead of using `r.Context()`. This means the simulation isn't cancelled if the HTTP client disconnects.
- **How to fix**: Use `r.Context()` as the parent.

### BE-03: Typo in Log Message
- **Severity**: 🟢 Low
- **File**: `control-plane/internal/api/rest/server.go` L314
- **Explanation**: `"Shutding down API Server"` should be `"Shutting down API Server"`.

### BE-04: `cardChange` Returns Hardcoded Values
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/api/rest/server.go` L444-449
- **Explanation**: `cardChange()` returns `14.5` or `2.1` regardless of actual data changes. This is fake data presented as real metrics.

### BE-05: Unused `var _ = ...` Import Guards
- **Severity**: 🟢 Low
- **File**: `control-plane/internal/kafka/workers.go` L188-192
- **Explanation**: `var _ = kafkago.Message{}`, `var _ = strconv.Itoa`, `var _ = time.Now` — these are import guards for packages that ARE used elsewhere in the file. They're unnecessary noise.

### BE-06: Producer.Close() Missing rawspan Writer
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/kafka/producer.go` L123-141
- **Explanation**: `Close()` closes cardinality, orphan, and coverage writers but NOT the `rawspan` writer, leaking its connection.
- **How to fix**: Add `p.rawspan.Close()` to the Close method.

### BE-07: Error Handling Inconsistencies
- **Severity**: 🟡 Medium
- **Explanation**: Handler error response codes are inconsistent:
  - `GetTenantHealth` returns 503 on DB errors
  - `handleBehaviorGraph` returns 501 on missing features  
  - `SimulateFailure` returns 500 on errors
  - `GetCoverage` never returns errors (hardcoded data)
- **How to fix**: Standardize: 503 for downstream unavailability, 500 for internal errors, 400 for input validation.

### BE-08: `encodeResponse` Logs But Can't Fix HTTP Status
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/api/rest/server.go`
- **Explanation**: `encodeResponse` sets the status code then writes JSON. If JSON encoding fails, the status code has already been sent. The error is logged but the client receives an empty body with a 200 status.
- **How to fix**: Buffer the JSON response before writing headers.
