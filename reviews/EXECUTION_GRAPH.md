# TelemetryHealth Complete Execution Graph

Date: 2026-07-17
Scope: Every executable under `control-plane/cmd/`, plus `processor/`, `tools/docs-bot/`, and `dashboard/`.
Method: Source-traced actual `import` chains, constructor calls, and runtime invocations — no assumptions.

---

## Master Subsystem Reachability Matrix

| Subsystem | api-server | ingest-gateway | worker | init-db | seeder | simulator | e2e-test | dashboard | processor | docs-bot |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `internal/api/rest` | ✅ | — | — | — | — | — | — | — | — | — |
| `internal/ingest` | — | ✅ | — | — | — | — | — | — | — | — |
| `internal/kafka` (Producer) | — | ✅ | — | — | — | — | — | — | — | — |
| `internal/kafka` (WorkerSet) | — | — | ✅ | — | — | — | — | — | — | — |
| `internal/kafka` (EnsureTopics) | — | ✅ | ✅ | — | — | — | — | — | — | — |
| `internal/storage/clickhouse` (Client) | ✅ | — | ✅ | — | — | — | — | — | — | — |
| `internal/storage/clickhouse` (Schema) | — | — | — | ✅ | — | — | — | — | — | — |
| `internal/storage/clickhouse` (HealthRepo) | ✅ | — | — | — | — | — | — | — | — | — |
| `internal/storage/clickhouse` (ReplayRepo) | ✅ | — | — | — | — | — | — | — | — | — |
| `internal/authz` | ✅ | ✅ | — | — | — | — | — | — | — | — |
| `internal/telemetry` (Prometheus) | ✅ | ✅ | ✅ | — | — | — | — | — | — | — |
| `internal/telemetry` (OTel SDK) | ✅ | ✅ | ✅ | — | — | — | — | — | — | — |
| `internal/telemetry` (HealthScore) | ✅ | — | — | — | — | — | — | — | — | — |
| `internal/remediation` | ✅ | — | — | — | — | — | — | — | — | — |
| `internal/engine` (Graph) | ✅ | — | — | — | — | — | — | — | — | — |
| `internal/behavior` | ✅ | — | — | — | — | — | — | — | — | — |
| `internal/decision` | ✅ | — | — | — | — | — | — | — | — | — |
| `internal/rootcause` | ✅ | — | — | — | — | — | — | — | — | — |
| `internal/simulator` | ✅ | — | — | — | — | ✅ | — | — | — | — |
| `internal/mcp` (DTOs only) | ✅† | — | — | — | — | — | — | — | — | — |
| `pkg/models` | ✅ | — | — | — | — | — | — | — | — | — |
| `dashboard` (React) | — | — | — | — | — | — | — | ✅ | — | — |
| **`internal/alerting`** | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ |
| **`internal/streaming`** | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ |
| **`internal/mcp` (Server/Tools)** | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ |
| **`internal/mcp` (Client)** | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ |
| **`processor/`** | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔ | ⛔‡ | ⛔ |

> ✅ = reached  — = not applicable  ⛔ = **NEVER REACHED**
> † = `mcp.HealthResponse` DTO imported but `mcp.NewServer`/`HandleToolCall` never called
> ‡ = processor module has `NewFactory()` but no binary in this repo imports it

---

## Executable 1: `cmd/api-server`

> [main.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/cmd/api-server/main.go)

