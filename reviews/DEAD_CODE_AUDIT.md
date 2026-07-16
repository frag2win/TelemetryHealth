# TelemetryHealth Dead Code Audit

Date: 2026-07-16  
Scope: Entire repository, excluding generated/vendor directories for reachability checks (`dashboard/node_modules`, `dashboard/dist`).  
Definition used: code/artifacts present in the repository but not reachable from any repository-owned runtime entrypoint, route, worker, server, frontend render path, API call path, Kafka consumer path, or ClickHouse query/write path.

## Method

I traced execution from these concrete entrypoints:

- Go binaries: `control-plane/cmd/api-server`, `control-plane/cmd/ingest-gateway`, `control-plane/cmd/worker`, `control-plane/cmd/init-db`, `control-plane/cmd/seeder`, `control-plane/cmd/simulator`, `control-plane/cmd/e2e-test`.
- Processor module: `processor.NewFactory`, but only as a library entrypoint. There is no collector distribution in this repo that imports it.
- Dashboard entrypoint: `dashboard/src/main.tsx` -> `dashboard/src/App.tsx`.
- REST routes registered in `control-plane/internal/api/rest/server.go`.
- Kafka topics produced/consumed in `control-plane/internal/kafka`.
- ClickHouse tables created, inserted, and queried in `control-plane/internal/storage/clickhouse`, `control-plane/internal/kafka/workers.go`, and `control-plane/cmd/seeder`.
- MCP package call sites.

Dead-code categories:

- Planned but unwired: source has tests or clear intended design, but no runtime path.
- Experimental: demo/prototype code intentionally isolated.
- Legacy: stale artifact superseded by current implementation.
- Genuine dead code: no runtime/test/docs reason remains apparent.

## Summary

High-confidence dead or unwired areas:

- `control-plane/internal/alerting`: bridges are never instantiated or called.
- `control-plane/internal/streaming`: job implementations are test-only; production worker bypasses them.
- `control-plane/internal/mcp/server.go` and `tools.go`: in-process tool server is never instantiated by any binary or route.
- `control-plane/internal/mcp/client.go`: simulated SigNoz MCP client is never called.
- `processor` package: implemented as an OTel Collector processor library, but no collector binary/config in this repo imports it.
- `signoz_implementations/clickhouse_migration.sql`: legacy migration creates tables never used by the current code.
- ClickHouse tables `health_score` and `alert_event`: created but never queried or inserted by runtime code.
- REST routes `/simulate`, `/config`, `/api/agents/.../{behavior,decisions,root-cause}` are registered but not invoked by the dashboard.
- Dashboard assets `hero.png`, `react.svg`, and `vite.svg` are not imported or referenced.
- No Zustand store exists in source; the repository has no reachable Zustand usage.

## Complete Runtime Paths Checked

### API Server Path

`control-plane/cmd/api-server/main.go:62` constructs `rest.NewServer`; `main.go:67` starts `server.Start(":8080")`.  
`control-plane/internal/api/rest/server.go:272-326` registers REST routes.

Registered route path:

- `/api/v1/tenant/{tenant_id}/health` -> `GetTenantHealth`
- `/api/v1/tenant/{tenant_id}/simulate` -> `SimulateFailure`
- `/api/v1/tenant/{tenant_id}/issues` -> `GetTenantIssues`
- `/api/v1/tenant/{tenant_id}/agents` -> `GetAgentTraces`
- `/api/v1/tenant/{tenant_id}/coverage` -> `GetCoverage`
- `/api/v1/tenant/{tenant_id}/traces/orphans` -> `GetTracesOrphans`
- `/api/v1/tenant/{tenant_id}/config` -> `HandleTenantConfigGet/Put`
- `/api/v1/tenant/{tenant_id}/behavior` -> `handleBehaviorGraph`
- `/api/v1/tenant/{tenant_id}/root-cause` -> `GetTenantRootCause`
- `/api/v1/remediation/apply` -> `ApplyRemediation`
- `/api/agents/{agent_id}/traces/{trace_id}/behavior` -> `GetBehaviorGraph`
- `/api/agents/{agent_id}/traces/{trace_id}/decisions` -> `GetDecisionGraph`
- `/api/agents/{agent_id}/traces/{trace_id}/root-cause` -> `GetRootCause`

