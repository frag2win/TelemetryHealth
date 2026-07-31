# 09 — Testing Audit

## Coverage Analysis

| Package | Test Files | Coverage Estimate | Gap |
|---------|-----------|-------------------|-----|
| `processor/cardinality` | ✅ Yes | ~93% (claimed) | Good |
| `processor/failopen` | ✅ Yes | ~94% (claimed) | Good |
| `processor/tracechain` | ✅ Yes | Unknown | Need to verify |
| `control-plane/authz` | ✅ Yes | ~100% (claimed) | Good |
| `control-plane/api/rest` | ✅ Yes | ~30% (estimated) | Missing: most handlers, CORS, tenant validation |
| `control-plane/streaming` | ✅ Yes | Unknown | Unit tests exist |
| `control-plane/mcp` | ✅ Yes | Unknown | |
| `control-plane/kafka` | ✅ Yes | Unknown | |
| `control-plane/behavior` | ✅ Yes | Unknown | |
| `control-plane/decision` | ✅ Yes | Unknown | |
| `control-plane/rootcause` | ✅ Yes | Unknown | |
| `control-plane/remediation` | ✅ Yes | Unknown | |
| **`control-plane/ingest`** | ❌ **None** | **0%** | **Critical gap — gRPC handler is untested** |
| **`control-plane/storage/clickhouse`** | ❌ **None** | **0%** | **All real DB queries untested** |
| **`control-plane/alerting`** | ❌ **None** | **0%** | **Alert bridge + poller untested** |
| **`control-plane/engine`** | ❌ **None** | **0%** | **Graph engine untested** |
| **`dashboard/`** | ❌ **None** | **0%** | **Entire frontend untested** |

## Critical Test Gaps

1. **No integration tests** that test the actual Kafka → ClickHouse pipeline
2. **No API contract tests** — no test validates the JSON schema of API responses
3. **No load/stress tests in CI** — a k6 script exists in `test/load/` but has no CI integration
4. **E2E test** (`cmd/e2e-test`) is a standalone binary, not integrated into CI
5. **No race condition tests** — CI doesn't run with `-race` flag

## Testing Anti-patterns

### TEST-01: Test Relies on Server-Level Integration Instead of Unit Testing
- **File**: `control-plane/internal/api/rest/server_test.go`
- **Explanation**: Tests create a full `Server` struct with mock repository, which is good for integration but means individual handler logic isn't unit-tested in isolation.

### TEST-02: No Table-Driven Tests
- **Explanation**: Most test files use individual test functions instead of Go's idiomatic `t.Run` subtests / table-driven pattern.

### TEST-03: No Fuzz Testing
- **Explanation**: No `Fuzz*` test functions exist. The remediation validator (which parses untrusted YAML) and the OIDC token parser would greatly benefit from fuzz testing.

### TEST-04: No Benchmark Tests
- **Explanation**: No `Benchmark*` test functions exist. The HLL cardinality tracker and health score calculator are performance-sensitive and should be benchmarked.
