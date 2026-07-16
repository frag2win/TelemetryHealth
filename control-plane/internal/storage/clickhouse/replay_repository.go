package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/engine"
	"go.uber.org/zap"
)

type ClickhouseReplayRepository struct {
	db     driver.Conn
	logger *zap.Logger
}

func NewReplayRepository(db driver.Conn, logger *zap.Logger) *ClickhouseReplayRepository {
	return &ClickhouseReplayRepository{db: db, logger: logger}
}

func (r *ClickhouseReplayRepository) GetReplay(ctx context.Context, tenantID, traceID string) ([]engine.ReplayEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT trace_id, span_id, parent_span_id, service_name, operation_name, start_time, end_time, status, attributes
		FROM telemetry_health.telemetryhealth_trace_index_spans
		WHERE tenant_id = $1 AND trace_id = $2
		ORDER BY start_time ASC
	`, tenantID, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query replay events: %w", err)
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

func (r *ClickhouseReplayRepository) GetRecentReplays(ctx context.Context, tenantID string, limit int) ([]engine.ReplayEvent, error) {
	// Fetch recent distinct trace IDs first
	traceRows, err := r.db.Query(ctx, `
		SELECT DISTINCT trace_id
		FROM telemetry_health.telemetryhealth_trace_index_spans
		WHERE tenant_id = $1
		ORDER BY start_time DESC
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent trace IDs: %w", err)
	}
	defer traceRows.Close()

	var traceIDs []string
	for traceRows.Next() {
		var id string
		if err := traceRows.Scan(&id); err == nil {
			traceIDs = append(traceIDs, id)
		}
	}

	if len(traceIDs) == 0 {
		return []engine.ReplayEvent{}, nil
	}

	// Fetch all events for those traces
	rows, err := r.db.Query(ctx, `
		SELECT trace_id, span_id, parent_span_id, service_name, operation_name, start_time, end_time, status, attributes
		FROM telemetry_health.telemetryhealth_trace_index_spans
		WHERE tenant_id = $1 AND trace_id IN ($2)
		ORDER BY start_time ASC
	`, tenantID, traceIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent events: %w", err)
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

func (r *ClickhouseReplayRepository) scanEvents(rows driver.Rows) ([]engine.ReplayEvent, error) {
	var events []engine.ReplayEvent
	for rows.Next() {
		var (
			traceID, spanID, parentSpanID, serviceName, operationName, status, attributesStr string
			startTime, endTime                                                               time.Time
		)

		if err := rows.Scan(&traceID, &spanID, &parentSpanID, &serviceName, &operationName, &startTime, &endTime, &status, &attributesStr); err != nil {
			r.logger.Error("failed to scan replay event row", zap.Error(err))
			continue
		}

		var attrs map[string]interface{}
		if attributesStr != "" {
			_ = json.Unmarshal([]byte(attributesStr), &attrs)
		}

		events = append(events, engine.ReplayEvent{
			TraceID:       traceID,
			SpanID:        spanID,
			ParentSpanID:  parentSpanID,
			ServiceName:   serviceName,
			OperationName: operationName,
			StartTime:     startTime,
			EndTime:       endTime,
			Status:        status,
			Attributes:    attrs,
		})
	}
	return events, nil
}