```mermaid
graph TD
    M["main()"] --> S1["authz.ValidateStartupConfig()"]
    M --> S2["zap.NewProduction()"]
    M --> S3["telemetry.InitOTelSDK('api-server')"]
    M --> S4["clickhouse.NewClient(127.0.0.1:9000)"]
    S4 -->|success| S5["clickhouse.NewHealthRepository(conn)"]
    S4 -->|fail| S5b["healthRepo = nil (mock mode)"]
    S5 --> S6["rest.NewServer(logger, healthRepo)"]
    S5b --> S6
    S6 --> DI

    subgraph DI["Dependency Injection (NewServer L66-78)"]
        DI1["remediation.NewValidator()"]
        DI2["remediation.NewGenerator()"]
        DI3["clickhouse.NewReplayRepository(conn)"]
        DI4["engine.NewEngine(replayRepo)"]
        DI4 --> DI4a["engine.NewBehaviorBuilder()"]
        DI4 --> DI4b["engine.NewDecisionBuilder()"]
        DI4 --> DI4c["engine.NewRootCauseBuilder()"]
    end

    S6 --> SRV["server.Start(':8080')"]

    subgraph SRV["HTTP Server (chi router)"]
        MW1["middleware.RequestID"]
        MW2["middleware.RealIP"]
        MW3["rateLimitMiddleware"]
        MW4["corsMiddleware"]
        MW5["metricsMiddleware → telemetry.ApiRequestsTotal"]
        MW6["tracingMiddleware → otel.Tracer"]
    end

    SRV --> INFRA
    SRV --> TENANT
    SRV --> REM
    SRV --> AGENT

    subgraph INFRA["Infrastructure (no auth)"]
        I1["/metrics → promhttp.Handler()"]
        I2["/swagger/* → httpSwagger"]
        I3["/healthz → 200 ok"]
        I4["/readyz → readyzHandler"]
    end

    subgraph TENANT["Tenant Routes (oidcAuthMiddleware)"]
        T1["/health → GetTenantHealth"]
        T2["/simulate → SimulateFailure"]
        T3["/issues → GetTenantIssues"]
        T4["/agents → GetAgentTraces"]
        T5["/coverage → GetCoverage"]
        T6["/traces/orphans → GetTracesOrphans"]
        T7["/config GET → HandleTenantConfigGet"]
        T8["/config PUT → HandleTenantConfigPut"]
        T9["/behavior → handleBehaviorGraph"]
        T10["/root-cause → GetTenantRootCause"]
    end

    subgraph REM["Remediation"]
        R1["/api/v1/remediation/apply → ApplyRemediation"]
    end

    subgraph AGENT["Agent Trace Routes (oidcAuthMiddleware)"]
        A1["/api/agents/{id}/traces/{tid}/behavior → GetBehaviorGraph"]
        A2["/api/agents/{id}/traces/{tid}/decisions → GetDecisionGraph"]
        A3["/api/agents/{id}/traces/{tid}/root-cause → GetRootCause"]
    end
```

### Route → Repository → ClickHouse Table Mapping

| Route | Handler | Repository Method | ClickHouse Table | Status |
|---|---|---|---|---|
| `/health` | `GetTenantHealth` | `HealthRepo.QueryHealthMetrics` | `cardinality_signal`, `orphan_signal`, `coverage_signal`, `tenant_config` | ✅ Live |
| `/health` | `GetTenantHealth` | `HealthRepo.GetTenantWeights` | `tenant_config` | ✅ Live |
| `/health` | `GetTenantHealth` | `generator.Generate` + `validator.Validate` | — (in-memory) | ✅ Live |
| `/issues` | `GetTenantIssues` | — (hardcoded JSON) | — | ⚠️ Mock only |
| `/agents` | `GetAgentTraces` | `HealthRepo.QueryAgentTraces` | `signoz_traces.signoz_index_v2` (fallback: mock) | ⚠️ Fallback mock |
| `/coverage` | `GetCoverage` | — (hardcoded JSON) | — | ⚠️ Mock only |
| `/traces/orphans` | `GetTracesOrphans` | — (hardcoded JSON) | — | ⚠️ Mock only |
| `/config GET` | `HandleTenantConfigGet` | `HealthRepo.GetTenantWeights` | `tenant_config` | ✅ Live |
| `/config PUT` | `HandleTenantConfigPut` | `HealthRepo.SaveTenantConfig` | `tenant_config` | ✅ Live |
| `/behavior` | `handleBehaviorGraph` | `engine.GenerateBehaviorGraph` → `ReplayRepo.GetRecentReplays` | `telemetryhealth_trace_index_spans` | ✅ Live |
| `/root-cause` | `GetTenantRootCause` | `engine.GenerateRootCause` → `ReplayRepo.GetReplay` | `telemetryhealth_trace_index_spans` | ✅ Live |
| `/simulate` | `SimulateFailure` | `simulator.NewSimulator` → gRPC to ingest-gateway | — (sends OTLP) | ✅ Live |
| `/remediation/apply` | `ApplyRemediation` | `HealthRepo.LogRemediationEvent` | `remediation_event` | ✅ Live |
| `/agents/.../behavior` | `GetBehaviorGraph` | `HealthRepo.QuerySpansByTraceID` → `behavior.NewEngine().Reconstruct` | `signoz_traces.signoz_index_v2` (fallback: mock) | ⚠️ Fallback mock |
| `/agents/.../decisions` | `GetDecisionGraph` | spans → `behavior.Reconstruct` → `decision.Reconstruct` | `signoz_traces.signoz_index_v2` (fallback: mock) | ⚠️ Fallback mock |
| `/agents/.../root-cause` | `GetRootCause` | spans → `behavior` → `decision` → `rootcause.Analyze` | `signoz_traces.signoz_index_v2` (fallback: mock) | ⚠️ Fallback mock |

