package streaming

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// spanTuple is the minimal structural identity of a span needed for orphan correlation.
type spanTuple struct {
	TenantID     string
	TraceID      string
	SpanID       string
	ParentSpanID string
	CollectorID  string
	EventTime    time.Time
}

// orphanWindow holds tuples within the bounded-lateness window keyed by traceID.
type orphanWindow struct {
	mu        sync.Mutex
	byTrace   map[string][]spanTuple // traceID → all tuples seen
	watermark time.Time              // current event-time watermark
}

// TraceChainJob performs cross-collector orphan correlation with bounded-lateness windowing.
// It implements PRD §8.2: distributed correlation across all collector instances.
type TraceChainJob struct {
	logger           *zap.Logger
	mu               sync.Mutex
	window           orphanWindow
	latenessWindow   time.Duration // default 30s per PRD §8.2
	clockSkewTolerance time.Duration // 5s NTP tolerance per PRD §8.2
}

func NewTraceChainJob(logger *zap.Logger) *TraceChainJob {
	return &TraceChainJob{
		logger:           logger,
		latenessWindow:   30 * time.Second,
		clockSkewTolerance: 5 * time.Second,
		window: orphanWindow{
			byTrace: make(map[string][]spanTuple),
		},
	}
}

// Process ingests a structural span tuple for cross-collector orphan correlation.
// It implements bounded out-of-orderness allowance (30s) based on event-time watermarks (PRD §8.2).
func (j *TraceChainJob) Process(ctx context.Context, tenantID string, traceID, spanID, parentSpanID string, eventTime time.Time) error {
	j.window.mu.Lock()
	defer j.window.mu.Unlock()

	// Advance watermark if this event is newer than current (+ skew tolerance)
	effectiveTime := eventTime.Add(j.clockSkewTolerance)
	if effectiveTime.After(j.window.watermark) {
		j.window.watermark = effectiveTime
	}

	// Drop spans that are too far behind the watermark (late data beyond lateness window)
	cutoff := j.window.watermark.Add(-j.latenessWindow)
	if eventTime.Before(cutoff) {
		j.logger.Warn("Dropping late span tuple, beyond lateness window",
			zap.String("trace_id", traceID),
			zap.Time("event_time", eventTime),
			zap.Time("cutoff", cutoff),
		)
		return nil
	}

	// Store the tuple for correlation
	t := spanTuple{
		TenantID:     tenantID,
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
		EventTime:    eventTime,
	}
	j.window.byTrace[traceID] = append(j.window.byTrace[traceID], t)

	j.logger.Debug("Buffered trace tuple for cross-collector correlation",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("parent_span_id", parentSpanID),
	)
	return nil
}

// FlushOrphans evaluates all trace windows whose entire duration is older than
// the watermark minus the lateness window, and returns orphaned spans (spans
// whose parentSpanID was never observed in this window).
func (j *TraceChainJob) FlushOrphans(ctx context.Context) []spanTuple {
	j.window.mu.Lock()
	defer j.window.mu.Unlock()

	cutoff := j.window.watermark.Add(-j.latenessWindow)
	var orphans []spanTuple
	var toDelete []string

	for traceID, tuples := range j.window.byTrace {
		// Only evaluate windows fully before the watermark cutoff
		allExpired := true
		for _, t := range tuples {
			if t.EventTime.After(cutoff) {
				allExpired = false
				break
			}
		}
		if !allExpired {
			continue
		}

		// Build set of all known span IDs in this trace window
		knownSpanIDs := make(map[string]struct{}, len(tuples))
		for _, t := range tuples {
			knownSpanIDs[t.SpanID] = struct{}{}
		}

		// Flag spans whose parent was never seen (cross-collector orphan)
		for _, t := range tuples {
			if t.ParentSpanID == "" {
				// Root span — not an orphan
				continue
			}
			if _, seen := knownSpanIDs[t.ParentSpanID]; !seen {
				j.logger.Info("Cross-collector orphan detected",
					zap.String("trace_id", traceID),
					zap.String("span_id", t.SpanID),
					zap.String("missing_parent_id", t.ParentSpanID),
				)
				orphans = append(orphans, t)
			}
		}
		toDelete = append(toDelete, traceID)
	}

	for _, traceID := range toDelete {
		delete(j.window.byTrace, traceID)
	}

	return orphans
}
