CREATE DATABASE IF NOT EXISTS telemetry_health;

-- Table: telemetry_health.signal_metrics
-- Engine: AggregatingMergeTree
-- TTL: 30 days
-- Primary index: (tenant_id, timestamp)
CREATE TABLE IF NOT EXISTS telemetry_health.signal_metrics
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

-- Table: telemetry_health.root_cause_records
-- Engine: ReplacingMergeTree
-- TTL: 90 days
-- Primary index: (agent_id, trace_id)
CREATE TABLE IF NOT EXISTS telemetry_health.root_cause_records
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