---

## Executable 2: `cmd/ingest-gateway`

> [main.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/cmd/ingest-gateway/main.go)

```mermaid
graph TD
    M["main()"] --> S1["INSECURE_DEV_MODE auto-set"]
    M --> S2["zap.NewProduction()"]
    M --> S3["telemetry.InitOTelSDK('ingest-gateway')"]
    M --> S4["kafka.EnsureTopics(brokers)"]
    S4 --> S4a["Creates: telemetry.cardinality, telemetry.orphan, telemetry.coverage"]
    S4 --> S4b["⛔ MISSING: telemetry.rawspan NOT created"]
    M --> S5["kafka.NewProducer(brokers)"]
    S5 --> S5a["4 writers: cardinality, orphan, coverage, rawspan"]
    M --> S6["ingest.NewServer(logger, producer)"]
    S6 --> S7["authz.TenantAuthInterceptor()"]
    S6 --> SRV

    subgraph SRV["gRPC Server (:4317)"]
        G1["ptraceotlp.RegisterGRPCServer(receiver)"]
        G2["pmetricotlp.RegisterGRPCServer(metricsReceiver)"]
        G3["plogotlp.RegisterGRPCServer(logsReceiver)"]
    end

    M --> METRICS["Prometheus metrics server (:9094)"]

    G1 --> TE["receiver.Export()"]
    TE --> K1["producer.PublishOrphan → telemetry.orphan"]
    TE --> K2["publishCardinalityEvents → producer.PublishCardinality → telemetry.cardinality"]
    TE --> K3["producer.PublishRawSpan → telemetry.rawspan"]
    TE --> P1["telemetry.IngestedSpansTotal.Inc()"]

    G2 --> ME["metricsReceiver.Export()"]
    ME --> K4["producer.PublishCoverage → telemetry.coverage"]

    G3 --> LE["logsReceiver.Export()"]
    LE --> K5["producer.PublishCoverage → telemetry.coverage"]
```

### Kafka Topics Produced

| Topic | Producer Method | Source Signal |
|---|---|---|
| `telemetry.cardinality` | `PublishCardinality` | Per-attribute-key from trace spans |
| `telemetry.orphan` | `PublishOrphan` | Per-span structural tuple (trace/span/parent) |
| `telemetry.coverage` | `PublishCoverage` | Per-service heartbeat from metrics & logs |
| `telemetry.rawspan` | `PublishRawSpan` | Full span tuple for trace index |

---

## Executable 3: `cmd/worker`

> [main.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/cmd/worker/main.go)

```mermaid
graph TD
    M["main()"] --> S1["zap.NewProduction()"]
    M --> S2["telemetry.InitOTelSDK('stream-worker')"]
    M --> S3["kafka.EnsureTopics(brokers)"]
    M --> S4["clickhouse.NewClient(127.0.0.1:9000)"]
    S4 -->|fail| FATAL["logger.Fatal — exits process"]
    S4 -->|success| S5["kafka.NewWorkerSet(brokers, chClient)"]
    S5 --> RUN["workers.Run(ctx)"]

    subgraph RUN["WorkerSet.Run() — 4 goroutines"]
        W1["runCardinalityWorker"]
        W2["runOrphanWorker"]
        W3["runCoverageWorker"]
        W4["runRawSpanWorker"]
    end

    W1 --> C1["Consume telemetry.cardinality"]
    C1 --> CH1["INSERT INTO cardinality_signal<br/>(tenant_id, service, attribute_key, window_start, unique_estimate)"]

    W2 --> C2["Consume telemetry.orphan"]
    C2 --> CH2["INSERT INTO orphan_signal<br/>(tenant_id, trace_id, span_id, parent_span_id, collector_id, detected_at)"]

    W3 --> C3["Consume telemetry.coverage"]
    C3 --> CH3["INSERT INTO coverage_signal<br/>(tenant_id, service, last_seen_at, baseline_expected)"]
    CH3 --> CH3b["⚠️ MISSING columns: environment, grace_period_seconds"]

    W4 --> C4["Consume telemetry.rawspan"]
    C4 --> CH4["INSERT INTO telemetryhealth_trace_index_spans<br/>(trace_id, span_id, parent_span_id, service_name,<br/>operation_name, start_time, end_time, status, attributes, tenant_id)"]

    M --> METRICS["Prometheus metrics server (:9091)"]
```