### Ingest Path

`control-plane/cmd/ingest-gateway/main.go:52` constructs `kafka.NewProducer`; `main.go:59` constructs `ingest.NewServer`; `main.go:66` starts gRPC on `:4317`.  
`control-plane/internal/ingest/grpc_server.go:243-245` registers OTLP traces, metrics, and logs gRPC receivers.

Produced topics:

- `telemetry.cardinality` via `PublishCardinality`
- `telemetry.orphan` via `PublishOrphan`
- `telemetry.coverage` via `PublishCoverage`
- `telemetry.rawspan` via `PublishRawSpan`

### Worker Path

`control-plane/cmd/worker/main.go:63` constructs `kafka.NewWorkerSet`; `main.go:88` calls `workers.Run`.  
`control-plane/internal/kafka/workers.go:31-46` starts four goroutines:

- `runCardinalityWorker`
- `runOrphanWorker`
- `runCoverageWorker`
- `runRawSpanWorker`

Consumed topics:

- `telemetry.cardinality`
- `telemetry.orphan`
- `telemetry.coverage`
- `telemetry.rawspan`

### Dashboard Path

`dashboard/src/main.tsx:76-78` renders `<App />`.  
`dashboard/src/App.tsx:410-416` conditionally renders all main views:

- `Overview`
- `Cardinality`
- `TraceChains`
- `Coverage`
- `Remediation`
- `AgentTraces`
- `DigitalTwin`

Dashboard API calls:

- `App.tsx:166` fetches `/api/v1/tenant/${selectedTenantId}/health`.
- `Shared.tsx:36-38` builds tenant-scoped API calls for views.
- `Remediation.tsx:348-352` posts `/api/v1/remediation/apply`.

## Dead / Unwired Items

### 1. Alerting Bridges Are Never Instantiated

- Category: Planned but unwired
- Files:
  - `control-plane/internal/alerting/signoz_bridge.go`
  - `control-plane/internal/alerting/slack_bridge.go`
  - `control-plane/internal/alerting/pagerduty_bridge.go`
- Evidence:
  - Constructors exist: `NewSigNozBridge`, `NewSlackBridge`, `NewPagerDutyBridge`.
  - Search found no calls to those constructors outside the alerting package.
  - `AlertBridge` interface is declared in `signoz_bridge.go:49-51`, but no runtime service accepts or stores that interface.
  - `FireAlert` methods exist but are never invoked by API, worker, ingest, or simulator code.
- Why it is dead:
  - Runtime path stops at worker writes to ClickHouse. No code converts health-score changes into `AlertPayload`, chooses a bridge, or calls `FireAlert`.
- Complete execution path proving absence:
  - `cmd/worker` -> `kafka.NewWorkerSet` -> `runCardinalityWorker/runOrphanWorker/runCoverageWorker/runRawSpanWorker` -> direct ClickHouse inserts.
  - No import of `control-plane/internal/alerting` appears in `cmd/*`, `internal/kafka`, `internal/streaming`, or `internal/api/rest`.
- Suggested action:
  - Wire alert emission into health score/issue detection or remove/downgrade the alerting bridge claim.

### 2. `internal/streaming` Jobs Are Test-Only, Not Production Workers

- Category: Planned but unwired
- Files:
  - `control-plane/internal/streaming/cardinality_job.go`
  - `control-plane/internal/streaming/tracechain_job.go`
  - `control-plane/internal/streaming/coverage_job.go`
  - `control-plane/internal/streaming/healthscore_job.go`
  - `control-plane/internal/streaming/ai_health_job.go`
- Evidence:
  - Constructors exist: `NewCardinalityJob`, `NewTraceChainJob`, `NewCoverageJob`, `NewHealthScoreJob`, `NewAIAgentHealthJob`.
  - Search found runtime calls only in tests for the first four, and no calls at all for `NewAIAgentHealthJob`.
  - Production worker path is `cmd/worker/main.go:63` -> `kafka.NewWorkerSet` -> `workers.Run`.
  - `workers.Run` starts functions in `control-plane/internal/kafka/workers.go:31-46`, not `internal/streaming` jobs.
