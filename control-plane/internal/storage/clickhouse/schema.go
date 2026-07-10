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
func (s *Schema) InitSchema() error {
	queries := []string{
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
		TTL window_start + INTERVAL 30 DAY;`,
		
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
		TTL detected_at + INTERVAL 30 DAY;`,

		`CREATE TABLE IF NOT EXISTS telemetry_health.coverage_signal (
			tenant_id UUID,
			service LowCardinality(String),
			last_seen_at DateTime64(3),
			baseline_expected UInt8
		) ENGINE = ReplacingMergeTree(last_seen_at)
		ORDER BY (tenant_id, service);`,
	}

	for _, q := range queries {
		s.logger.Info("Executing DDL", zap.String("query", q[:30]+"..."))
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
