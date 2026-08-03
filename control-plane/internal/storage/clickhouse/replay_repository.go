package clickhouse

import (
	"context"
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
		SELECT traceID, spanID, parentSpanID, serviceName, name, timestamp, durationNano, statusCode, stringTagMap
		FROM signoz_traces.signoz_index_v2
		WHERE traceID = $1
		ORDER BY timestamp ASC
	`, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query replay events: %w", err)
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

func (r *ClickhouseReplayRepository) GetRecentReplays(ctx context.Context, tenantID string, limit int) ([]engine.ReplayEvent, error) {
	// Fetch recent distinct trace IDs first
	offset := 0

	traceRows, err := r.db.Query(ctx, `
		SELECT traceID
		FROM signoz_traces.signoz_index_v2
		GROUP BY traceID
		ORDER BY max(timestamp) DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
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
		SELECT traceID, spanID, parentSpanID, serviceName, name, timestamp, durationNano, statusCode, stringTagMap
		FROM signoz_traces.signoz_index_v2
		WHERE traceID IN ($1)
		ORDER BY timestamp ASC
	`, traceIDs)
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
			traceID, spanID, parentSpanID, serviceName, operationName string
			startTime                                                 time.Time
			durationNano                                              uint64
			statusCode                                                int16
			stringTagMap                                              map[string]string
		)

		if err := rows.Scan(&traceID, &spanID, &parentSpanID, &serviceName, &operationName, &startTime, &durationNano, &statusCode, &stringTagMap); err != nil {
			r.logger.Error("failed to scan replay event row", zap.Error(err))
			continue
		}

		endTime := startTime.Add(time.Duration(durationNano))

		statusStr := "UNSET"
		if statusCode == 1 {
			statusStr = "OK"
		} else if statusCode == 2 {
			statusStr = "ERROR"
		}

		var attrs map[string]interface{}
		if len(stringTagMap) > 0 {
			attrs = make(map[string]interface{})
			for k, v := range stringTagMap {
				attrs[k] = v
			}
		}

		events = append(events, engine.ReplayEvent{
			TraceID:       traceID,
			SpanID:        spanID,
			ParentSpanID:  parentSpanID,
			ServiceName:   serviceName,
			OperationName: operationName,
			StartTime:     startTime,
			EndTime:       endTime,
			Status:        statusStr,
			Attributes:    attrs,
		})
	}
	return events, nil
}
