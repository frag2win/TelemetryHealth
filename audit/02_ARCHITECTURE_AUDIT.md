# 02 — Architecture Audit

## System Architecture Overview

The system follows a **three-tier architecture**: OTel Processor → Control Plane → Dashboard.

```
┌─────────────────┐     gRPC/OTLP     ┌─────────────────┐
│  OTel Collector  │──────────────────▶│  Ingest Gateway  │
│   processor/     │                   │ cmd/ingest-gw    │
└─────────────────┘                   └────────┬─────────┘
                                               │ Kafka
                                      ┌────────▼─────────┐
                                      │  Stream Workers   │
                                      │  cmd/worker       │
                                      └────────┬─────────┘
                                               │ Batch Insert
                                      ┌────────▼─────────┐
                                      │    ClickHouse     │
                                      └────────┬─────────┘
                                               │ Query
                                      ┌────────▼─────────┐     REST JSON     ┌─────────────────┐
                                      │   API Server      │──────────────────▶│  React Dashboard │
                                      │  cmd/api-server   │                   │  dashboard/      │
                                      └────────┬─────────┘                   └─────────────────┘
                                               │ MCP/SSE
                                      ┌────────▼─────────┐
                                      │   MCP Server      │
                                      │  cmd/mcp-server   │
                                      └──────────────────┘
```

---

## SOLID Audit

### Single Responsibility Violations

| Severity | File | Issue |
|----------|------|-------|
| 🟠 High | `control-plane/internal/api/rest/server.go` (1134 lines) | Contains middleware definitions, handler implementations, helper functions, fallback data generators, ReactFlow graph transformations. Should be split into: `middleware.go`, `handlers.go`, `helpers.go`, `fallback.go` |
| 🟠 High | `control-plane/internal/storage/clickhouse/health_repository.go` (676 lines) | Contains pricing config, hallucination risk calculation, health queries, agent trace queries, tenant config, remediation logging, span queries, AND fallback mock data. Business logic mixed with data access |
| 🟡 Medium | `control-plane/internal/storage/mock/repository.go` (316 lines) | Implements BOTH `HealthRepository` AND `ReplayRepository`, AND contains extensive test fixture generation |

### Open/Closed Principle Violations

- **Remediation generator** uses string pattern matching (`strings.Contains`) instead of a strategy pattern. Adding a new issue type requires modifying the switch statement in `generator.go`.
- **Alert bridges** (SigNoz, PagerDuty, Slack) are individual files but there's no registry or plugin mechanism.

### Dependency Inversion Violations

- `cmd/api-server/main.go` imports `storage/mock` directly into the production binary. Mock implementations should only exist in test binaries.
- `server.go` imports `storage/mock` for fallback in `GetTenantReplay`.

---

## Package Dependency Audit

### Concerning Dependencies

| Import | Location | Issue |
|--------|----------|-------|
| `storage/mock` | `cmd/api-server/main.go`, `api/rest/server.go` | Mock code compiled into production binaries |
| `simulator` | `api/rest/server.go` | Simulation logic (sends gRPC to ingest gateway) callable from the REST API. API handler creates gRPC connections to itself. |
| `internal/mcp` | `api/rest/server.go` | REST server depends on MCP response types for JSON serialization |

### Circular Dependency Risk

No actual circular dependencies exist, but the dependency graph is overly coupled:
- `api/rest` → `{behavior, decision, engine, mcp, remediation, rootcause, simulator, storage, storage/mock, storage/signoz, telemetry, models}`
- This single package depends on 12 other packages — a god-package anti-pattern.

---

## API Design Audit

### REST API Issues

| Issue | Severity | Endpoint | Problem |
|-------|----------|----------|---------|
| Inconsistent tenant scoping | 🟡 Medium | `POST /api/v1/remediation/apply` | Not scoped under `/tenant/{tenant_id}/` unlike all other endpoints — takes tenant_id in body instead |
| Two different agent trace routes | 🟡 Medium | `/api/v1/tenant/{tenant_id}/agents` vs `/api/agents/{agent_id}/traces/{trace_id}/*` | Two completely different URL structures for agent-related endpoints |
| Hardcoded responses | 🟠 High | `GetTenantIssues`, `GetCoverage`, `GetTracesOrphans` | Return hardcoded JSON arrays — not connected to any data source |
| No pagination | 🟡 Medium | All list endpoints | No `limit`, `offset`, or cursor parameters |
| No versioning strategy | 🟡 Medium | `/api/v1/` | V1 prefix exists but there's no mechanism for version negotiation or deprecation |
| Missing `Content-Type` validation | 🟡 Medium | POST endpoints | Don't validate `Content-Type: application/json` before attempting JSON decode |

---

## DRY Violations

| Code Pattern | Locations | Description |
|-------------|-----------|-------------|
| ClickHouse host/port env var reading | `api-server/main.go`, `worker/main.go`, `mcp-server/main.go`, `init-db/main.go`, `seeder/main.go` | Same 6-line pattern repeated 5 times |
| OTel SDK initialization | `api-server/main.go`, `ingest-gateway/main.go`, `worker/main.go` | Same 8-line init-and-defer block repeated 3 times |
| Mock data for agent traces | `mock/repository.go` L43-71, `health_repository.go` L450-479 | Identical fallback trace data duplicated |
| Fallback graph generation | `server.go` L774-856 | Three separate `fallback*` functions with similar hardcoded graph structures |
| `MockHealthRepository` | `storage/mock/repository.go`, `cmd/mcp-server/main.go` L24-61 | Two separate mock implementations of the same interface |

---

## YAGNI Violations

| Feature | Location | Concern |
|---------|----------|---------|
| PagerDuty bridge | `alerting/pagerduty_bridge.go` | Referenced in codebase but never wired into any cmd entrypoint |
| Slack bridge | `alerting/slack_bridge.go` | Referenced in codebase but never wired into any cmd entrypoint |
| Streaming jobs | `internal/streaming/` (6 files) | Appear to be standalone stream processing jobs but none are imported or used by any cmd |
| `casting.yaml` / `casting.yaml.lock` | Root directory | Unknown purpose, not referenced anywhere |
| `pours/kustomization.yaml` | Root directory | K8s manifest directory with unclear relation to the Helm chart |
