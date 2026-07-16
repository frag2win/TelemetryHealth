## Person A – Core Intelligence Engines

**Focus:** Reconstruction \& root-cause logic for AI agents.

### 1) Behavior Reconstruction Engine

**Files / areas:**

- `control-plane/internal/behavior/` (or equivalent package)
- Core types in `control-plane/pkg/models/` (e.g. `BehaviorNode`, `BehaviorGraph`)
- Processor hooks in `processor/` that emit behavior-relevant metadata

**Tasks:**

- Define `BehaviorGraph` and `BehaviorNode` models (inputs, outputs, relationships).
- Implement logic that:
    - Consumes OTel spans/traces from AI agents.
    - Groups spans into logical “steps” (tool calls, LLM calls, DB calls, etc.).
    - Builds a behavior graph per trace / agent session.
- Add tests that:
    - Given a fixed set of spans, produce a deterministic behavior graph.
    - Cover normal flow and at least one failure pattern (e.g., missing step,timeout).

***

### 2) Decision Reconstruction Engine

**Files / areas:**

- `control-plane/internal/decision/`
- Shared models in `control-plane/pkg/models/` (e.g. `DecisionNode`, `DecisionGraph`)
- Any existing DRE doc → map to actual code structure

**Tasks:**

- Define `DecisionGraph` and `DecisionNode`:
    - Each node represents a decision point (which tool, which prompt, which branch).
    - Store inputs, chosen option, and alternatives (if available).
- Implement logic that:
    - Uses span attributes (e.g., `llm.tool_choice`, `llm.prompt`, `llm.response`) to infer decisions.
    - Links decisions to behavior nodes (1:1 or 1:many).
- Add tests that:
    - Reconstruct a simple multi-step agent workflow as a decision graph.
    - Validate that a “wrong tool call” scenario shows up clearly in the graph.

***

### 3) Root Cause Intelligence Engine

**Files / areas:**

- `control-plane/internal/rootcause/`
- Models: `RootCause`, `RootCauseRecord`, possibly in `control-plane/pkg/models/`
- Integration points with behavior \& decision engines

**Tasks:**

- Define `RootCause` schema:
    - Fields: `trace_id`, `agent_id`, `failure_type`, `description`, `evidence_span_ids`, `severity`, `timestamp`.
- Implement logic that:
    - Takes behavior + decision graphs + error signals.
    - Identifies the most likely failure point (e.g., tool call timeout, hallucinated output, token limit exceeded).
    - Produces a human-readable explanation and links to evidence spans.
- Add tests that:
    - Given a trace with a known failure pattern, produce the expected root cause record.
    - Cover at least 2–3 distinct failure types (latency, error, token issue).

***

### 4) Engine APIs (read-only for now)

**Files / areas:**

- `control-plane/internal/api/` or `handlers/`
- Handlers: `GetBehaviorGraph`, `GetDecisionGraph`, `GetRootCause`

**Tasks:**

- Implement HTTP/gRPC endpoints:
    - `GET /api/agents/{id}/traces/{trace_id}/behavior`
    - `GET /api/agents/{id}/traces/{trace_id}/decisions`
    - `GET /api/agents/{id}/traces/{trace_id}/root-cause`
- Ensure:
    - Request/response models match the domain model (coordinate with Person C).
    - Errors are clear (404 for missing trace, 400 for bad IDs, etc.).
- Add basic integration tests that call these endpoints against an in-memory or test DB.

**Deliverables for Person A:**

- Working engines that can:
    - Ingest traces → build behavior/decision graphs → compute root causes.
- APIs that return these structures for a given trace/agent.
- Tests proving correctness on sample traces.

***

## Person B – Telemetry Pipeline \& SigNoz Integration

**Focus:** Getting data in, storing it, and exposing it via SigNoz.

### 1) OTel Collector Pipeline

**Files / areas:**

- `processor/` (your OTel Collector processor)
- OTel Collector config (e.g., `otelcol-config.yaml` or similar)
- Any test data generators in `test/` or `sdk-clients/`

**Tasks:**

- Ensure AI agent traces (from your demo agent or test harness) are:
    - Instrumented with needed attributes (service name, operation, LLM-specific attributes).
    - Sent to your OTel Collector.
- In the processor:
    - Add/verify enrichment steps that tag traces for AI workloads (e.g., `telemetry.health.ai_agent=true`).
    - Ensure spans are forwarded correctly to SigNoz/ClickHouse.
- Write a small integration test or script that:
    - Sends a few test traces.
    - Confirms they appear in SigNoz/ClickHouse with expected attributes.

***

### 2) Custom OTel Metrics for TelemetryHealth

**Files / areas:**

- Metrics emitter in `control-plane/` or `processor/` (where you emit health metrics)
- Metric definitions (names, labels) documented somewhere central

**Tasks:**

- Define a small set of key metrics, for example:
    - `telemetryhealth_agent_health_score` (gauge)
    - `telemetryhealth_agent_token_burn_rate` (gauge / counter)
    - `telemetryhealth_agent_trace_error_count` (counter)
- Implement code that:
    - Computes these metrics from internal state / analysis results.
    - Exports them as OTel metrics to SigNoz.
- Verify in SigNoz UI that:
    - These metrics appear and can be queried.
    - They update as your system processes more traces.

***

### 3) SigNoz Dashboards \& Alerts

**Files / areas:**

- `signoz_implementations/` (dashboards, queries, configs)
- `alerts/` (alert rules YAMLs or JSONs)
- SigNoz Query Builder queries (exported or scripted)

**Tasks:**

- Create at least one core dashboard that shows:
    - Agent health score over time.
    - Token burn rate per agent/service.
    - Error count / failure rate.
    - A panel that can be filtered by `service.name` or `agent_id`.