- Why it is dead:
  - The production Kafka worker directly inserts raw event batches into ClickHouse. It does not instantiate the richer streaming jobs that implement HLL merging, bounded orphan correlation, sampling drift detection, or AI-agent health aggregation.
- Complete execution path proving absence:
  - `cmd/worker/main.go:88` -> `WorkerSet.Run`.
  - `WorkerSet.Run` -> `runCardinalityWorker` -> `PrepareBatch INSERT INTO telemetry_health.cardinality_signal`.
  - `WorkerSet.Run` -> `runOrphanWorker` -> `PrepareBatch INSERT INTO telemetry_health.orphan_signal`.
  - `WorkerSet.Run` -> `runCoverageWorker` -> `PrepareBatch INSERT INTO telemetry_health.coverage_signal`.
  - `WorkerSet.Run` -> `runRawSpanWorker` -> `PrepareBatch INSERT INTO telemetry_health.telemetryhealth_trace_index_spans`.
  - No call to `streaming.New*Job` in that path.
- Suggested action:
  - Either replace direct batch handlers with these job implementations, or mark the `streaming` package as prototype/test-only.

### 3. `AIAgentHealthJob` Is Never Used Anywhere

- Category: Planned but unwired
- File: `control-plane/internal/streaming/ai_health_job.go`
- Evidence:
  - `AIAgentHealth`, `AIAgentHealthJob`, `NewAIAgentHealthJob`, `ProcessSpan`, and `GetMetrics` are defined.
  - Search found no runtime or test call to `NewAIAgentHealthJob` or `ProcessSpan`.
- Why it is dead:
  - No raw span worker parses `llm.token_usage`, `llm.tool_name`, or `llm.tool_call.error` through this job. API agent traces are instead queried/faked in `HealthRepository.QueryAgentTraces`.
- Complete execution path proving absence:
  - Agent dashboard path: `App.tsx` -> `AgentTraces` -> `useTenantData('agents')` -> `/api/v1/tenant/{tenant_id}/agents` -> `GetAgentTraces` -> `HealthRepository.QueryAgentTraces`.
  - That path never touches `AIAgentHealthJob`.
- Suggested action:
  - Connect `AIAgentHealthJob` to `runRawSpanWorker`, persist results, and expose them through `/agents`; otherwise remove it.

### 4. MCP Server Is Not Reachable

- Category: Planned but unwired
- Files:
  - `control-plane/internal/mcp/server.go`
  - `control-plane/internal/mcp/tools.go`
- Evidence:
  - `mcp.NewServer(toolset)` and `HandleToolCall` exist.
  - Search found no call to `mcp.NewServer`, `mcp.NewToolset`, or `HandleToolCall` outside the MCP package.
  - `server.go` has no route for MCP transport, and no `cmd/mcp-server` binary exists.
- Why it is dead:
  - MCP tool handlers are plain in-process functions with no stdio, HTTP, SSE, websocket, or SigNoz MCP transport.
- Complete execution path proving absence:
  - API server path: `cmd/api-server` -> `rest.NewServer` -> `Start` registers HTTP routes. No MCP route is registered.
  - Worker/ingest paths do not import `internal/mcp`.
  - REST only imports MCP response DTOs in `server.go:28` and constructs `mcp.HealthResponse` in `GetTenantHealth`; this does not instantiate the MCP server.
- Suggested action:
  - Add a real MCP server binary/transport and integration test, or rename this package to `mcpdto`/`mcpprototype`.

### 5. Simulated SigNoz MCP Client Is Never Called

- Category: Experimental
- File: `control-plane/internal/mcp/client.go`
- Evidence:
  - `QueryAgentTraces` returns hardcoded `[SIMULATED]` traces.
  - Search found no call to `mcp.QueryAgentTraces`.
  - `InjectTraceContext` is tested via `TestInjectTraceContext`, but not used by runtime HTTP client code.
- Why it is dead:
  - The actual agent trace API uses `HealthRepository.QueryAgentTraces`, not `mcp.QueryAgentTraces`.
