# 04 — Performance Audit

## Database Query Performance

### PERF-01: `FINAL` Modifier on ReplacingMergeTree
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/storage/clickhouse/health_repository.go` L490
- **Explanation**: `SELECT ... FROM telemetry_health.tenant_config FINAL` forces ClickHouse to merge all parts on read. For ReplacingMergeTree with low cardinality (one row per tenant), this is acceptable. But as the table grows, FINAL becomes expensive.
- **How to fix**: For small config tables this is fine. Document that this is intentional. For larger tables, consider `OPTIMIZE TABLE ... FINAL` on a schedule instead.

### PERF-02: `attributes LIKE '%gen_ai%'` Full Text Scan
- **Severity**: 🟠 High
- **File**: `control-plane/internal/storage/clickhouse/health_repository.go` L335
- **Explanation**: The agent trace query uses `LIKE '%gen_ai%'` on a `String` column with no index. This forces a full table scan on every request.
- **How to fix**: Add a materialized column or bloom filter index: `INDEX idx_attrs attributes TYPE tokenbf_v1(10240, 3, 0) GRANULARITY 4`.

### PERF-03: No Query Timeouts on ClickHouse Queries
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/storage/clickhouse/health_repository.go`
- **Explanation**: While the client sets `max_execution_time: 60`, individual queries don't pass context deadlines. A slow query will block the HTTP request for up to 60 seconds.
- **How to fix**: Pass `context.WithTimeout` from the HTTP handler through to all DB queries.

## Kafka Performance

### PERF-04: Per-Span Kafka Publish in Hot Path
- **Severity**: 🟠 High
- **File**: `control-plane/internal/ingest/grpc_server.go` L66-97
- **Explanation**: For every span in an OTLP export, the ingest gateway publishes 3 separate Kafka messages (orphan, cardinality per attribute, rawspan). For a batch of 1000 spans with 10 attributes each, this is 12,000 Kafka writes per export.
- **Why it matters**: Kafka write amplification will bottleneck the ingest pipeline under real load.
- **How to fix**: Batch Kafka publishes per OTLP export request. Collect all events into slices, then publish once per topic using `WriteMessages(ctx, msgs...)`.

### PERF-05: Kafka Consumer 100ms Fetch Timeout Creates Busy Loop
- **Severity**: 🟡 Medium
- **File**: `control-plane/internal/kafka/consumer.go` L67
- **Explanation**: The consumer polls every 100ms with `context.WithTimeout(ctx, 100*time.Millisecond)`. When there are no messages, this creates 10 context allocations per second per consumer (40/sec total for 4 workers).
- **How to fix**: Increase to 500ms-1s, or use a blocking read with the batch timer.

## API Performance

### PERF-06: Unbounded Prometheus Label Cardinality
- **Severity**: 🟠 High
- **File**: `control-plane/internal/api/rest/server.go` L172-173
- **Explanation**: `r.URL.Path` is used as a Prometheus label. With path parameters like `/api/v1/tenant/{tenant_id}/health`, each unique tenant_id creates a new time series. This will cause a cardinality explosion in Prometheus.
- **How to fix**: Use the chi route pattern (`chi.RouteContext(r.Context()).RoutePattern()`) instead of the actual path.