- Implement alerts, for example:
    - Health score drops below threshold.
    - Token burn rate spikes above X per minute.
    - Error rate increases beyond baseline.
- Export/save:
    - Dashboard configs (JSON or whatever SigNoz uses).
    - Alert rule definitions.
- Verify end-to-end:
    - Run a test scenario that degrades health.
    - Confirm dashboard updates and alert fires.

***

### 4) SigNoz Field Requirements \& Reproducibility

**Files / areas:**

- `casting.yaml`, `casting.yaml.lock`
- Any Foundry-related docs or scripts
- Deployment notes (internal, not final submission docs)

**Tasks:**

- Verify that:
    - SigNoz deployment is done via Foundry (not just raw Docker).
    - `casting.yaml` and `casting.yaml.lock` fully describe the SigNoz + MCP setup.
- If needed:
    - Re-run Foundry install to regenerate a clean lock file.
    - Document (internally) the exact commands used so the deployment is reproducible.
- Ensure:
    - The MCP server is part of the deployment and accessible to your code.

**Deliverables for Person B:**

- A working OTel pipeline from agents → collector → SigNoz.
- Custom OTel metrics visible in SigNoz.
- At least one meaningful dashboard + alert rules.
- Verified, reproducible SigNoz deployment via Foundry.

***

## Person C – Architecture, APIs, and Runbooks

**Focus:** System design, API contracts, and developer-facing documentation.

### 1) Domain Model \& Data Models

**Files / areas:**

- `docs/DOMAIN_MODEL.md` (or equivalent)
- Actual Go/TS/Python model definitions in `control-plane/pkg/models/` and/or shared libs

**Tasks:**

- Define/clean up core entities:
    - `Agent`, `Trace`, `Span`, `BehaviorGraph`, `BehaviorNode`, `DecisionGraph`, `DecisionNode`, `RootCause`, `Remediation`, etc.
- Ensure:
    - Field names and types are consistent between docs and code.
    - Relationships are clear (e.g., Trace has many Spans; BehaviorGraph belongs to Trace).
- Update `DOMAIN_MODEL.md` to reflect the current implementation.
- Share this model with Person A and B as the single source of truth.

***

### 2) System \& Storage Architecture

**Files / areas:**

- `docs/SYSTEM_ARCHITECTURE.md`
- `docs/STORAGE_ARCHITECTURE.md`
- Actual DB schemas (ClickHouse tables, migrations, etc.)

**Tasks:**

- Produce/update a clear system diagram (can be ASCII or drawn then embedded):
    - AI agent → OTel Collector (with TelemetryHealth processor) → Control Plane → SigNoz/ClickHouse → Dashboard (later).
- Define:
    - Key components and their responsibilities.
    - Data flow for traces, metrics, logs.
- In storage architecture:
    - Document ClickHouse tables:
        - Traces/spans storage.
        - Behavior/decision/root-cause tables.
        - Metrics tables (if custom).
    - Include indexing and retention strategy at a high level.
- Ensure these docs match what Person B has implemented in SigNoz/ClickHouse.

***

### 3) API Design \& Contracts

**Files / areas:**

- `docs/API_DESIGN.md`
- Actual API handlers in `control-plane/internal/api/` (or similar)
- Shared request/response types

**Tasks:**

- Define and document:
    - All current and planned API endpoints (even if some are not fully implemented yet).
    - Request/response schemas (JSON structures).
    - Error formats and status codes.
- Make sure:
    - Person A’s engine APIs match the documented contracts.
    - Future frontend work (later phase) can rely on these docs.
- Add examples:
    - Sample requests/responses for:
        - Fetching behavior graph.
        - Fetching decision graph.
        - Fetching root cause.
        - Listing agents / traces.

***

### 4) Developer Runbooks

**Files / areas:**

- `docs/runbooks/` (multiple files)

**Tasks:**
Create at least these runbooks:

1. **Getting Started (Local Dev)**
    - How to run SigNoz locally (via Foundry or Docker, as per your setup).
    - How to run OTel Collector with your processor.
    - How to run the control-plane service.
    - How to send test traces.
2. **Adding a New Detector / Engine**
    - Where to add new detection logic.
    - How to register it in the pipeline.
    - How to expose its results via API.
3. **Telemetry Flow End-to-End**
    - High-level but precise description of:
        - How traces are generated.
        - How they are processed.
        - How they are stored and queried.
    - Include key config files and entry points.
4. **SigNoz Integration Guide (Internal)**
    - How metrics are emitted.
    - How dashboards and alerts are managed.
    - How to reproduce the SigNoz deployment.

**Deliverables for Person C:**

- Domain model, system architecture, and storage architecture docs that match reality.
- Clear API design docs that Person A’s endpoints follow.
- Practical runbooks that let a new developer get the system running and extend it.

***

## Minimal Cross-Person Coordination Points

To avoid conflict, agree on these once and treat them as shared contracts:

1. **Glossary \& Naming**
    - One shared list of terms: `Agent`, `BehaviorGraph`, `DecisionGraph`, `RootCause`, etc.
    - No renaming without updating:
        - Domain model doc.
        - Code types.
        - API docs.
2. **API Contracts**
    - Person C owns the API doc; Person A implements to that spec.
    - Any change in request/response shape must:
        - Update `API_DESIGN.md`.
        - Be communicated to anyone consuming the API (even if just internal).
3. **Telemetry Schema**
    - Person B owns the core OTel attribute conventions (e.g., `llm.*`, `agent.*`).
    - Person A uses those attributes to build graphs and root causes.
    - Any new important attribute must be:
        - Added to the shared schema doc (can live in `docs/` or `API_DESIGN.md`).

***