- Complete execution path proving absence:
  - Dashboard `AgentTraces` -> `/api/v1/tenant/{tenant_id}/agents` -> `GetAgentTraces` -> `s.healthRepo.QueryAgentTraces`.
  - No branch calls `mcp.QueryAgentTraces`.
- Suggested action:
  - Remove this mock client or replace `HealthRepository.QueryAgentTraces` with a real SigNoz/MCP client abstraction.

### 6. OTel Processor Package Is Not Wired Into a Collector Binary in This Repo

- Category: Planned but unwired
- Files:
  - `processor/factory.go`
  - `processor/base_consumer.go`
  - `processor/traces_consumer.go`
  - `processor/metrics_consumer.go`
  - `processor/logs_consumer.go`
- Evidence:
  - `processor.NewFactory` creates a valid OTel Collector processor factory.
  - Search found no import of `github.com/frag2win/TelemetryHealth/processor` from any repository-owned collector distribution or binary.
  - No `otelcol` builder config imports this module.
- Why it is dead within this repository:
  - It can be used by an external collector build, but this repository does not provide the executable path that loads it.
- Complete execution path proving absence:
  - Available Go binaries are under `control-plane/cmd/*`; none import `processor`.
  - The processor module has tests, but no runtime command.
- Suggested action:
  - Add an OTel Collector distribution or builder manifest that imports `processor.NewFactory`, or document it as a library-only module.

### 7. Raw Span Kafka Topic Is Produced and Consumed but Not Created by Topic Bootstrap

- Category: Genuine runtime gap
- Files:
  - `control-plane/internal/kafka/producer.go:13-17`
  - `control-plane/internal/kafka/admin.go:13-14`
  - `control-plane/internal/kafka/workers.go:150-184`
- Evidence:
  - `TopicRawSpan = "telemetry.rawspan"` exists and is produced by `PublishRawSpan`.
  - `runRawSpanWorker` consumes `TopicRawSpan`.
  - `EnsureTopics` creates only `{TopicCardinality, TopicOrphan, TopicCoverage}`.
- Why it is dead/broken:
  - If Kafka auto-topic creation is disabled, raw span production/consumption has no bootstrapped topic, so the replay graph pipeline cannot start from a clean environment.
- Complete execution path:
  - Ingest path publishes raw span: `grpc_server.go:84-95` -> `Producer.PublishRawSpan`.
  - Worker path consumes raw span: `workers.go:150-184`.
  - Bootstrap path: `cmd/ingest-gateway/main.go:47` and `cmd/worker/main.go:44` call `EnsureTopics`, which omits `TopicRawSpan`.
- Suggested action:
  - Add `TopicRawSpan` to `EnsureTopics`.

### 8. Legacy ClickHouse Migration Tables Are Unused

- Category: Legacy
- File: `signoz_implementations/clickhouse_migration.sql`
- Evidence:
  - Migration creates `telemetry_health.signal_metrics` and `telemetry_health.root_cause_records`.
  - Runtime code never queries or inserts either table.
  - Current schema is in `control-plane/internal/storage/clickhouse/schema.go`.
- Why it is dead:
  - No `cmd/init-db` path reads this SQL file. `init-db` uses `clickhouse.NewSchema(...).InitSchema()`.
- Complete execution path proving absence:
  - `cmd/init-db` -> `NewSchema` -> `InitSchema` -> DDL embedded in Go source.
  - No code opens or executes `signoz_implementations/clickhouse_migration.sql`.
- Suggested action:
  - Move this SQL file to historical docs or delete it to avoid false operational guidance.

### 9. `health_score` Table Is Created but Never Used

- Category: Planned but unwired
- File: `control-plane/internal/storage/clickhouse/schema.go:67-80`
- Evidence:
  - DDL creates `telemetry_health.health_score`.
  - Search found no `INSERT INTO telemetry_health.health_score` and no `FROM telemetry_health.health_score`.
  - Health score is calculated on demand in `HealthRepository.QueryHealthMetrics`, not persisted.
- Why it is dead:
  - Table has no writer or reader in runtime code.
- Complete execution path proving absence:
  - API health path: `GetTenantHealth` -> `HealthRepository.QueryHealthMetrics` -> queries cardinality/orphan/coverage tables -> `telemetry.CalculateHealthScore`.
  - No persisted `health_score` table involved.
