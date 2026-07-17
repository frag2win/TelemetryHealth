# TelemetryHealth Final Execution Backlog

This document is the systematic execution plan for the final push before hackathon submission. Tasks are strictly prioritized by their immediate return on investment for the demo and judge confidence.

---

## 🚫 Don't Change Before Demo

To prevent regressions and avoid introducing last-minute risks, the following architectural elements **must not be refactored or modified** before submission:

| Component | Recommendation |
| :--- | :--- |
| **ClickHouse Schema** | Don't refactor before submission; the raw SQL queries already work perfectly for the demo. |
| **Kafka Ingest Gateway** | Leave as-is unless actively demonstrating it. Do not attempt to wire it into the default `api-server` flow. |
| **Vendor-Neutral Storage** | Postpone building generalized `SpanReader` abstractions until after the hackathon. |
| **Repository Interfaces** | Postpone broad redesigns of `HealthRepository` and `ReplayRepository` to avoid cascading build breaks. |

---

## 🔴 P0: Must Fix Before Submission
*Tasks that actively improve the demo, remove obvious architectural smells, or reduce demo failure risk.*

### RFC-001: Extract `generateMockSpans()` into `MockReplayRepository`
* **Priority:** P0
* **Demo Impact:** High (Protects the presentation layer from infrastructure leakage. Judges inspecting `server.go` will see clean Dependency Injection instead of hardcoded structs).
* **Estimated Effort:** 1.5 Hours
* **Acceptance Criteria (Done when):**
  - `server.go` no longer contains mock generation logic.
  - `internal/storage/mock/replay_repository.go` is created and implements `engine.ReplayRepository`.
  - `main.go` gracefully injects the mock repository if ClickHouse fails.
  - The API returns the identical JSON payloads under mock conditions.
  - Existing `go test ./...` passes.

### RFC-002: Add Benchmark Controls to Frontend
* **Priority:** P0
* **Demo Impact:** Very High (Exposes existing powerful backend simulation capabilities directly to the judges via the UI).
* **Estimated Effort:** 1 Hour
* **Acceptance Criteria (Done when):**
  - A "Run Benchmark Scenario" UI control (dropdown or button) is added to the React Flow dashboard header.
  - Selecting a scenario passes the `benchmark-*` trace ID down to the `RootCauseGraph` and `AgentTraces` components.
  - The frontend successfully renders the simulated prompt explosion trace.

### RFC-003: Update Architecture Documentation
* **Priority:** P0
* **Demo Impact:** High (Prevents judges from failing the project for "missing" features by setting accurate architectural expectations).
* **Estimated Effort:** 0.5 Hours
* **Acceptance Criteria (Done when):**
  - `smallfi.md` and `HOW_TO_RUN.md` explicitly state: *"TelemetryHealth operates as an analytical overlay on standard OpenTelemetry stores. Kafka/OTLP services are optional ingest prototypes."*
  - Any claims of active real-time streaming are scoped to the `cmd/worker` prototype only.

---

## 🟠 P1: Strongly Recommended
*Tasks that significantly improve codebase credibility but don't strictly change the happy-path demo.*

### RFC-004: Wire Alertmanager Bridge
* **Priority:** P1
* **Demo Impact:** Medium (Fulfills the "Automated Alerting" PRD claim if a judge asks to see the code, even if not explicitly demoed).
* **Estimated Effort:** 2 Hours
* **Acceptance Criteria (Done when):**
  - A `telemetry_poller` routine is created that periodically fetches health scores.
  - The poller triggers `FireAlert()` on the existing `SigNozBridge` when scores drop.
  - `main.go` initializes and starts the poller.

### RFC-005: Reduce `server.go` Complexity & Improve DI
* **Priority:** P1
* **Demo Impact:** Low (Purely code quality / judge confidence).
* **Estimated Effort:** 1.5 Hours
* **Acceptance Criteria (Done when):**
  - Routing logic is extracted into a dedicated `routes.go` or router initialization block.
  - Dependency Injection is fully respected (completed largely by RFC-001).

---

## 🔵 P2: Post-Hackathon
*Valuable engineering improvements, but too large of an architectural investment with zero demo ROI.*

### RFC-006: ClickHouse Abstraction & Vendor-Neutral Storage Layer
* **Priority:** P2
* **Demo Impact:** None
* **Estimated Effort:** 12+ Hours
* **Acceptance Criteria:** A generalized `SpanReader` interface supports SigNoz, Jaeger, and Datadog backend schemas seamlessly.

### RFC-007: Add MCP Resources & Prompts
* **Priority:** P2
* **Demo Impact:** None (The core MCP tools capability is already sufficient for a hackathon).
* **Estimated Effort:** 4 Hours
* **Acceptance Criteria:** Claude Desktop can successfully pull architecture markdown via `resources/read` and utilize pre-built diagnostic templates via `prompts/list`.

### RFC-008: Broad Repository Interface Redesign
* **Priority:** P2
* **Demo Impact:** None
* **Estimated Effort:** 8 Hours
* **Acceptance Criteria:** Storage and domain logic boundaries are perfectly segregated with independent data transfer objects (DTOs).