### ClickHouse Tables Written

| Table | Writer | Columns Written | Schema Columns | Mismatch |
|---|---|---|---|---|
| `cardinality_signal` | `runCardinalityWorker` | 5 | 6 (`hll_sketch` not written) | ⚠️ `hll_sketch` dead |
| `orphan_signal` | `runOrphanWorker` | 6 | 6 | ✅ |
| `coverage_signal` | `runCoverageWorker` | 4 | 6 | ⚠️ `environment`, `grace_period_seconds` missing |
| `telemetryhealth_trace_index_spans` | `runRawSpanWorker` | 10 | 10 | ✅ |

---

## Executable 4: `cmd/init-db`

> [main.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/cmd/init-db/main.go)

```mermaid
graph TD
    M["main()"] --> S1["sql.Open('clickhouse', '127.0.0.1:9000')"]
    M --> S2["zap.NewDevelopment()"]
    M --> S3["clickhouse.NewSchema(db, logger)"]
    S3 --> S4["schema.InitSchema()"]

    subgraph S4["InitSchema() — 9 DDL statements"]
        T0["CREATE DATABASE telemetry_health"]
        T1["CREATE TABLE cardinality_signal"]
        T2["CREATE TABLE orphan_signal"]
        T3["CREATE TABLE coverage_signal"]
        T4["CREATE TABLE health_score ⛔"]
        T5["CREATE TABLE remediation_event"]
        T6["CREATE TABLE alert_event ⛔"]
        T7["CREATE TABLE tenant_config"]
        T8["CREATE TABLE telemetryhealth_trace_index_spans"]
    end
```

> ⛔ `health_score` and `alert_event` tables are created but **never read or written** by any runtime path.

---

## Executable 5: `cmd/seeder`

> [main.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/cmd/seeder/main.go)

```mermaid
graph TD
    M["main()"] --> S1["flag.Parse() — host, tenant UUID"]
    M --> S2["clickhouse.Open(host)"]
    M --> CARD["INSERT INTO cardinality_signal<br/>(tenant_id, service, attribute_key, window_start, unique_estimate)"]
    M --> ORPH["INSERT INTO orphan_signal<br/>(tenant_id, trace_id, span_id, parent_span_id, collector_id, detected_at)"]
    M --> COV["INSERT INTO coverage_signal<br/>(tenant_id, service, last_seen_at, baseline_expected)"]
```

> Standalone utility. Directly inserts into ClickHouse. No Kafka, no REST, no dependency injection.

---

## Executable 6: `cmd/simulator`

> [main.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/cmd/simulator/main.go)

```mermaid
graph TD
    M["main()"] --> S1["flag.Parse() — scenario, tenant, endpoint"]
    M --> S2["simulator.NewSimulator(logger, endpoint)"]
    S2 --> SC{"scenario switch"}
    SC -->|high_cardinality| HC["sim.InjectHighCardinality()"]
    SC -->|dropped_spans| DS["sim.InjectDroppedSpans()"]
    HC --> GRPC["gRPC dial → ingest-gateway:4317"]
    DS --> GRPC
    GRPC --> OTLP["ptraceotlp.Export(traces)"]
```

> Sends OTLP traces to the ingest-gateway. No direct ClickHouse or Kafka access.

---

## Executable 7: `cmd/e2e-test`

> [main.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/cmd/e2e-test/main.go)

```mermaid
graph TD
    M["main()"] --> S1["flag.Parse() — gateway addr, tenant UUID"]
    M --> S2["grpc.NewClient(gateway)"]
    S2 --> T1["ptraceotlp.Export(traces) — 5 spans"]
    S2 --> T2["pmetricotlp.Export(metrics) — 1 gauge"]
    T1 --> OUT["Prints ✓ Traces sent"]
    T2 --> OUT2["Prints ✓ Metrics sent"]
```

> End-to-end test client. Sends OTLP to ingest-gateway and prints results. No assertion on downstream ClickHouse.

---

## Frontend: `dashboard/`

> [main.tsx](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/dashboard/src/main.tsx) → [App.tsx](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/dashboard/src/App.tsx)