- Suggested action:
  - Either persist computed scores from a worker or remove the table until needed.

### 10. `alert_event` Table Is Created but Never Used

- Category: Planned but unwired
- File: `control-plane/internal/storage/clickhouse/schema.go:101-115`
- Evidence:
  - DDL creates `telemetry_health.alert_event`.
  - Search found no runtime insert/query of `telemetry_health.alert_event`.
  - Alert bridges are also unwired.
- Why it is dead:
  - No alerting pipeline exists to populate the table.
- Complete execution path proving absence:
  - Worker only inserts cardinality, orphan, coverage, rawspan.
  - API only reads health/agents/spans/config and writes remediation events/config.
- Suggested action:
  - Implement alert event persistence when wiring alert bridges, or remove the table.

### 11. `hll_sketch` Column Is Dead in Current Worker Path

- Category: Architecture drift / genuine dead field
- File: `control-plane/internal/storage/clickhouse/schema.go:30-37`, `control-plane/internal/kafka/workers.go:58-70`
- Evidence:
  - `cardinality_signal` has `hll_sketch AggregateFunction(uniqCombined, String)`.
  - Worker inserts only `(tenant_id, service, attribute_key, window_start, unique_estimate)`.
  - No query reads `hll_sketch`.
- Why it is dead:
  - The implemented path writes scalar estimates, not aggregate sketches.
- Complete execution path:
  - Ingest publishes `UniqueValues: 1` per attribute key.
  - Worker writes `event.UniqueValues` into `unique_estimate`.
  - `HealthRepository.QueryHealthMetrics` reads `max(unique_estimate)`.
  - No code writes or reads `hll_sketch`.
- Suggested action:
  - Implement real aggregate-state insertion/querying or remove the column.

### 12. Registered API Routes Not Invoked by the Dashboard

- Category: Planned but unwired from UI
- Files:
  - `control-plane/internal/api/rest/server.go:292-317`
  - `dashboard/src/App.tsx`
  - `dashboard/src/components/views/*`
- Evidence:
  - Dashboard invokes tenant routes through `useTenantData`: `issues`, `agents`, `coverage`, `behavior`, `root-cause`, `traces/orphans`.
  - Dashboard posts `/api/v1/remediation/apply`.
  - Search found no dashboard call to `/api/v1/tenant/{tenant_id}/simulate`.
  - Search found no dashboard call to `/api/v1/tenant/{tenant_id}/config`.
  - Search found no dashboard call to `/api/agents/{agent_id}/traces/{trace_id}/behavior`, `/decisions`, or `/root-cause`.
- Why it is dead from product UI:
  - Routes are registered and testable manually, but they are not reachable from the shipped React dashboard workflows.
- Complete execution path proving absence:
  - UI render path is `main.tsx` -> `App.tsx` -> view components.
  - Fetch calls are only `App.tsx:166`, `Shared.tsx:38`, and `Remediation.tsx:348`.
  - No component constructs the `/simulate`, `/config`, or `/api/agents/...` URLs.
- Suggested action:
  - Add UI controls for simulation/config/deep trace drilldown or document them as API-only endpoints.

### 13. Agent Trace Deep-Dive Endpoints Are Registered but Mostly Shadowed by Tenant Graph Routes

- Category: Experimental
- File: `control-plane/internal/api/rest/server.go:312-317`, `server.go:710-849`
- Evidence:
  - Routes under `/api/agents/{agent_id}/traces/{trace_id}` exist.
  - Dashboard `AgentTraces` fetches only `/api/v1/tenant/{tenant_id}/agents`.
  - Overview root-cause graph calls tenant route `root-cause?issue_id=...`, not agent trace root-cause route.
- Why it is dead from application flow:
  - No UI state selects an agent trace and calls these endpoints.
- Complete execution path:
  - `AgentTraces` displays local Gantt spans derived from `/agents` response.
  - It does not call `GetBehaviorGraph`, `GetDecisionGraph`, or `GetRootCause`.
- Suggested action:
  - Add trace click-through to these endpoints or remove the parallel endpoint family.

