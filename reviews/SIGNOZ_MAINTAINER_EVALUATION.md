# SigNoz Maintainer Evaluation: TelemetryHealth Architecture

**Evaluator:** SigNoz Core Maintainer
**Scope:** Evaluation of integration points against public SigNoz extension paths, official APIs, and documented database interfaces.

---

## 1. Direct ClickHouse SQL Querying (`signoz_traces.signoz_index_v2`)

### Classification
*   **Uses internal SigNoz implementation details**

### Source Code Evidence
*   **File:** [health_repository.go L149](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/storage/clickhouse/health_repository.go#L149) and [L309](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/storage/clickhouse/health_repository.go#L309):
    ```sql
    FROM signoz_traces.signoz_index_v2
    ```

### Runtime Call Chain
```
REST Server Router 
  ↳ GetAgentTraces (REST Handler)
    ↳ healthRepo.QueryAgentTraces(ctx)
      ↳ clickhouse-go/v2 Driver
        ↳ SQL Query: "SELECT ... FROM signoz_traces.signoz_index_v2"
```

### Supported by SigNoz Documentation
*   **No.** SigNoz does not expose or support direct SQL querying against the `signoz_traces.signoz_index_v2` schema as a public API contract. The schema is internal and subject to change without deprecation warnings. The official documented path for querying trace metrics is the **Query Service REST API** (e.g. `/api/v3/query_range`).

### Upgrade Risks
*   **Extreme.** SigNoz regularly migrates trace indices to improve performance (e.g. transitioning from `signoz_index` to `signoz_index_v2`, changing column structures, or moving to multi-tenant database designs). Minor-version upgrades of SigNoz will break this query and cause execution failures.

### Maintenance Risks
*   **High.** Querying the backing database directly bypasses SigNoz's query optimization engine, access control layers (RBAC), and query rate limits, potentially leading to performance degradation on the database cluster.

---

## 2. SigNoz Alertmanager Integration (`SigNozBridge`)

### Classification
*   **Mock implementation**
    *(Note: The codebase claims to "fire alerts to SigNoz Alertmanager" in comments/documentation, but the actual code contains no network connection or payload serialization).*

### Source Code Evidence
*   **File:** [signoz_bridge.go L71-93](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/alerting/signoz_bridge.go#L71-L93):
    ```go
    func (b *SigNozBridge) FireAlert(ctx context.Context, payload AlertPayload) error {
        // ... (cooldown check only) ...
        b.lastFired[payload.AlertID] = now
        b.logger.Info("Firing alert to SigNoz Alertmanager", ...)
        return nil
    }
    ```

### Runtime Call Chain
```
Go Runtime Invoker
  ↳ SigNozBridge.FireAlert(ctx, payload)
    ↳ zap.Logger.Info(...)
      ↳ Return nil
```

### Supported by SigNoz Documentation
*   **No.** While SigNoz officially supports webhook targets and Prometheus Alertmanager integrations, this Go structure does not make network contact with any documented SigNoz alerting endpoint.

### Upgrade Risks
*   **Low.** There is no network serialization logic present to break during updates.

### Maintenance Risks
*   **Critical.** Alert conditions will silently fail to register on any live monitoring panel, leading to completely silent failures in production.

---

## 3. Dashboard Configurations (`agent_health.json`)

### Classification
*   **Prototype implementation**

### Source Code Evidence
*   **File:** [agent_health.json](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/signoz_implementations/agent_health.json)

### Runtime Call Chain
```
JSON File Import (SigNoz UI / Schema Import Engine)
  ↳ SigNoz Dashboard Store
    ↳ Dashboard renders Prometheus queries against the data source
```

### Supported by SigNoz Documentation
*   **Yes.** Importing dashboards via JSON templates is fully supported in SigNoz.

### Upgrade Risks
*   **Medium.** The dashboard queries metrics (like `telemetryhealth_agent_health_score`) that are not generated or exported by the Go codebase (the codebase implements `telemetryhealth_pipeline_health_score`). If imported, the dashboard will load successfully but the panels will fail to display metric graphs.

### Maintenance Risks
*   **Medium.** Keeping dashboard files in sync with code-level metric name changes must be handled manually since there is no validator or build sync step.

---

## 4. OpenTelemetry Ingestion & Pipeline Configuration (`casting.yaml`)

### Classification
*   **Official OpenTelemetry extension / Vendor-neutral implementation**

### Source Code Evidence
*   **File:** [grpc_server.go](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/ingest/grpc_server.go) and [casting.yaml](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/casting.yaml).

### Runtime Call Chain
```
OTel Tracer Clients
  ↳ gRPC / OTLP Connection (Ingest Gateway)
    ↳ grpc_server.go (Unmarshals spans using go.opentelemetry.io/collector/pdata)
      ↳ Storage / Kafka Write
```

### Supported by SigNoz Documentation
*   **Yes.** SigNoz is natively built on OpenTelemetry standards and fully supports receiving standard OTLP data formatted this way.

### Upgrade Risks
*   **Low.** The implementation uses stable, standardized OpenTelemetry `pdata` APIs which are backward-compatible.

### Maintenance Risks
*   **Low.** Standardized OTel APIs require very low maintenance overhead.

---

## 5. SigNoz MCP Client (`client.go`)

### Classification
*   **Mock implementation**

### Source Code Evidence
*   **File:** [client.go L25-42](file:///c:/Users/sunanda.AMFIIND/Desktop/SHUBHAM%20PROJECT/TelemetryHealth_/control-plane/internal/mcp/client.go#L25-L42):
    ```go
    func QueryAgentTraces(ctx context.Context, tenantID string, serverURL string) (*Traces, error) {
        query := `SELECT * FROM traces WHERE attributes['service.name'] = 'ai-agent'`
        _ = query // keep compiler happy
        return &Traces{
            Count: 2,
            Data: []map[string]interface{}{ ... }, // simulated slice
        }, nil
    }
    ```

### Runtime Call Chain
```
Go Caller (currently unreachable)
  ↳ mcp.QueryAgentTraces(...)
    ↳ Discards query
      ↳ Returns simulated map array
```

### Supported by SigNoz Documentation
*   **No.** SigNoz does not provide or support Model Context Protocol (MCP) servers or libraries. The client uses an imaginary/unsupported library dependency referenced in comments (`github.com/signoz/mcp-go`).

### Upgrade Risks
*   **High.** Any attempt to implement real library imports will require replacing the entire codebase within this file.

### Maintenance Risks
*   **High.** The code returns simulated mock trace IDs, hiding query issues from downstream orchestrators during development.