```mermaid
graph TD
    ENTRY["main.tsx → createRoot()"] --> EB["ErrorBoundary"]
    EB --> APP["App"]

    APP --> FETCH["fetch /api/v1/tenant/{id}/health<br/>(every 20s, App.tsx:166)"]

    APP --> V1["Overview"]
    APP --> V2["Cardinality"]
    APP --> V3["TraceChains"]
    APP --> V4["Coverage"]
    APP --> V5["Remediation"]
    APP --> V6["AgentTraces"]
    APP --> V7["DigitalTwin"]

    V1 --> API1["useTenantData('issues')"]
    V1 --> RCG["RootCauseGraph"]
    RCG --> API_RC["useTenantData('root-cause?issue_id=...')"]
    V3 --> API3["useTenantData('traces/orphans')"]
    V4 --> API4["useTenantData('coverage')"]
    V5 --> API5["fetch POST /api/v1/remediation/apply"]
    V6 --> API6["useTenantData('agents')"]
    V7 --> API7["useTenantData('behavior')"]
```

### Dashboard → API Server Route Mapping

| Dashboard Component | API Call via `useTenantData` | Matched Server Route | Handler |
|---|---|---|---|
| `App.tsx` | `GET /api/v1/tenant/{id}/health` | ✅ `/health` | `GetTenantHealth` |
| `Overview` | `GET /api/v1/tenant/{id}/issues` | ✅ `/issues` | `GetTenantIssues` |
| `Overview → RootCauseGraph` | `GET /api/v1/tenant/{id}/root-cause?issue_id=...` | ✅ `/root-cause` | `GetTenantRootCause` |
| `TraceChains` | `GET /api/v1/tenant/{id}/traces/orphans` | ✅ `/traces/orphans` | `GetTracesOrphans` |
| `Coverage` | `GET /api/v1/tenant/{id}/coverage` | ✅ `/coverage` | `GetCoverage` |
| `AgentTraces` | `GET /api/v1/tenant/{id}/agents` | ✅ `/agents` | `GetAgentTraces` |
| `DigitalTwin` | `GET /api/v1/tenant/{id}/behavior` | ✅ `/behavior` | `handleBehaviorGraph` |
| `Remediation` | `POST /api/v1/remediation/apply` | ✅ `/remediation/apply` | `ApplyRemediation` |

### Registered Routes NOT Called by Dashboard

| Route | Handler | Evidence |
|---|---|---|
| `POST /api/v1/tenant/{id}/simulate` | `SimulateFailure` | ⛔ No dashboard component calls `/simulate` |
| `GET /api/v1/tenant/{id}/config` | `HandleTenantConfigGet` | ⛔ No dashboard component calls `/config` |
| `PUT /api/v1/tenant/{id}/config` | `HandleTenantConfigPut` | ⛔ No dashboard component calls `/config` |
| `GET /api/agents/{id}/traces/{tid}/behavior` | `GetBehaviorGraph` | ⛔ No dashboard component calls agent-trace routes |
| `GET /api/agents/{id}/traces/{tid}/decisions` | `GetDecisionGraph` | ⛔ No dashboard component calls agent-trace routes |
| `GET /api/agents/{id}/traces/{tid}/root-cause` | `GetRootCause` | ⛔ No dashboard component calls agent-trace routes |

---

## Library Module: `processor/`

> [factory.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/processor/factory.go)

```
processor.NewFactory()
    ↓
    No binary in this repository imports it.
    No cmd/ executable references the processor Go module.
    No otelcol-builder config exists.
    ⛔ NEVER REACHED from any executable.
```

Has working unit tests (`cardinality/`, `failopen/`, `tracechain/`, `coverage/`) but is a library-only module.

---

## Standalone Tool: `tools/docs-bot/`

> [main.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/tools/docs-bot/main.go)

Separate Go module (`go.mod`). Contains `changelog.go`, `commitlog.go`, `status.go`. Has tests.

```
docs-bot main()
    ↓
    Standalone CLI tool for documentation generation.
    Does NOT import control-plane packages.
    Does NOT connect to Kafka, ClickHouse, or the dashboard.
    Isolated utility.
```

---

## ⛔ NEVER-REACHED Subsystems — Detailed Evidence

### 1. `internal/alerting` — Completely Dead

> Files: [signoz_bridge.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/alerting/signoz_bridge.go), [slack_bridge.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/alerting/slack_bridge.go), [pagerduty_bridge.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/alerting/pagerduty_bridge.go)

