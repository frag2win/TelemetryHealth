package engine

import (
	"context"
	"time"
)

// ReplayEvent represents a single span of execution behavior.
type ReplayEvent struct {
	TraceID       string
	SpanID        string
	ParentSpanID  string
	ServiceName   string
	OperationName string
	StartTime     time.Time
	EndTime       time.Time
	Status        string
	Attributes    map[string]interface{}
	TenantID      string
}

// ReplayRepository abstracts the retrieval of trace events for behavior graph generation.
type ReplayRepository interface {
	// GetReplay fetches all events for a specific trace, useful for Root Cause graphs.
	GetReplay(ctx context.Context, tenantID, traceID string) ([]ReplayEvent, error)
	// GetRecentReplays fetches events from the most recent traces, useful for Topology graphs.
	GetRecentReplays(ctx context.Context, tenantID string, limit int) ([]ReplayEvent, error)
}
