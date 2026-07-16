package clickhouse

import (
	"database/sql"
	"go.uber.org/zap"
)

// Schema manages the ClickHouse DDL as specified in PRD §9.1.
type Schema struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewSchema(db *sql.DB, logger *zap.Logger) *Schema {
	return &Schema{db: db, logger: logger}
}

// InitSchema creates the explicitly optimized tables with TTLs and AggregatingMergeTree engines.
// All tables use:
// - LowCardinality(String) for bounded-vocabulary columns (PRD §9.1)
// - AggregatingMergeTree or ReplacingMergeTree for sketch/aggregate tables
// - Explicit TTL: 30 days for raw signals, 12 months for health scores, 90 days for remediation
// - PARTITION BY toYYYYMM for efficient TTL-based partition drops
func (s *Schema) InitSchema() error {
	queries := []string{
		// Database
		`CREATE DATABASE IF NOT EXISTS telemetry_health`,

		// cardinality_signal: AggregatingMergeTree with 30-day TTL (PRD §9.1)
		`CREATE TABLE IF NOT EXISTS telemetry_health.cardinality_signal (
			tenant_id UUID,
			service LowCardinality(String),
			attribute_key LowCardinality(String),
			window_start DateTime64(3),
			unique_estimate UInt64,
			hll_sketch AggregateFunction(uniqCombined, String)
		) ENGINE = AggregatingMergeTree()
		PARTITION BY toYYYYMM(window_start)
		ORDER BY (tenant_id, service, attribute_key, window_start)
		TTL toDateTime(window_start) + INTERVAL 30 DAY`,

		// orphan_signal: raw event store with 30-day TTL (PRD §9.1)
		`CREATE TABLE IF NOT EXISTS telemetry_health.orphan_signal (
			tenant_id UUID,
			trace_id String,
			span_id String,
			parent_span_id String,
			collector_id LowCardinality(String),
			detected_at DateTime64(3)
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(detected_at)
		ORDER BY (tenant_id, detected_at)
		TTL toDateTime(detected_at) + INTERVAL 30 DAY`,

		// coverage_signal: ReplacingMergeTree for upserts, no TTL (baseline-driven)
		`CREATE TABLE IF NOT EXISTS telemetry_health.coverage_signal (
			tenant_id UUID,
			service LowCardinality(String),
			environment LowCardinality(String),
			last_seen_at DateTime64(3),
			baseline_expected UInt8,
			grace_period_seconds UInt32
		) ENGINE = ReplacingMergeTree(last_seen_at)
		ORDER BY (tenant_id, service, environment)`,

		// health_score: SummingMergeTree for roll-ups, 12-month retention (PRD §9.1, §10)
		`CREATE TABLE IF NOT EXISTS telemetry_health.health_score (
			tenant_id UUID,
			scope LowCardinality(String),
			service LowCardinality(String),
			environment LowCardinality(String),
			score Float64,
			cardinality_violation Float64,
			orphan_rate Float64,
			coverage_drop Float64,
			ts DateTime64(3)
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(ts)
		ORDER BY (tenant_id, scope, service, environment, ts)
		TTL toDateTime(ts) + INTERVAL 12 MONTH`,

		// remediation_event: SOC 2 audit trail, 90-day retention (PRD §9.1, §10 Security)
		`CREATE TABLE IF NOT EXISTS telemetry_health.remediation_event (
			tenant_id UUID,
			issue_type LowCardinality(String),
			generated_yaml String,
			validated UInt8,
			applied UInt8,
			actor_id String,
			actor_role LowCardinality(String),
			source_ip String,
			action LowCardinality(String),
			resource_id String,
			ts DateTime64(3)
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(ts)
		ORDER BY (tenant_id, ts)
		TTL toDateTime(ts) + INTERVAL 90 DAY`,

		// alert_event: deduplicated alert history with delivery tracking, 30-day TTL
		`CREATE TABLE IF NOT EXISTS telemetry_health.alert_event (
			tenant_id UUID,
			alert_id String,
			score_at_fire Float64,
			contributing_signals String,
			affected_service LowCardinality(String),
			remediation_snippet String,
			dashboard_link String,
			suppressed UInt8,
			delivered_to Array(String),
			ts DateTime64(3)
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(ts)
		ORDER BY (tenant_id, alert_id, ts)
		TTL toDateTime(ts) + INTERVAL 30 DAY`,

		// tenant_config: per-tenant configurable health score weights (PRD §8.4)
		`CREATE TABLE IF NOT EXISTS telemetry_health.tenant_config (
			tenant_id UUID,
			cardinality_weight Float64 DEFAULT 0.20,
			orphan_weight Float64 DEFAULT 0.30,
			coverage_weight Float64 DEFAULT 0.50,
			coverage_grace_period_seconds UInt32 DEFAULT 600,
			updated_at DateTime64(3)
		) ENGINE = ReplacingMergeTree(updated_at)
		ORDER BY (tenant_id)`,

		// Phase 3: Lightweight local trace index for Graph Engine
		`CREATE TABLE IF NOT EXISTS telemetry_health.telemetryhealth_trace_index_spans (
			trace_id String CODEC(ZSTD(1)),
			span_id String CODEC(ZSTD(1)),
			parent_span_id String CODEC(ZSTD(1)),
			service_name LowCardinality(String) CODEC(ZSTD(1)),
			operation_name LowCardinality(String) CODEC(ZSTD(1)),
			start_time DateTime64(9) CODEC(DoubleDelta, LZ4),
			end_time DateTime64(9) CODEC(DoubleDelta, LZ4),
			status LowCardinality(String),
			attributes String,
			tenant_id LowCardinality(String) CODEC(ZSTD(1))
		) ENGINE = MergeTree
		PARTITION BY toDate(start_time)
		ORDER BY (tenant_id, start_time, service_name, trace_id)`,
	}

	for _, q := range queries {
		preview := q
		if len(preview) > 60 {
			preview = preview[:60] + "..."
		}
		s.logger.Info("Executing DDL", zap.String("query_preview", preview))
		if _, err := s.db.Exec(q); err != nil {
			s.logger.Error("DDL execution failed", zap.String("query_preview", preview), zap.Error(err))
			return err
		}
	}
	s.logger.Info("ClickHouse schema initialized successfully", zap.Int("table_count", len(queries)-1))
	return nil
}
