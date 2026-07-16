# Storage Architecture

This document specifies the exact ClickHouse table architectures, indexing layouts, and TTL policies.

## Table: signal_metrics

Used to track rolled-up operational metrics across telemetry data over time.

```sql
CREATE TABLE telemetry_health.signal_metrics
(
    tenant_id UUID,
    timestamp DateTime,
    metric_name LowCardinality(String),
    value Float64,
    labels Map(String, String)
)
ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (tenant_id, timestamp, metric_name)
TTL timestamp + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192;
```

**Layout Details:**
- **Engine:** `AggregatingMergeTree` for optimized rollup queries.
- **TTL:** 30 days retention policy to automatically clear ephemeral signaling data.
- **Indexing (ORDER BY):** `tenant_id` and `timestamp` allow optimal time-series filtering on a per-tenant basis.

## Table: root_cause_records

Used to persist the analytical engine's verdicts regarding tracing breakdowns, sampling gaps, and cardinality explosions.

```sql
CREATE TABLE telemetry_health.root_cause_records
(
    tenant_id UUID,
    agent_id UUID,
    trace_id UUID,
    failure_type LowCardinality(String),
    severity LowCardinality(String),
    evidence_span_ids Array(String),
    description String,
    created_at DateTime DEFAULT now()
)
ENGINE = ReplacingMergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (agent_id, trace_id, created_at)
TTL created_at + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192;
```

**Layout Details:**
- **Engine:** `ReplacingMergeTree` ensures deduplication of identical analysis payloads per trace.
- **TTL:** 90 days retention policy for long-term audit and post-mortem review.
- **Indexing (ORDER BY):** Ordered primarily by `agent_id` and `trace_id` for extremely fast lookup when exploring a specific trace's root cause within an agent's historical context.