- **Constructors:** `NewSigNozBridge`, `NewSlackBridge`, `NewPagerDutyBridge`
- **Interface:** `AlertBridge` with `FireAlert` method
- **Proof of absence:** No `cmd/` binary, no REST handler, no worker, and no other internal package imports `internal/alerting`. Zero call sites outside the package itself.
- **Broken chain:** Worker writes to ClickHouse → stops. No code converts health changes → `AlertPayload` → bridge → `FireAlert`.

### 2. `internal/streaming` — Test-Only, Not Wired

> Files: [cardinality_job.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/streaming/cardinality_job.go), [tracechain_job.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/streaming/tracechain_job.go), [coverage_job.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/streaming/coverage_job.go), [healthscore_job.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/streaming/healthscore_job.go), [ai_health_job.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/streaming/ai_health_job.go)

- **Constructors:** `NewCardinalityJob`, `NewTraceChainJob`, `NewCoverageJob`, `NewHealthScoreJob`, `NewAIAgentHealthJob`
- **Proof of absence:** `cmd/worker/main.go:63` calls `kafka.NewWorkerSet`, which calls `runCardinalityWorker`/`runOrphanWorker`/`runCoverageWorker`/`runRawSpanWorker` in [workers.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/kafka/workers.go#L31-L46). Those functions do direct ClickHouse batch inserts. They never instantiate `streaming.New*Job`.
- **`NewAIAgentHealthJob`** has zero callers even in tests. Completely orphaned.
- **Broken chain:** Production Kafka worker → direct batch insert → ClickHouse. The `streaming` jobs (with HLL merging, trace chain correlation, health score computation) are bypassed entirely.

### 3. `internal/mcp` Server & Tools — No Transport

> Files: [server.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/mcp/server.go), [tools.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/mcp/tools.go)

- **Functions:** `NewServer`, `NewToolset`, `HandleToolCall`
- **Proof of absence:** No `cmd/mcp-server` binary exists. No REST route starts an MCP server. No `cmd/*` binary imports `mcp.NewServer` or `mcp.NewToolset`. The REST server imports `mcp` but only uses `mcp.HealthResponse` and `mcp.MetricsPayload` as DTO structs (response models), not the server/tools functions.
- **Broken chain:** In-process tool handlers exist but have no stdio, HTTP, SSE, or websocket transport.

### 4. `internal/mcp` Client — Never Called

> File: [client.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/mcp/client.go)

- **Function:** `QueryAgentTraces` (returns `[SIMULATED]` data), `InjectTraceContext`
- **Proof of absence:** Dashboard agent traces path is: `AgentTraces.tsx` → `useTenantData('agents')` → `GET /api/v1/tenant/{id}/agents` → `GetAgentTraces` → `HealthRepository.QueryAgentTraces`. This goes to `HealthRepository`, not `mcp.QueryAgentTraces`.
- `InjectTraceContext` is tested but has no runtime caller.

### 5. `processor/` Module — No Collector Binary

- **Entrypoint:** `processor.NewFactory()` creates a valid OTel Collector processor factory.
- **Proof of absence:** No Go binary in `control-plane/cmd/` imports `github.com/frag2win/TelemetryHealth/processor`. No `otelcol-builder` config exists. The processor module is a separate `go.mod`. It has unit tests but is unreachable from any executable in this repository.

---

## ClickHouse Tables: Created vs. Used

| Table | Created By | Written By | Read By | Status |
|---|---|---|---|---|
| `cardinality_signal` | `init-db` | `worker`, `seeder` | `api-server (QueryHealthMetrics)` | ✅ Active |
| `orphan_signal` | `init-db` | `worker`, `seeder` | `api-server (QueryHealthMetrics)` | ✅ Active |
| `coverage_signal` | `init-db` | `worker`, `seeder` | `api-server (QueryHealthMetrics)` | ✅ Active (⚠️ column mismatch) |
| `telemetryhealth_trace_index_spans` | `init-db` | `worker` | `api-server (ReplayRepo, QuerySpansByTraceID)` | ✅ Active |
| `tenant_config` | `init-db` | `api-server (SaveTenantConfig)` | `api-server (GetTenantWeights)` | ✅ Active |
| `remediation_event` | `init-db` | `api-server (LogRemediationEvent)` | — | ⚠️ Write-only audit |
| **`health_score`** | `init-db` | ⛔ Never | ⛔ Never | ⛔ **Dead table** |
| **`alert_event`** | `init-db` | ⛔ Never | ⛔ Never | ⛔ **Dead table** |

---

## Kafka Topics: Produced vs. Consumed vs. Bootstrapped

| Topic | Produced By | Consumed By | Bootstrapped (EnsureTopics) | Status |
|---|---|---|---|---|
| `telemetry.cardinality` | `ingest-gateway` | `worker` | ✅ | ✅ Full path |
| `telemetry.orphan` | `ingest-gateway` | `worker` | ✅ | ✅ Full path |
| `telemetry.coverage` | `ingest-gateway` | `worker` | ✅ | ✅ Full path |
| `telemetry.rawspan` | `ingest-gateway` | `worker` | ⛔ **NOT bootstrapped** | ⚠️ Works only with Kafka auto-create |

---

## Dashboard Assets: Reachable vs. Dead

| Asset | Imported By | Status |
|---|---|---|
| `src/assets/hero.png` | Nothing | ⛔ Dead |
| `src/assets/react.svg` | Nothing | ⛔ Dead |
| `src/assets/vite.svg` | Nothing | ⛔ Dead |

---

## End-to-End Data Flow (Actual Runtime Path)

The actual working pipeline when all services are running:

```
OTLP Client (collector/simulator/e2e-test)
    │
    │ gRPC :4317
    ▼
┌──────────────────────────┐
│  cmd/ingest-gateway      │
│  ├─ authz.TenantAuth     │
│  ├─ receiver.Export()    │
│  │  ├─ PublishOrphan     ├──→ Kafka: telemetry.orphan
│  │  ├─ PublishCardinality├──→ Kafka: telemetry.cardinality
│  │  └─ PublishRawSpan   ├──→ Kafka: telemetry.rawspan
│  ├─ metricsReceiver     │
│  │  └─ PublishCoverage  ├──→ Kafka: telemetry.coverage
│  └─ logsReceiver        │
│     └─ PublishCoverage  ├──→ Kafka: telemetry.coverage
└──────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────┐
│  cmd/worker                              │
│  ├─ runCardinalityWorker                 │
│  │  └─ INSERT → cardinality_signal       │
│  ├─ runOrphanWorker                      │
│  │  └─ INSERT → orphan_signal            │
│  ├─ runCoverageWorker                    │
│  │  └─ INSERT → coverage_signal          │
│  └─ runRawSpanWorker                     │
│     └─ INSERT → trace_index_spans        │
└──────────────────────────────────────────┘
                               │
                               ▼
                          ClickHouse
                               │
                               ▼
┌──────────────────────────────────────────┐
│  cmd/api-server                          │
│  ├─ HealthRepo.QueryHealthMetrics        │
│  │  └─ SELECT cardinality/orphan/coverage│
│  ├─ HealthRepo.QueryAgentTraces          │
│  │  └─ SELECT signoz_traces (fallback)   │
│  ├─ ReplayRepo.GetRecentReplays          │
│  │  └─ SELECT trace_index_spans          │
│  └─ REST routes → JSON responses         │
└──────────────────────────────────────────┘
                               │
                               │ HTTP :8080
                               ▼
┌──────────────────────────────────────────┐
│  dashboard (React, Vite)                 │
│  ├─ App.tsx polls /health every 20s      │
│  ├─ Overview → /issues, /root-cause      │
│  ├─ TraceChains → /traces/orphans        │
│  ├─ Coverage → /coverage                 │
│  ├─ AgentTraces → /agents                │
│  ├─ DigitalTwin → /behavior              │
│  └─ Remediation → POST /remediation/apply│
└──────────────────────────────────────────┘
```

### Subsystems Outside This Path (Never Reached)

```
⛔ internal/alerting        — No caller from any executable
⛔ internal/streaming       — Bypassed by direct worker batch inserts
⛔ internal/mcp (server)    — No transport, no binary
⛔ internal/mcp (client)    — HealthRepo used instead
⛔ processor/               — No collector binary imports it
⛔ health_score table       — Created, never used
⛔ alert_event table        — Created, never used
⛔ hll_sketch column        — In schema, never written or read
⛔ signoz_implementations/  — Legacy SQL, never executed
⛔ sdk-clients/             — Demo docs, not runtime
⛔ dashboard assets         — hero.png, react.svg, vite.svg unused
```
