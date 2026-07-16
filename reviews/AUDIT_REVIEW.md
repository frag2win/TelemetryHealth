# TelemetryHealth Repository Audit

Role: Principal Software Engineer / Hackathon Judge  
Date: 2026-07-16  
Scope: Source-code-only verification. README and docs were treated as claims, not truth.

## Executive Verdict

TelemetryHealth has meaningful implementation work: Go control-plane services compile, the React dashboard builds, the processor module has focused unit tests, and there are real packages for behavior, decision, root-cause, remediation, ingestion, Kafka workers, ClickHouse storage, and dashboard views.

However, several headline claims are only partially true or demo-only. The most serious problems are:

- The README tells users to run the ingest gateway but claims it starts the REST API.
- MCP is implemented as in-process structs, but no reachable MCP server transport or binary is wired.
- Dashboard and backend paths silently fall back to mock/simulated data, making demos look healthier than the real pipeline.
- ClickHouse schema and worker inserts have runtime mismatches.
- Private keys and Terraform state are committed.
- There are duplicate intelligence engines (`internal/engine` vs `internal/behavior`/`decision`/`rootcause`) with different models and execution paths.
- Coverage claims in the README are not reproducible from the repository as tested.

Feature completion should be scored as "partial" for most architecture claims, not complete.

## Feature Verification Matrix

| Feature | Verdict | Evidence |
|---|---:|---|
| Clean Architecture | Partial | Some separation exists, but REST server constructs concrete repos, generators, validators, graph engines directly in `control-plane/internal/api/rest/server.go:66-78`; duplicate engine families exist. |
| Repository Pattern | Partial | `HealthRepository` and `ClickhouseReplayRepository` exist, but most code depends on concrete ClickHouse types; only `internal/engine` uses a `ReplayRepository` interface. |
| BRE | Partial | `internal/behavior` reconstructs graphs and is reachable through `/api/agents/{agent_id}/traces/{trace_id}/behavior`; tenant graph path uses different `internal/engine` builder. Mock spans are returned when real spans are absent. |
| BIE | Weak / Mostly Documentation | `AIAgentHealthJob` exists in `internal/streaming/ai_health_job.go`, but no production worker instantiates it. Search found only constructor/method definitions. |
| DRE | Partial | `internal/decision` has tests and is reachable through agent trace endpoint; not unified with tenant graph engine. |
| RCIE | Partial | `internal/rootcause` has tests and is reachable through agent trace endpoint; tenant root-cause graph uses separate `internal/engine` path and default graph fallbacks. |
| Agent Replay | Partial / Demo-biased | Raw spans are published and stored; replay repo exists. Missing full replay session lifecycle/API. Benchmark traces are injected by `traceID` prefix. |
| Auto-Remediation | Partial | Generator and structural validator exist; `ApplyRemediation` logs/audits but does not actually apply remediation to collectors/SigNoz. |
| Benchmark Framework | Minimal | `GetBenchmarkScenario` returns deterministic events and is triggered by `benchmark-` trace IDs; no benchmark runner/results API as docs claim. |
| MCP Server | Not Complete | `internal/mcp/server.go` has a `HandleToolCall` function, but no server transport or route starts it. `mcp/client.go` explicitly returns simulated data. |
| SigNoz Integration | Partial | SigNoz dashboard JSON and ClickHouse query attempts exist, but integration is mostly import/query conventions and docs; fallback mocks dominate UI and agent traces. |

## 1. Build Verification

### Finding 1.1 - Control plane, dashboard, docs-bot build/test pass in basic mode

- Severity: Low
- File: `control-plane/`, `dashboard/`, `tools/docs-bot/`
- Evidence:
  - `cd control-plane && go test ./...` passed.
  - `cd dashboard && npm run build` passed and produced Vite output.
  - `cd tools/docs-bot && go test ./...` passed.
- Root Cause: The baseline build is reasonably maintained.
- Suggested Fix: Keep these commands in CI and add integration tests for ClickHouse/Kafka paths.

### Finding 1.2 - Processor test command fails with default Go cache in this environment