### 14. Dashboard Assets Are Unused

- Category: Legacy / genuine dead assets
- Files:
  - `dashboard/src/assets/hero.png`
  - `dashboard/src/assets/react.svg`
  - `dashboard/src/assets/vite.svg`
- Evidence:
  - Asset directory contains all three files.
  - Search found no reference to `hero.png`, `react.svg`, `vite.svg`, or `assets/` from `dashboard/src`, `dashboard/public`, or `dashboard/index.html`.
- Why it is dead:
  - Vite default assets and a hero image remain in the source tree but are not imported by the built app.
- Complete execution path proving absence:
  - `main.tsx` imports `App.tsx` and CSS.
  - App and view components do not import any `src/assets/*` file.
- Suggested action:
  - Delete unused assets or wire `hero.png` into a real UI surface.

### 15. Zustand Store Usage Does Not Exist

- Category: Not present / no dead store found
- File: `dashboard/package.json`, `dashboard/src`
- Evidence:
  - `dashboard/package.json` does not list `zustand`.
  - Search found no `zustand`, `createStore`, or Zustand `create(...)` store usage in `dashboard/src`.
- Why this matters:
  - The dead-code request included "Zustand stores never used"; there are no Zustand stores to mark dead.
- Suggested action:
  - None, unless Zustand is intended; then add it deliberately.

### 16. SDK Client Directories Are Documentation/Demo Only

- Category: Experimental
- Files:
  - `sdk-clients/go/README.md`
  - `sdk-clients/python/README.md`
  - `sdk-clients/ai-agent-demo/agent.py`
- Evidence:
  - No Go, dashboard, or control-plane runtime imports or executes SDK client code.
  - `README.md` points users to the AI agent demo manually.
- Why it is dead from repository runtime:
  - These are standalone examples, not part of any build/test/deploy path observed in the repo.
- Complete execution path proving absence:
  - Go binaries under `control-plane/cmd/*` do not execute files under `sdk-clients`.
  - Dashboard does not import SDK files.
- Suggested action:
  - Keep as examples, but label them explicitly as demos and add a smoke test if they are submission-critical.

## Not Dead: Verified Reachable Items

These items may look unused at first glance but have a proven path:

- `GetTenantHealth`: registered at `/api/v1/tenant/{tenant_id}/health` and invoked by `dashboard/src/App.tsx:166`.
- `GetTenantIssues`: registered at `/issues` and invoked by `Overview` via `useTenantData('issues')`.
- `GetAgentTraces`: registered at `/agents` and invoked by `AgentTraces` via `useTenantData('agents')`.
- `GetCoverage`: registered at `/coverage` and invoked by `Coverage`.
- `GetTracesOrphans`: registered at `/traces/orphans` and invoked by `TraceChains`.
- `handleBehaviorGraph`: registered at `/behavior` and invoked by `DigitalTwin`.
- `GetTenantRootCause`: registered at `/root-cause` and invoked by `RootCauseGraph`.
- `ApplyRemediation`: registered at `/api/v1/remediation/apply` and invoked by `Remediation.tsx`.
- `RootCauseGraph`: not top-level navigation, but rendered inside `Overview` issue rows.
- `DigitalTwin`: rendered from App when active view is `topology`.
- Kafka topics `telemetry.cardinality`, `telemetry.orphan`, `telemetry.coverage`: produced by ingest and consumed by worker.
- Kafka topic `telemetry.rawspan`: produced and consumed, but topic bootstrap omission makes it operationally fragile rather than completely dead.

## Final Dead-Code Risk Rating

Risk: High

The repository has a lot of code that is useful as prototype material, but a significant portion is not wired into runtime execution paths. The highest-risk dead-code zones are the claimed platform integrations: alerting, MCP, advanced streaming jobs, benchmark/replay scaffolding, and SDK/demo assets. These areas create the appearance of a complete architecture while the actual runtime path is narrower:

`OTLP gRPC ingest -> Kafka producer -> Kafka worker direct ClickHouse inserts -> REST reads ClickHouse or mock data -> React dashboard`.

Everything outside that path should be treated as either demo-only or unwired until a service, route, worker, or UI action proves otherwise.

