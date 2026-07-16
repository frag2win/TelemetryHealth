# TelemetryHealth — SigNoz Integration Forensic Audit

**Perspective:** SigNoz maintainer reviewing this project for genuine SigNoz integration.
**Date:** 2026-07-17
**Method:** Traced every `signoz` reference across all Go source, config files, dashboards, docs, and dependencies. Verified each against actual import chains, API calls, and runtime behavior.

---

## Executive Verdict

> [!CAUTION]
> **This project does NOT genuinely integrate with SigNoz.** It uses raw ClickHouse and raw OpenTelemetry SDKs directly — both of which are general-purpose technologies that exist independently of SigNoz. Every reference to "SigNoz" in the codebase is either a comment, a log string, a mock, or dead code. There is **zero runtime dependency** on any SigNoz API, service, library, or process.

---

## Layer Classification: What Is Actually Used

### ✅ Genuinely Used: OpenTelemetry (Vendor-Neutral)

These are **standard OTel** components. They work with any OTLP-compatible backend (Jaeger, Grafana Tempo, Datadog, SigNoz, etc.). They are not SigNoz-specific.

| Component | File | Evidence |
|---|---|---|
| `ptraceotlp` gRPC receiver | [grpc_server.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/ingest/grpc_server.go) | Standard `go.opentelemetry.io/collector/pdata` imports |
| `pmetricotlp` gRPC receiver | Same file | Standard OTLP metrics receiver |
| `plogotlp` gRPC receiver | Same file | Standard OTLP logs receiver |
| OTel propagation SDK | [otel.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/telemetry/otel.go) | `go.opentelemetry.io/otel` — sets `TraceContext` + `Baggage` propagators |
| OTel trace context injection | [client.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/mcp/client.go#L12) | `otel.GetTextMapPropagator().Inject()` — standard W3C propagation |
| Prometheus metrics | [metrics.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/telemetry/metrics.go) | `prometheus/client_golang` — standard Prometheus, not SigNoz |
| `ptrace` span construction | [scenarios.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/simulator/scenarios.go) | `pdata/ptrace` for building synthetic OTLP spans |

> **Verdict:** All telemetry code is vendor-neutral OpenTelemetry. Replacing "SigNoz" with "Jaeger" or "Grafana Tempo" would require zero code changes.

### ✅ Genuinely Used: ClickHouse (Generic Database)

These are **raw ClickHouse** operations using the `clickhouse-go/v2` driver directly. They target TelemetryHealth's own custom tables (not SigNoz's schema).

| Component | File | Evidence |
|---|---|---|
| `clickhouse.Open()` connection | [client.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/storage/clickhouse/client.go) | Direct `clickhouse-go/v2` driver, no SigNoz wrapper |
| `telemetry_health.*` schema DDL | [schema.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/storage/clickhouse/schema.go) | Custom database `telemetry_health` with custom tables |
| Health metrics queries | [health_repository.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/storage/clickhouse/health_repository.go) | Direct SQL against `telemetry_health.cardinality_signal`, etc. |
| Replay event queries | [replay_repository.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/storage/clickhouse/replay_repository.go) | Direct SQL against `telemetry_health.telemetryhealth_trace_index_spans` |
| Worker batch inserts | [workers.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/kafka/workers.go) | `PrepareBatch` + raw SQL into custom tables |

> **Verdict:** The project brings its own ClickHouse client, its own database (`telemetry_health`), and its own schema. It does not use SigNoz's query service, SigNoz's schema, or SigNoz's ClickHouse deployment.

---

## Forensic Findings: Every SigNoz Reference

### Finding 1: `signoz_traces.signoz_index_v2` — Querying SigNoz's Internal Table Directly

