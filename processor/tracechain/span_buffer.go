package tracechain

import (
	"sync"
	"time"
)

// SpanTuple represents the structural identity of a span needed for orphan correlation.
type SpanTuple struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Timestamp    time.Time
}

// Buffer holds span tuples until they exceed a lateness window, 
// at which point they are flushed out for evaluation.
type Buffer struct {
	mu        sync.Mutex
	tuples    []SpanTuple
	retention time.Duration
}

// NewBuffer creates a new span tuple buffer.
func NewBuffer(retention time.Duration) *Buffer {
	return &Buffer{
		tuples:    make([]SpanTuple, 0),
		retention: retention,
	}
}

// Add inserts a new span tuple into the buffer.
func (b *Buffer) Add(tuple SpanTuple) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tuples = append(b.tuples, tuple)
}

// Flush returns all tuples older than the retention window.
func (b *Buffer) Flush(now time.Time) []SpanTuple {
	b.mu.Lock()
	defer b.mu.Unlock()

	var active []SpanTuple
	var expired []SpanTuple

	for _, t := range b.tuples {
		if now.Sub(t.Timestamp) > b.retention {
			expired = append(expired, t)
		} else {
			active = append(active, t)
		}
	}

	b.tuples = active
	return expired
}
