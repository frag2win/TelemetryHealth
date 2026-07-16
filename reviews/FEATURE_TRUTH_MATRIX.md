# TelemetryHealth — Feature Truth Matrix

This matrix maps every functional capability claimed or suggested in the repository to its actual status in the executable source code. It relies strictly on static call graph analysis and direct source verification.

---

## 1. Feature Status Matrix

| Feature | Implemented | Compiled | Reachable | Persisted | User-Visible Output | Uses Mock Data | External Service Dependency |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **REST API Server** | **YES** | **YES** | REST, Dashboard, Tests | **YES** (ClickHouse) | **YES** (JSON payloads) | **YES** (fallback paths) | **YES** (ClickHouse) |
| **React Dashboard** | **YES** | **YES** | Dashboard | **NO** | **YES** (Browser UI) | **YES** (via mock API fallback) | **YES** (REST API) |
| **OTel Ingestion Gateway** | **YES** | **YES** | Simulator, External API | **YES** (Kafka/ClickHouse) | **NO** | **NO** | **YES** (Kafka/ClickHouse) |
| **ClickHouse Storage Client** | **YES** | **YES** | REST, Worker, Ingestion | **YES** | **NO** | **NO** | **YES** (ClickHouse TCP) |
| **Kafka Stream Aggregator** | **YES** | **YES** | Worker, Tests | **YES** (ClickHouse) | **NO** | **NO** | **YES** (Kafka, ClickHouse) |
| **AI Behavior Engine** | **YES** | **YES** | REST, Tests | **NO** (In-memory) | **YES** (via REST response) | **YES** (if SQL is empty) | **YES** (ClickHouse) |
| **AI Decision Engine** | **YES** | **YES** | REST, Tests | **NO** (In-memory) | **YES** (via REST response) | **YES** (if SQL is empty) | **YES** (ClickHouse) |
| **AI Root Cause Engine** | **YES** | **YES** | REST, Tests | **NO** (In-memory) | **YES** (via REST response) | **YES** (if SQL is empty) | **YES** (ClickHouse) |
| **Remediation Generator** | **YES** | **YES** | REST, Tests | **NO** | **YES** (YAML outputs) | **NO** | **NO** |
| **Slack Alert Bridge** | **YES** | **YES** | **NO** (Dead code) | **NO** | **NO** | **NO** | **YES** (Slack Web API) |
| **PagerDuty Alert Bridge** | **YES** | **YES** | **NO** (Dead code) | **NO** | **NO** | **NO** | **YES** (PagerDuty API v2) |
| **SigNoz Alert Bridge** | **YES** | **YES** | **NO** (Dead code) | **NO** | **NO** | **YES** (Logs only, no HTTP) | **NO** |
| **Model Context Protocol** | ⚠️ **PARTIAL** | **YES** | **NO** (Dead code) | **NO** | **NO** | **YES** (Simulated returns) | **NO** |

---

## 2. Exact Execution Paths from `main()`

### 1. REST API Server

* **Path Type:** Reachable (Active)
* **Execution Trace:**

  ```text
  cmd/api-server/main.go::main()
    ↳ api.NewServer(logger, healthRepo, generator, validator, ...) [server.go L54]
      ↳ server.Start(addr) [server.go L272]
        ↳ chi.NewRouter()
          ↳ r.Get("/api/v1/tenant/{tenant_id}/health", s.GetTenantHealth) [server.go L295]
            ↳ s.healthRepo.QueryHealthMetrics(ctx, tenantID) [server.go L368]
              ↳ database/sql execution -> ClickHouse TCP
            ↳ json.NewEncoder(w).Encode(resp) [server.go L414]
  ```

### 2. React Dashboard

* **Path Type:** Reachable (Active)
* **Execution Trace:**

  ```text
  dashboard/src/main.tsx (Vite Entrypoint)
    ↳ ReactDOM.createRoot()
      ↳ App.tsx (Root Router)
        ↳ Overview.tsx (Dashboard Component)
          ↳ fetch("/api/v1/tenant/acme-prod/health")
            ↳ Renders DOM elements (Gauge, Line charts)
  ```

### 3. OpenTelemetry Ingestion Gateway

* **Path Type:** Reachable (Active)
* **Execution Trace:**

  ```text
  cmd/ingest-gateway/main.go::main()
    ↳ ingest.NewGateway(logger, producer, clickhouseConn)
      ↳ gateway.Start(addr)
        ↳ grpc.NewServer()
          ↳ ptraceotlp.RegisterGRPCServer(grpcServer, gateway)
            ↳ gateway.Export(ctx, ExportTraceServiceRequest)
              ↳ gateway.producer.WriteMessages(ctx, KafkaMessage)
  ```

### 4. ClickHouse Storage Client

* **Path Type:** Reachable (Active)
* **Execution Trace:**

  ```text
  cmd/api-server/main.go::main()
    ↳ clickhouse.Open(Options) [client.go L22]
      ↳ sql.Open("clickhouse", ...)
        ↳ clickhouse.NewHealthRepository(conn, logger) [health_repository.go L98]
  ```

### 5. Kafka Stream Aggregator