- Severity: Medium
- File: `processor/`
- Evidence:
  - Initial `go test ./...` failed with `Access is denied` under `C:\Users\sunanda.AMFIIND\AppData\Local\go-build`.
  - Rerun with `GOCACHE=C:\Users\sunanda.AMFIIND\Desktop\SHUBHAM PROJECT\TelemetryHealth_\.gocache` passed.
- Root Cause: Build depends on user-profile cache write permissions. This is fragile in sandboxed/CI environments.
- Suggested Fix: Document or configure a workspace-local cache in CI scripts, or ensure the default cache path is writable.

### Finding 1.3 - Coverage claims are not reproducible

- Severity: Medium
- File: `README.md:237-250`
- Evidence:
  - README claims `processor/cardinality: 93.3%`, `processor/failopen: 93.9%`, `control-plane/authz: 100%`.
  - `processor go test ./... -cover` observed `cardinality 87.5%`, `failopen 97.3%`, `tracechain 48.6%`, processor root `0.0%`.
  - `control-plane go test ./... -cover` failed locally with missing `covdata` / package errors even though plain tests pass.
- Root Cause: Documentation contains stale or non-reproducible coverage numbers; Go toolchain/module settings also look unstable.
- Suggested Fix: Generate coverage in CI, publish the exact command and artifact, and remove static coverage claims from README.

### Finding 1.4 - README Go version claim conflicts with module

- Severity: Medium
- File: `README.md`, `control-plane/go.mod:3`
- Evidence:
  - README badge/prerequisite says Go `1.22+`.
  - `control-plane/go.mod` declares `go 1.26.3`.
- Root Cause: Docs were not updated after module/toolchain changes.
- Suggested Fix: Pin the supported Go version in all modules and docs; use a real released Go version if this repo is intended to build outside a custom toolchain.

### Finding 1.5 - Dashboard lint has warnings

- Severity: Low
- File: `dashboard/src/components/Shared.tsx`, `dashboard/src/components/views/AgentTraces.tsx`, `dashboard/src/components/views/Overview.tsx`
- Evidence:
  - `npm run lint` completed with warnings for Fast Refresh export shape and React hook dependencies.
- Root Cause: UI code prioritizes demo completion over strict React hygiene.
- Suggested Fix: Split shared hooks from component files and fix hook dependency arrays.

## 2. Runtime Verification

### Finding 2.1 - README run command starts the wrong service

- Severity: High
- File: `README.md:159-167`, `control-plane/cmd/ingest-gateway/main.go:63-84`, `control-plane/cmd/api-server/main.go:62-68`
- Evidence:
  - README says "Start the Go Control Plane API" then runs `go run ./cmd/ingest-gateway`.
  - It claims REST API starts on `http://localhost:8080`.
  - Source proves `ingest-gateway` starts gRPC on `:4317` and metrics on `:9094`.
  - REST API is started by `cmd/api-server`, which calls `server.Start(":8080")`.
- Root Cause: Documentation conflates ingest gateway and REST API binaries.
- Suggested Fix: Update README to run `go run ./cmd/api-server` for REST and separately run `cmd/ingest-gateway` for OTLP gRPC.

### Finding 2.2 - API server silently falls back to mock data when ClickHouse is down

- Severity: High
- File: `control-plane/cmd/api-server/main.go:44-60`, `dashboard/src/App.tsx:176-185`
- Evidence:
  - API logs "ClickHouse unavailable, using mock data" and continues with `healthRepo == nil`.
  - Dashboard catches backend failures and displays simulated fallback data with `version: v1.1.0-mock`.
- Root Cause: Demo-friendly fallback is mixed into production runtime paths.
- Suggested Fix: Make mock mode explicit via `DEMO_MODE=true`; fail readiness and show a hard UI warning when backing services are unavailable.

### Finding 2.3 - Worker cannot run without ClickHouse, unlike API mock mode

- Severity: Medium
- File: `control-plane/cmd/worker/main.go:48-59`
- Evidence:
  - Worker calls `logger.Fatal("clickhouse connect failed")` if ClickHouse is unavailable.
  - API server continues without ClickHouse.
