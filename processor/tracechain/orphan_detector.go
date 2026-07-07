package tracechain

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// OrphanDetector provides a fast local signal for orphan traces.
// It matches expired span tuples against a rolling set of seen span IDs.
type OrphanDetector struct {
	mu        sync.Mutex
	seenSpans map[string]time.Time
	logger    *zap.Logger
}

// NewOrphanDetector creates a new local orphan detector.
func NewOrphanDetector(logger *zap.Logger) *OrphanDetector {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &OrphanDetector{
		seenSpans: make(map[string]time.Time),
		logger:    logger,
	}
}

// ObserveSpan records that a span ID was seen locally.
func (d *OrphanDetector) ObserveSpan(spanID string, now time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seenSpans[spanID] = now
}

// CheckOrphans evaluates expired tuples to see if their parents were observed locally.
// It also cleans up the seenSpans map to bound memory.
func (d *OrphanDetector) CheckOrphans(expiredTuples []SpanTuple, now time.Time, retention time.Duration) []SpanTuple {
	var orphans []SpanTuple

	d.mu.Lock()
	for _, t := range expiredTuples {
		// Only check spans that have a parent
		if t.ParentSpanID != "" {
			if _, ok := d.seenSpans[t.ParentSpanID]; !ok {
				orphans = append(orphans, t)
			}
		}
	}
	d.mu.Unlock()

	d.mu.Lock()
	var toDelete []string
	for id, timestamp := range d.seenSpans {
		if now.Sub(timestamp) > retention*2 {
			toDelete = append(toDelete, id)
		}
	}
	d.mu.Unlock()

	if len(toDelete) > 0 {
		d.mu.Lock()
		for _, id := range toDelete {
			delete(d.seenSpans, id)
		}
		d.mu.Unlock()
	}

	if len(orphans) > 0 {
		d.logger.Debug("Local orphan detection found orphans", zap.Int("count", len(orphans)))
	}

	return orphans
}
