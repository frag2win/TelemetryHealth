# TelemetryHealth Production Readiness Audit

## Summary
- **Total Findings:** 24
- **Critical:** 6
- **High:** 9
- **Medium:** 6
- **Low:** 3

This repository currently functions as a highly polished, deterministic **demo environment** rather than an enterprise-grade production platform. To graduate to production readiness, significant decoupling, configuration management, and dependency injection patterns must be applied.

---

## Detailed Findings

| File | Line | Category | Hardcoded Value | Why | Recommendation | Priority |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `cmd/api-server/main.go` | 81 | Configuration | `:8080` | Prevents port mapping/conflict resolution on host machines or k8s pods. | Bind to `PORT` environment variable or structured config (e.g. Viper). | Critical |
| `cmd/ingest-gateway/main.go` | 42 | Configuration | `[]string{"127.0.0.1:9092"}` | Kafka broker addresses are hardcoded. Fails entirely in Docker/Cloud. | Use `KAFKA_BROKERS` env var mapped to a split string array. | Critical |
| `cmd/worker/main.go` | 52 | Configuration | `[]string{"127.0.0.1:9000"}` | ClickHouse hosts are hardcoded. | Use `CH_HOST` and `CH_PORT` variables consistently. | Critical |
| `internal/api/rest/server.go` | 111 | Configuration | `"http://localhost:5173"` | CORS origin hardcoded to localhost dev server. Production browsers will block requests. | Pass an `AllowedOrigins` array into the Server config block. | Critical |
| `health_repository.go` | Multiple | ClickHouse | `signoz_traces.distributed_signoz_index_v2` | Hardcodes a specific distributed database/table layout. Breaks on single-node setups. | Abstract `DatabaseName` and `TableName` inside the repository struct. | Critical |
| `internal/api/rest/server.go` | 224 | Architecture | `os.Getenv("INSECURE_DEV_MODE") == "true"` | Auth bypass logic evaluated inline per HTTP request instead of via DI middleware. | Inject a `TokenValidator` interface at boot. | Critical |
| `TraceChains.tsx` | 68-150 | UI / Business Logic | `const nodes: ServiceNode[] = [...]` | Service topology, latency, throughput, and error rates are completely faked. | Wire up live websocket/REST topology generation backend endpoints. | High |
| `internal/alerting/signoz_bridge.go` | 77 | Configuration | `"http://localhost:9093/api/v2/alerts"` | Assumes SigNoz AlertManager runs locally on standard ports. | Read external alerting webhooks from tenant-level configuration. | High |
| `internal/alerting/poller.go` | 81 | Configuration | `DashboardLink: "http://localhost:5173"` | Pushes local links into alerts, making alerts useless to external users. | Define `DASHBOARD_PUBLIC_URL` at application startup. | High |
| `processor/tracechain/span_buffer.go`| 32 | Magic Number | `const maxTuples = 50000` | Limits memory buffer arbitrarily. OOMs on small containers, limits big ones. | Expose as `MaxSpanBufferSize` in worker config. | High |
| `processor/cardinality/tracker.go` | 69 | Magic Number | `const sketchMem = 16384` | HLL++ sketch size is hardcoded in function scope, controlling accuracy vs memory. | Make HLL precision configurable per tenant. | High |
| `processor/cardinality/tracker.go` | 11 | Business Logic | `defaultMaxKeysPerService = 100` | Hardcodes what constitutes an "explosion" universally for all tenants. | Load threshold dynamically via `TenantWeights` config. | High |
| `processor/failopen/circuit_breaker.go`| 38 | Business Logic | `limit = 5` | Breaker trips identically for fast vs slow systems. | Make breaker thresholds configurable per downstream dependency. | High |
| `internal/api/rest/server.go` | 780-820| Architecture | `Position: engine.NodePosition{X: xPos, Y: 120}` | API endpoints return UI-specific ReactFlow X/Y coordinates (Presentation Leak). | API should return semantic data; UI layout should occur in frontend/Dagre. | High |
| `Overview.tsx` / `Coverage.tsx` | 16, 78 | UI | `const fallbackIssues = [...]` | UI renders fake lists when data is loading or missing. | Implement proper empty states; don't fake data. | Medium |
| `processor/base_consumer.go` | 14 | Magic Number | `const healthExportQueueSize = 1000` | Hardcoded queue sizes can lead to unexpected backpressure behaviors. | Use dynamic queue sizes or config blocks. | Medium |
| `dashboard/vite.config.ts` | 10 | Configuration | `target: 'http://localhost:8080'` | Frontend dev server proxy assumes API is on 8080. | Accept acceptable convention, but document clearly. | Medium |
| `cmd/init-db/main.go` | 14 | Configuration | `clickhouse://127.0.0.1:9000?dial_timeout=5s` | Hardcodes DB initialization target. | Use standard DB URL env var parsing. | Medium |
| `internal/storage/mock/repository.go`| Multiple| AI | `"llm.model": "gpt-4o"` | Traces mock AI agent logic exclusively using OpenAI attributes. | N/A (acceptable for mock repos, but flag for real implementation). | Low |
| `mcp/main.go` | 33 | MCP | `[]string{"127.0.0.1:9000"}` | Hardcoded DB inside MCP bootstrap. | N/A. | Low |
| `App.tsx` | 51 | UI | `tenantId: '0000...'` | Tenant selector defaults to hardcoded zero-UUID. | Fetch actual tenants from backend. | Low |

---

## 🚀 Top Production Blockers

1. **Hardcoded IP Addresses:** The `127.0.0.1` strings peppered across the `cmd/` packages prevent this system from deploying into Kubernetes or Docker Swarm without source code modifications.
2. **Fake UI Topology:** The primary value proposition of the platform (the Trace-Chains visualization) is rendering completely hardcoded nodes and fake latency metrics, undermining the core product function.
3. **Database Table Assumptions:** Hardcoding `distributed_signoz_index_v2` breaks non-distributed ClickHouse deployments instantly.
4. **CORS Localhost Lock-in:** The API server enforces `http://localhost:5173` on preflight CORS requests, preventing access from any deployed frontend domains.

## ⚡ Quick Wins
- Replace all instances of `"127.0.0.1:9092"` and `"127.0.0.1:9000"` in the `cmd/` directory with `os.Getenv()` calls with simple defaults.
- Extract Magic Numbers (`maxTuples`, `sketchMem`, `healthExportQueueSize`) from inner function scopes and place them into a loaded `Config` struct passed down from `main.go`.
- Remove hardcoded ReactFlow `X`/`Y` coordinates from the backend `server.go` file and use a library like `dagre` in the React frontend to auto-layout the behavior graphs.

## 🏗 Long-term Architectural Improvements
- **Strict Dependency Injection:** Remove inline `os.Getenv()` checks embedded deep in middleware (like `INSECURE_DEV_MODE`). Load configuration once at boot in `main.go`, instantiate interfaces (e.g., `Authenticator`, `AlertNotifier`), and pass them into the handlers.
- **Tenant Configuration Policies:** Move thresholds like `defaultMaxKeysPerService` out of the processor pipeline and into a database-backed `TenantConfig` cache, allowing enterprise customers to tune their own health sensitivity without code redeploys.

## 📊 Score & Effort
- **Production-Readiness Score:** 25% (Suitable for local demo and Hackathons, unsafe for cloud deploys).
- **Estimated Effort to Fix:** ~3 Days. Refactoring the Go CLI bootstrappers and extracting Magic Numbers is trivial (1 day). Replacing the hardcoded UI topology in `TraceChains.tsx` with a live websocket stream processing actual OTel data requires moderate effort (2 days).