- Root Cause: Inconsistent runtime failure policy between API and ingestion/worker services.
- Suggested Fix: Align service readiness semantics: either fail fast everywhere in production or require explicit mock mode everywhere.

### Finding 2.4 - Coverage worker insert does not match schema

- Severity: Critical
- File: `control-plane/internal/storage/clickhouse/schema.go:56-64`, `control-plane/internal/kafka/workers.go:123-134`
- Evidence:
  - Schema defines `coverage_signal` columns: `tenant_id`, `service`, `environment`, `last_seen_at`, `baseline_expected`, `grace_period_seconds`.
  - Worker insert writes only `(tenant_id, service, last_seen_at, baseline_expected)`.
  - `environment` and `grace_period_seconds` have no defaults in the DDL.
- Root Cause: Schema and writer evolved independently; no integration test covers real ClickHouse inserts.
- Suggested Fix: Add defaults to schema or write all required columns; add a ClickHouse integration test for every worker insert.

### Finding 2.5 - Tenant validation conflicts with ClickHouse UUID query parameters

- Severity: High
- File: `control-plane/internal/api/rest/server.go:88-95`, `control-plane/internal/storage/clickhouse/health_repository.go:49-55`
- Evidence:
  - REST accepts slugs like `acme-prod`, `tenant-alpha`.
  - Health queries bind `tenant_id` as `{tenant_id:UUID}`.
  - Non-UUID slugs will fail against real ClickHouse despite passing API validation.
- Root Cause: API validation was demo-friendly, while storage schema expects UUID.
- Suggested Fix: Use UUID tenant IDs end to end, or change schema/query types to `String/LowCardinality(String)` consistently.

### Finding 2.6 - mTLS claim is weaker than runtime default

- Severity: High
- File: `control-plane/cmd/ingest-gateway/main.go:20-23`, `control-plane/internal/authz/tenant_verifier.go:63-72`, `README.md:229-233`
- Evidence:
  - In non-production, `INSECURE_DEV_MODE=true` is automatically set.
  - In dev mode, tenant verification is bypassed.
  - README states "mTLS Everywhere" and "requires mutual TLS for all incoming OTLP connections."
- Root Cause: Development bypass is documented as production behavior.
- Suggested Fix: README must explicitly distinguish dev mode from production; require explicit opt-in for insecure mode.

## 3. Dead Code

### Finding 3.1 - MCP server is not reachable

- Severity: High
- File: `control-plane/internal/mcp/server.go`, `control-plane/internal/mcp/tools.go`, `control-plane/internal/mcp/client.go`
- Evidence:
  - Search found `NewToolset`, `NewServer`, and `HandleToolCall` only inside the MCP package.
  - No binary, REST route, stdio server, SSE server, or HTTP handler instantiates `mcp.NewServer`.
  - `mcp/client.go:24-42` says it is mocking the MCP query and returns `[SIMULATED]` traces.
- Root Cause: MCP logic was implemented as isolated structs without transport integration.
- Suggested Fix: Add a real MCP server entrypoint and integration test that invokes tools over the same protocol SigNoz agents will use.

### Finding 3.2 - AI agent health job is not wired into worker runtime

- Severity: Medium
- File: `control-plane/internal/streaming/ai_health_job.go`
- Evidence:
  - Search found only `NewAIAgentHealthJob` and `ProcessSpan` definitions.
  - `control-plane/internal/kafka/workers.go` starts cardinality, orphan, coverage, and rawspan workers only.
- Root Cause: AI-specific metrics were added as an isolated in-memory job and never connected to Kafka/raw span processing.
- Suggested Fix: Wire it into raw span processing or remove the claim until it produces persisted/queryable metrics.

### Finding 3.3 - Benchmark framework is only a trace-ID-triggered fixture

- Severity: Medium
- File: `control-plane/internal/engine/benchmark.go`, `control-plane/internal/engine/graph.go:78-82`
- Evidence:
  - `GetBenchmarkScenario` returns hardcoded events.
  - It is used only when `traceID` starts with `benchmark-`.
  - No `/api/v1/benchmark/run` or results API exists in `server.go`.
- Root Cause: A demo fixture is presented as a framework.
- Suggested Fix: Add benchmark jobs, persisted results, scoring assertions, and endpoints; otherwise rename this to demo scenario fixtures.