> **Classification:** ⚠️ **Bypasses SigNoz** (accesses SigNoz internals, not SigNoz's API)
> **Severity:** Critical

**Files:**
- [health_repository.go L149](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/storage/clickhouse/health_repository.go#L149) — `QueryAgentTraces`
- [health_repository.go L309](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/storage/clickhouse/health_repository.go#L309) — `QuerySpansByTraceID`

```sql
SELECT trace_id, attributes_map['gen_ai.request.model'] AS model, ...
FROM signoz_traces.signoz_index_v2
WHERE attributes_map['gen_ai.system'] != ''
```

**What's happening:**
The code directly queries `signoz_traces.signoz_index_v2` — this is SigNoz's **internal ClickHouse table**. It is not a public API. This table's schema is an internal implementation detail of SigNoz that changes between SigNoz versions without warning or migration guarantees.

**Why this is NOT SigNoz integration:**
- SigNoz exposes a **Query Service REST API** (`/api/v1/traces`, `/api/v3/query_range`) for reading trace data. This project does not use it.
- Direct table access bypasses SigNoz's RBAC, query limits, and data access policies.
- The column `attributes_map` is a SigNoz-internal column name that has changed in past SigNoz versions.

**What makes it worse:**
Both functions **fall back to hardcoded mock data** if the query fails (L183, L348). This means the feature works entirely without SigNoz. SigNoz is not a dependency — it's an optional, fragile optimization.

```go
// L183: Fallback to rich, realistic traces if ClickHouse returned nothing or errored out
if len(traces) == 0 {
    traces = []AgentTrace{
        {ID: "trace-991", Model: "gpt-4o", ...},  // hardcoded mock
    }
}
```

---

### Finding 2: `SigNozBridge` — Fake Alert Bridge

> **Classification:** 🎭 **Imitates SigNoz** (claims SigNoz Alertmanager, does nothing)
> **Severity:** Critical

**File:** [signoz_bridge.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/alerting/signoz_bridge.go)

```go
// "fires alerts to SigNoz Alertmanager" — the comment says
func (b *SigNozBridge) FireAlert(ctx context.Context, payload AlertPayload) error {
    // ...deduplication logic...
    b.logger.Info("Firing alert to SigNoz Alertmanager",
        zap.String("alert_id", payload.AlertID),
        // ...
    )
    return nil   // ← DOES NOTHING. No HTTP call. No API interaction.
}
```

**What's happening:**
- The comment says "fires alerts to SigNoz Alertmanager"
- The function logs a message and returns `nil`
- There is **no HTTP client**, no URL, no API call, no webhook, no network I/O
- No import of `net/http` in this file
- The `SigNozBridge` struct has no URL field, no API key field, no client field

**Compare with the real bridges:**
- `SlackBridge` ([slack_bridge.go L38](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/alerting/slack_bridge.go#L38)): Has `httpClient *http.Client`, `webhookURL string`, and actually calls `b.httpClient.Do(req)`
- `PagerDutyBridge` ([pagerduty_bridge.go L44-45](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/alerting/pagerduty_bridge.go#L44-L45)): Has `httpClient *http.Client`, `apiEndpoint string`, and actually calls `b.httpClient.Do(req)`
- `SigNozBridge`: Has **none of these**. It's a log statement pretending to be an integration.

**Additionally:** The entire `alerting` package is dead code — never reached from any executable (see Execution Graph audit V1).

---

### Finding 3: MCP Server — Claims SigNoz, Is Standalone

> **Classification:** 🎭 **Incorrectly Claims SigNoz Integration**
> **Severity:** High

**File:** [mcp/server.go L9](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/mcp/server.go#L9)

```go
// Server implements SigNoz MCP server tools for AI agent interaction.
```

**What's happening:**
- The comment says "SigNoz MCP server tools"
- The code has zero imports from SigNoz
- No `github.com/signoz/*` dependency in any `go.mod`
- `HandleToolCall` routes to `GetTelemetryHealth` and `GenerateRemediation` — both use this project's own `HealthRepository`, not SigNoz's API
- The server is never started (no transport wired, per Execution Graph audit)

**SigNoz MCP reality:**
SigNoz does not currently ship an MCP server SDK called `mcp-go`. The comment in [client.go L30](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/mcp/client.go#L30) explicitly acknowledges this:

```go
// In a real implementation:
// import "github.com/signoz/mcp-go"
// mcpClient := mcp.NewClient(serverURL)
// return mcpClient.Query(ctx, query)
```

This is aspirational code — `github.com/signoz/mcp-go` doesn't exist as a public package.

---

### Finding 4: MCP Client — Returns Hardcoded `[SIMULATED]` Data

> **Classification:** 🎭 **Imitates SigNoz** (simulates MCP query results)
> **Severity:** High

**File:** [mcp/client.go L25-43](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/mcp/client.go#L25-L43)

```go
// QueryAgentTraces queries the SigNoz MCP server for traces related to AI agents
func QueryAgentTraces(ctx context.Context, tenantID string, serverURL string) (*Traces, error) {
    // Mocking the MCP query for the hackathon demo
    query := `SELECT * FROM traces WHERE attributes['service.name'] = 'ai-agent'`
    _ = query // keep compiler happy

    return &Traces{
        Count: 2,
        Data: []map[string]interface{}{
            {"trace_id": "[SIMULATED] t1", ...},
            {"trace_id": "[SIMULATED] t2", ...},
        },
    }, nil
}
```

The function:
1. Constructs a SQL query string and immediately discards it (`_ = query`)
2. Ignores the `serverURL` parameter entirely
3. Returns hardcoded data with `[SIMULATED]` prefixes
4. Is never called from any runtime path (dead code)

---

### Finding 5: SigNoz Dashboard JSONs — Valid Format, But Untested

> **Classification:** ⚠️ **Partial/Plausible Integration Attempt**
> **Severity:** Medium

**Files:**
- [dashboard/signoz/telemetry-health-overview.json](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/dashboard/signoz/telemetry-health-overview.json)
- [dashboard/signoz/cardinality-top-offenders.json](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/dashboard/signoz/cardinality-top-offenders.json)
- [dashboard/signoz/coverage-gaps.json](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/dashboard/signoz/coverage-gaps.json)

**What's happening:**
Three JSON files use a structure that *resembles* SigNoz's dashboard import format (`panelTypes`, `clickhouse_sql`, `widgets`). The SQL queries target TelemetryHealth's own `telemetry_health.*` tables.

**Assessment as a SigNoz maintainer:**
- The JSON schema **is not the exact SigNoz dashboard export format**. SigNoz dashboards include fields like `id`, `uuid`, `created_at`, `updated_at`, `layout`, `panelMap`, and use a different widget structure. These JSONs would fail import into SigNoz without modification.
- The SQL queries are valid ClickHouse SQL and would work if imported manually.
- No CI/CD pipeline or import script exists in this repo to sync these dashboards to a SigNoz instance.
- **Verdict:** These are mock-ups *inspired by* SigNoz's dashboard concept but are not importable artifacts.

---

### Finding 6: `signoz_implementations/agent_health.json` — Closer to Real SigNoz Format

> **Classification:** ⚠️ **Partial/Plausible Integration Attempt**
> **Severity:** Medium

**File:** [signoz_implementations/agent_health.json](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/signoz_implementations/agent_health.json)

**Assessment:**
- Uses `queryType: "promql"` with PromQL queries like `avg(telemetryhealth_agent_health_score{...}) by (agent_id)` — this is closer to SigNoz's actual dashboard format.
- Has `variables`, `layout`, `widgets` structure that is more aligned with SigNoz's JSON dashboard schema.
- References metrics like `telemetryhealth_agent_health_score`, `telemetryhealth_agent_token_burn_rate`, `telemetryhealth_agent_trace_error_count` — **none of these metrics exist in the codebase**. The actual metrics defined in [metrics.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/telemetry/metrics.go) are:
  - `telemetryhealth_ingested_spans_total`
  - `telemetryhealth_kafka_messages_processed_total`
  - `telemetryhealth_api_requests_total`
  - `telemetryhealth_api_request_duration_seconds`
  - `telemetryhealth_clickhouse_write_duration_seconds`
  - `telemetryhealth_pipeline_health_score`

**Mismatch:** The dashboard references 3 metrics that don't exist; none of the 6 actual metrics appear in the dashboard. This dashboard would show **zero data** even with a running SigNoz instance.

---

### Finding 7: `signoz_implementations/clickhouse_migration.sql` — Orphaned Schema

> **Classification:** ⚠️ **Bypasses SigNoz** (uses ClickHouse directly)
> **Severity:** Low

**File:** [clickhouse_migration.sql](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/signoz_implementations/clickhouse_migration.sql)

Creates `telemetry_health.signal_metrics` and `telemetry_health.root_cause_records` tables. These tables:
- Are never referenced by any Go code in the repository
- Are not created by `cmd/init-db` (which creates a different set of tables via `schema.go`)
- Have no writer, no reader, no test
- Use `AggregatingMergeTree` and `ReplacingMergeTree` engines that differ from the `MergeTree` engines used by the actual schema

**Verdict:** Orphaned SQL file — aspirational but disconnected.

---

### Finding 8: `casting.yaml` — Deployment Config References Non-Existent Integration

> **Classification:** 🎭 **Incorrectly Claims SigNoz Integration**
> **Severity:** Medium

**File:** [casting.yaml](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/casting.yaml)

```yaml
name: telemetryhealth-signoz
variables:
  signoz:
    version: "0.112.0"
  exporters:
    signoz:
      dashboard_path: "./signoz_implementations/agent_health.json"
      alerts_path: "./alerts/telemetry-health-alerts.yaml"
```

- `alerts_path: "./alerts/telemetry-health-alerts.yaml"` — this file **does not exist** in the repository
- `dashboard_path` references the `agent_health.json` that uses non-existent metrics (Finding 6)
- SigNoz version `0.112.0` is referenced but no SigNoz Docker image, Helm chart, or deployment manifest is present
- No `docker-compose.yaml` mentions SigNoz

---

## Integration Reality Matrix

| Claim in Code/Docs | What Actually Happens | Classification |
|---|---|---|
| "SigNoz MCP server tools" | Standalone REST/MCP handlers querying own ClickHouse tables | 🎭 Incorrect claim |
| "Fires alerts to SigNoz Alertmanager" | `logger.Info()` + `return nil` — no HTTP call | 🎭 Imitation |
| "Queries SigNoz MCP server" | Returns `[SIMULATED]` hardcoded data | 🎭 Imitation |
| `FROM signoz_traces.signoz_index_v2` | Direct ClickHouse query bypassing SigNoz Query Service | ⚠️ Bypass |
| "SigNoz native dashboard" JSONs | Non-importable mock-ups with mismatched metrics | ⚠️ Partial attempt |
| `casting.yaml` references SigNoz 0.112.0 | No deployment config, no Helm chart, referenced files missing | 🎭 Incorrect claim |
| OTel SDK (traces, metrics, propagation) | Standard vendor-neutral OpenTelemetry — works with any backend | ✅ Real but NOT SigNoz-specific |
| ClickHouse (read/write `telemetry_health.*`) | Raw driver, custom schema, custom tables | ✅ Real but NOT SigNoz |
| Prometheus metrics (`promhttp`) | Standard Prometheus client library | ✅ Real but NOT SigNoz |

---

## Dependency Chain Proof

### `go.mod` Analysis (all modules)

```
grep result: "signoz" found in go.mod → 0 results
grep result: "signoz" found in go.sum → 0 results
```

**Zero SigNoz Go packages** are imported anywhere in the project. No `github.com/SigNoz/*` dependency. No `github.com/signoz/*` dependency.

### `docker-compose` Analysis

```
grep result: "signoz" found in docker-compose files → 0 results
```

**No SigNoz container** is declared in any Docker composition file.

### Runtime Process Dependencies

| Process | Depends on ClickHouse | Depends on Kafka | Depends on SigNoz | 
|---|---|---|---|
| `api-server` | Yes (direct driver) | No | **No** |
| `ingest-gateway` | No | Yes (direct driver) | **No** |
| `worker` | Yes (direct driver) | Yes (direct driver) | **No** |
| `init-db` | Yes (direct `database/sql`) | No | **No** |
| `seeder` | Yes (direct driver) | No | **No** |
| `simulator` | No | No (sends OTLP to gateway) | **No** |
| `e2e-test` | No | No (sends OTLP to gateway) | **No** |
| `dashboard` | No (calls api-server REST) | No | **No** |

**SigNoz does not appear in any runtime dependency chain.**

---

## What Genuine SigNoz Integration Would Look Like

As a SigNoz maintainer, here is what I would expect to see for a project that genuinely integrates with SigNoz:

| Integration Point | What Exists | What Should Exist |
|---|---|---|
| **Trace querying** | Direct SQL to `signoz_traces.signoz_index_v2` | `GET /api/v3/query_range` or SigNoz Query Service client |
| **Dashboard import** | Mock JSON files | Dashboards importable via SigNoz REST API (`POST /api/v1/dashboards`) with actual SigNoz schema |
| **Alerting** | Log-only `SigNozBridge` | Webhook/API calls to SigNoz's alert manager endpoint |
| **MCP tools** | Comment saying `import "github.com/signoz/mcp-go"` | Actual import and instantiation of SigNoz MCP client |
| **Deployment** | No SigNoz in docker-compose | `signoz/signoz` or `signoz/otel-collector` container with proper volume and network config |
| **Go dependency** | Zero SigNoz imports | `github.com/SigNoz/signoz/pkg/query-service` or similar |
| **Authentication** | None | SigNoz API key or PAT-based auth for Query Service calls |

---

## Conclusion

| Category | Count | Severity |
|---|---|---|
| 🎭 Incorrectly claims SigNoz integration | 4 | Critical |
| ⚠️ Bypasses SigNoz (uses ClickHouse directly) | 2 | High |
| ⚠️ Partial/aspirational attempts | 2 | Medium |
| ✅ Genuine SigNoz dependency | **0** | — |

**The project uses OpenTelemetry (vendor-neutral) and ClickHouse (generic database) directly.** Every reference to "SigNoz" is either:
1. A **comment or log string** with no backing implementation
2. A **mock function** that returns hardcoded data
3. A **dead code path** that is never reached from any executable
4. A **raw SQL query** that bypasses SigNoz's API layer to access internal tables

If SigNoz were removed from the universe tomorrow, this project would continue to function with **zero code changes**.
