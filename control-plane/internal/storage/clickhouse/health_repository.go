package clickhouse

import (
	"context"
	"fmt"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.uber.org/zap"
)

// HealthMetrics is the structured result returned from the health query.
type HealthMetrics struct {
	TenantID         string
	CardinalityMax   uint64
	OrphanCount      uint64
	ActiveServices   uint64
	CompositeScore   float64
	RemediationIssue string
	Window           time.Time
}

// HealthRepository handles all read queries for the health dashboard.
type HealthRepository struct {
	conn   driver.Conn
	logger *zap.Logger
}

func NewHealthRepository(conn driver.Conn, logger *zap.Logger) *HealthRepository {
	return &HealthRepository{conn: conn, logger: logger}
}

// QueryHealthMetrics fetches aggregated telemetry signals for a tenant.
func (r *HealthRepository) QueryHealthMetrics(ctx context.Context, tenantID string) (*HealthMetrics, error) {
	metrics := &HealthMetrics{TenantID: tenantID}

	// --- 1. Cardinality: peak unique estimate in last 30 min ---
	cardQuery := `
		SELECT max(unique_estimate) AS max_cardinality
		FROM telemetry_health.cardinality_signal
		WHERE tenant_id = {tenant_id:UUID}
		  AND window_start >= now() - INTERVAL 30 MINUTE`

	row := r.conn.QueryRow(ctx, cardQuery, ch.Named("tenant_id", tenantID))
	if err := row.Scan(&metrics.CardinalityMax); err != nil {
		r.logger.Warn("cardinality query failed, defaulting to 0", zap.Error(err))
		metrics.CardinalityMax = 0
	}

	// --- 2. Orphaned Traces: count in last 30 min ---
	orphanQuery := `
		SELECT count() AS orphan_count
		FROM telemetry_health.orphan_signal
		WHERE tenant_id = {tenant_id:UUID}
		  AND detected_at >= now() - INTERVAL 30 MINUTE`

	row2 := r.conn.QueryRow(ctx, orphanQuery, ch.Named("tenant_id", tenantID))
	if err := row2.Scan(&metrics.OrphanCount); err != nil {
		r.logger.Warn("orphan query failed, defaulting to 0", zap.Error(err))
		metrics.OrphanCount = 0
	}

	// --- 3. Active Services: distinct services seen in last 10 min ---
	coverageQuery := `
		SELECT count() AS active_services
		FROM telemetry_health.coverage_signal
		WHERE tenant_id = {tenant_id:UUID}
		  AND last_seen_at >= now() - INTERVAL 10 MINUTE`

	row3 := r.conn.QueryRow(ctx, coverageQuery, ch.Named("tenant_id", tenantID))
	if err := row3.Scan(&metrics.ActiveServices); err != nil {
		r.logger.Warn("coverage query failed, defaulting to 0", zap.Error(err))
		metrics.ActiveServices = 0
	}

	// --- 4. Composite Health Score ---
	// Weights: cardinality 20%, orphan 30%, coverage 50%
	// Normalise: >1M cardinality = 100% violation, >1000 orphans = 100% violation
	cardViolation := clamp(float64(metrics.CardinalityMax)/1_000_000.0) * 100
	orphanViolation := clamp(float64(metrics.OrphanCount)/1000.0) * 100
	// coverage: if active services < baseline (assume 10), flag as drop
	coverageDrop := 0.0
	if metrics.ActiveServices < 10 {
		coverageDrop = (1.0 - float64(metrics.ActiveServices)/10.0) * 100
	}

	metrics.CompositeScore = 100 - (0.20*cardViolation + 0.30*orphanViolation + 0.50*coverageDrop)
	if metrics.CompositeScore < 0 {
		metrics.CompositeScore = 0
	}

	if metrics.CardinalityMax > 1_000_000 {
		metrics.RemediationIssue = fmt.Sprintf("High cardinality detected: %d unique values", metrics.CardinalityMax)
	} else if metrics.OrphanCount > 100 {
		metrics.RemediationIssue = fmt.Sprintf("Elevated orphan spans: %d in last 30m", metrics.OrphanCount)
	}

	return metrics, nil
}

// clickhouse.Named is used for named parameters — this helper is in the driver package directly.
// The import alias below ensures we can use it.
func clamp(v float64) float64 {
	if v > 1.0 {
		return 1.0
	}
	return v
}

// Ensure driver package is used.
var _ driver.Conn