## 4. Architecture Drift

### Finding 4.1 - Duplicate intelligence engines with different models

- Severity: High
- File: `control-plane/internal/engine/*`, `control-plane/internal/behavior/engine.go`, `control-plane/internal/decision/engine.go`, `control-plane/internal/rootcause/engine.go`, `control-plane/internal/api/rest/server.go:66-78`, `server.go:738-841`
- Evidence:
  - Tenant graph routes use `internal/engine.NewEngine(replayRepo)`.
  - Agent trace routes instantiate `behavior.NewEngine`, `decision.NewEngine`, and `rootcause.NewEngine` directly.
  - These packages use different graph/domain types and fallback behaviors.
- Root Cause: Parallel implementations were added for dashboard graphing and AI-agent intelligence without consolidation.
- Suggested Fix: Define one domain model and one orchestration path; make REST handlers thin adapters around application services.

### Finding 4.2 - Repository pattern is inconsistent

- Severity: Medium
- File: `control-plane/internal/api/rest/server.go:66-78`, `control-plane/internal/storage/clickhouse/*.go`
- Evidence:
  - `Server` stores concrete `*clickhouse.HealthRepository`.
  - `NewServer` constructs `clickhouse.NewReplayRepository` internally.
  - Only `internal/engine` abstracts replay retrieval with an interface.
- Root Cause: Dependency inversion is partial; concrete infrastructure leaks into API/service layer.
- Suggested Fix: Introduce interfaces at application boundary and inject them from `cmd/*`.

### Finding 4.3 - Clean Architecture claim is overstated

- Severity: Medium
- File: `control-plane/internal/api/rest/server.go`
- Evidence:
  - REST layer handles routing, auth middleware, validation, remediation generation, data fallback, simulation, graph orchestration, and storage access.
- Root Cause: Handlers accumulated orchestration and infrastructure responsibilities.
- Suggested Fix: Split use cases into application services and keep handlers responsible for HTTP concerns only.

## 5. Documentation Drift

### Finding 5.1 - API docs list routes that do not exist

- Severity: High
- File: `docs/API_DESIGN.md:20-80`, `control-plane/internal/api/rest/server.go:292-317`
- Evidence:
  - Docs claim `/api/v1/tenant/{tenant_id}/agent/{id}/traces/{trace_id}/behavior`.
  - Actual route is `/api/agents/{agent_id}/traces/{trace_id}/behavior`, without `/api/v1/tenant/{tenant_id}`.
- Root Cause: API design document and implementation diverged.
- Suggested Fix: Generate API docs from source routes or update implementation to match documented contract.

### Finding 5.2 - README claims MCP server integration as complete

- Severity: High
- File: `README.md:218-225`, `control-plane/internal/mcp/server.go`
- Evidence:
  - README says TelemetryHealth implements an MCP server and exposes tools to SigNoz AI agents.
  - No reachable MCP transport is started by any binary.
- Root Cause: Internal tool handlers are documented as external integration.
- Suggested Fix: Downgrade README to "MCP toolset prototype" or wire a real MCP server.

### Finding 5.3 - Milestone table marks incomplete features complete

- Severity: Medium
- File: `README.md:266-275`
- Evidence:
  - M3 says Remediation & Hardening GA complete, but remediation does not apply configs.
  - M5 says Kafka integration complete, but schema/insert mismatch breaks coverage writes.
  - M2 says mTLS AuthZ complete, but dev runtime bypasses it by default.
- Root Cause: Milestone status tracks code presence rather than end-to-end verified behavior.
- Suggested Fix: Change milestone status to evidence-based definitions: builds, reachable, integration-tested, documented.

## 6. Technical Debt

### Finding 6.1 - Generated dashboard artifacts are tracked

- Severity: Low
- File: `dashboard/dist/*`, `.gitignore:26`
- Evidence:
  - `git ls-files dashboard/dist` returns 3 tracked files.
  - `.gitignore` excludes `dashboard/dist/`, but files already tracked remain in Git.