* **Path Type:** Reachable (Active)
* **Execution Trace:**

  ```text
  cmd/worker/main.go::main()
    ↳ worker.Start()
      ↳ kafka.NewConsumer(logger, brokers, topic, ...)
        ↳ consumer.ConsumeMessages(ctx)
          ↳ workers.ProcessMessage(ctx, message)
            ↳ clickhouse.BatchInsert(ctx, metrics)
  ```

### 6. AI Behavior Reconstruction Engine

* **Path Type:** Reachable (Active, but contains mock fallback)
* **Execution Trace:**

  ```text
  cmd/api-server/main.go::main()
    ↳ server.Start(addr)
      ↳ r.Get("/api/agents/{agent_id}/traces/{trace_id}/behavior", s.GetBehaviorGraph) [server.go L314 / L710]
        ↳ s.healthRepo.QuerySpansByTraceID(ctx, traceID) [server.go L720]
          ↳ clickhouse.Query() -> returns 0 rows (if DB is empty)
            ↳ health_repository.go::generateMockSpans(traceID) [health_repository.go L350] (MOCK FALLBACK)
        ↳ behavior.NewEngine() [server.go L738]
          ↳ engine.Reconstruct(traceID, spans) [server.go L739]
            ↳ Renders Tree node relationships
        ↳ json.NewEncoder(w).Encode(graph) [server.go L747]
  ```

### 7. AI Decision Reconstruction Engine

* **Path Type:** Reachable (Active, but contains mock fallback)
* **Execution Trace:**

  ```text
  cmd/api-server/main.go::main()
    ↳ server.Start(addr)
      ↳ r.Get("/api/agents/{agent_id}/traces/{trace_id}/decisions", s.GetDecisionGraph) [server.go L315 / L751]
        ↳ s.healthRepo.QuerySpansByTraceID(ctx, traceID) [server.go L761]
        ↳ behavior.NewEngine().Reconstruct(traceID, spans) [server.go L778]
        ↳ decision.NewEngine() [server.go L785]
          ↳ decEngine.Reconstruct(behGraph) [server.go L786]
        ↳ json.NewEncoder(w).Encode(decGraph) [server.go L794]
  ```

### 8. AI Root Cause Inference Engine

* **Path Type:** Reachable (Active, but contains mock fallback)
* **Execution Trace:**

  ```text
  cmd/api-server/main.go::main()
    ↳ server.Start(addr)
      ↳ r.Get("/api/agents/{agent_id}/traces/{trace_id}/root-cause", s.GetRootCause) [server.go L316 / L798]
        ↳ s.healthRepo.QuerySpansByTraceID(ctx, traceID) [server.go L808]
        ↳ behavior.NewEngine().Reconstruct(traceID, spans) [server.go L825]
        ↳ decision.NewEngine().Reconstruct(behGraph) [server.go L833]
        ↳ rootcause.NewEngine() [server.go L840]
          ↳ rcEngine.Analyze(behGraph, decGraph) [server.go L841]
        ↳ json.NewEncoder(w).Encode(rc) [server.go L849]
  ```

### 9. Remediation Snippet Generator

* **Path Type:** Reachable (Active)
* **Execution Trace:**

  ```text
  cmd/api-server/main.go::main()
    ↳ server.Start(addr)
      ↳ r.Get("/api/v1/tenant/{tenant_id}/health", s.GetTenantHealth) [server.go L295]
        ↳ s.generator.Generate(ctx, issueType) [server.go L380]
          ↳ template.Must(template.New("remediation").Parse(...))
          ↳ bytes.Buffer write
          ↳ Returns formatted YAML string
  ```

### 10. Slack Alert Bridge

* **Path Type:** Unreachable (Dead Code)
* **Execution Trace:**
  * `internal/alerting/slack_bridge.go` defines `NewSlackBridge` and `FireAlert`.
  * No occurrences of `NewSlackBridge` exist in any `cmd/` entrypoint.
  * Zero incoming edges in the application call graph.

### 11. PagerDuty Alert Bridge

* **Path Type:** Unreachable (Dead Code)
* **Execution Trace:**
  * `internal/alerting/pagerduty_bridge.go` defines `NewPagerDutyBridge` and `FireAlert`.
  * No occurrences of `NewPagerDutyBridge` exist in any `cmd/` entrypoint.
  * Zero incoming edges in the application call graph.

### 12. SigNoz Alert Bridge

* **Path Type:** Unreachable (Dead Code)
* **Execution Trace:**
  * `internal/alerting/signoz_bridge.go` defines `NewSigNozBridge` and `FireAlert`.
  * No occurrences of `NewSigNozBridge` exist in any `cmd/` entrypoint.
  * Zero incoming edges in the application call graph.

### 13. Model Context Protocol (MCP) Server

* **Path Type:** Unreachable (Dead Code)
* **Execution Trace:**
  * `internal/mcp/server.go` defines `NewServer` and `HandleToolCall`.
  * No transport wrappers (stdio/HTTP) invoke this server.
  * No `main()` entrypoint imports the `internal/mcp` server package.
  * Zero incoming edges in the application call graph.