- Root Cause: Generated build output was committed before or despite ignore rules.
- Suggested Fix: Remove `dashboard/dist` from Git history/current index and let CI produce artifacts.

### Finding 6.2 - Untracked `dashboard/node_modules` exists in working tree

- Severity: Low
- File: `dashboard/node_modules/`, `.gitignore:23`
- Evidence:
  - Repository scans initially timed out/noised because `dashboard/node_modules` exists locally.
  - `git ls-files dashboard/node_modules` returns 0, so it is untracked but present.
- Root Cause: Local dependency directory is present in workspace.
- Suggested Fix: Keep it untracked; use `rg -g '!dashboard/node_modules/**'` for local audits and clean workspaces before submission.

### Finding 6.3 - Hardcoded infrastructure endpoints

- Severity: Medium
- File: `control-plane/cmd/api-server/main.go:47-50`, `control-plane/cmd/ingest-gateway/main.go:42`, `control-plane/cmd/worker/main.go:39-53`
- Evidence:
  - Kafka and ClickHouse addresses are hardcoded to `127.0.0.1`.
  - README/HOW_TO_RUN mention overrides, but source shown here does not read env vars for these binaries.
- Root Cause: Demo defaults are embedded into production entrypoints.
- Suggested Fix: Read `KAFKA_BROKERS`, `CLICKHOUSE_ADDR`, database/user/password from environment/config with sane defaults.

## 7. Security Issues

### Finding 7.1 - Private keys and Terraform state are committed

- Severity: Critical
- File: `control-plane/deployments/terraform/single-node/telemetry-health-key.pem`, `telemetry-health-key-v2.pem`, `terraform.tfstate`, `terraform.tfstate.*.backup`
- Evidence:
  - `git ls-files control-plane/deployments/terraform/single-node/*.pem control-plane/deployments/terraform/single-node/*.tfstate*` lists private key files and Terraform state files.
- Root Cause: Secrets/state were not excluded before commit.
- Suggested Fix: Remove from Git, rotate any associated credentials, add `*.pem`, `*.tfstate`, `*.tfstate.*` to `.gitignore`, and use remote encrypted Terraform state.

### Finding 7.2 - OIDC fallback allows structural JWT parsing without signature verification

- Severity: High
- File: `control-plane/internal/api/rest/server.go:210-267`
- Evidence:
  - Comments state production verifies JWKS when `OIDC_ISSUER` is set.
  - Fallback path performs structural JWT parse only.
- Root Cause: Auth middleware mixes production and non-production validation modes.
- Suggested Fix: Require signature verification unless explicit dev mode is enabled; fail closed otherwise.

### Finding 7.3 - Rate limiter map can grow forever

- Severity: Medium
- File: `control-plane/internal/api/rest/server.go:147-178`
- Evidence:
  - `visitors` map stores one limiter per IP with no eviction.
- Root Cause: Per-IP limiter has no TTL cleanup.
- Suggested Fix: Add periodic cleanup or use a bounded/expiring cache.

## 8. Performance Issues

### Finding 8.1 - Replay recent-trace query likely misuses `IN ($2)` with an array

- Severity: Medium
- File: `control-plane/internal/storage/clickhouse/replay_repository.go:64-70`
- Evidence:
  - Query uses `trace_id IN ($2)` and passes `traceIDs` as a single parameter.
  - ClickHouse drivers typically require array binding with `IN ?`/named array handling or expanded parameters depending on driver syntax.
- Root Cause: Query construction was not validated with an integration test.
- Suggested Fix: Add ClickHouse integration test for `GetRecentReplays`; use supported array parameter syntax.

### Finding 8.2 - Cardinality signal is not actually HLL in the worker write path

- Severity: Medium
- File: `control-plane/internal/storage/clickhouse/schema.go:30-37`, `control-plane/internal/kafka/workers.go:58-70`, `control-plane/internal/ingest/grpc_server.go:120-129`
- Evidence:
  - Schema has `hll_sketch AggregateFunction(uniqCombined, String)`.
  - Worker inserts `unique_estimate` only, not `hll_sketch`.
  - Ingest publishes `UniqueValues: 1` per observed attribute key.
- Root Cause: HLL design exists in schema/docs, but streaming path stores counts/estimates directly.
- Suggested Fix: Either store proper aggregate states or change schema/docs to reflect the implemented approximate counter.

### Finding 8.3 - Dashboard polling uses one AbortSignal across repeated interval fetches

- Severity: Low
- File: `dashboard/src/App.tsx:193-208`
- Evidence:
  - One `AbortController` is created per effect and its `signal` is reused for every 20-second interval fetch.
- Root Cause: Polling implementation is simple but not ideal for independently aborting overlapping requests.
- Suggested Fix: Create a new controller per fetch or ensure intervals cannot overlap.

## 9. Hackathon Risks

### Finding 9.1 - Demo can pass while real pipeline is broken

- Severity: Critical
- File: `dashboard/src/App.tsx:146-185`, `dashboard/src/components/Shared.tsx:35-56`, `control-plane/internal/storage/clickhouse/health_repository.go:347-353`
- Evidence:
  - UI fallback provides realistic mock dashboard data.
  - Shared hook loads fallback data on fetch failure.
  - Backend generates mock spans when ClickHouse returns no spans.
- Root Cause: Mock/demo path is transparent enough to mask integration failures.
- Suggested Fix: Put a persistent, prominent "DEMO DATA" banner on all fallback paths and make judges run an end-to-end smoke test with mocks disabled.

### Finding 9.2 - SigNoz integration may be judged as documentation-only

- Severity: High
- File: `docs/runbooks/signoz_integration.md`, `dashboard/signoz/*.json`, `control-plane/internal/storage/clickhouse/health_repository.go:138-183`
- Evidence:
  - There are dashboard JSON files and queries against `signoz_traces.signoz_index_v2`.
  - No tested deployment sync or SigNoz API integration was verified.
  - Agent traces fallback to hardcoded examples if query fails.
- Root Cause: Integration artifacts exist, but there is no demonstrated live SigNoz contract test.
- Suggested Fix: Add a reproducible SigNoz local stack test or scripted dashboard import/query verification.

### Finding 9.3 - Auto-remediation is advice, not remediation

- Severity: High
- File: `control-plane/internal/remediation/generator.go`, `control-plane/internal/remediation/validator.go`, `control-plane/internal/api/rest/server.go:479-533`
- Evidence:
  - Generator returns YAML snippets.
  - Validator only parses YAML and checks component allowlist.
  - Apply endpoint logs/audits; no collector config update or SigNoz action is performed.
- Root Cause: "Auto-remediation" naming overstates implementation.
- Suggested Fix: Rename to "Remediation Advisor" unless it can apply a patch through a controlled deployment path.

### Finding 9.4 - Agent replay is not a full replay product

- Severity: High
- File: `control-plane/internal/storage/clickhouse/replay_repository.go`, `control-plane/internal/engine/graph.go`, `docs/API_DESIGN.md`
- Evidence:
  - Raw span replay events and graph conversion exist.
  - No replay session APIs from docs (`/api/v1/replays`, timeline/export/results) are implemented in `server.go`.
  - Missing immutable replay session lifecycle.
- Root Cause: Replay graph prototype is documented as a complete replay engine.
- Suggested Fix: Implement replay session CRUD/read APIs, timeline, export, and persisted signatures, or narrow the claim.

## 10. Final Score

Score: 58 / 100

Breakdown:

- Buildability: 14 / 15
- Runtime reachability: 10 / 20
- Architecture integrity: 8 / 15
- Feature completeness: 10 / 25
- Tests: 6 / 10
- Security: 2 / 10
- Documentation accuracy: 3 / 5
- Demo polish: 5 / 5

Judge's final assessment:

TelemetryHealth is a strong hackathon prototype with a polished dashboard, real Go services, and several credible subsystem implementations. It is not yet a production-grade explainable intelligence layer for OpenTelemetry and SigNoz. The repo currently mixes real services, mock/demo fallbacks, aspirational docs, duplicate engine paths, and committed secrets. For judging, I would credit the team for breadth and presentation, but I would not mark MCP Server, full Agent Replay, Benchmark Framework, Clean Architecture, or Auto-Remediation as complete.

