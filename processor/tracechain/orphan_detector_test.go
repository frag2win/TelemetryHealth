package tracechain_test

import (
	"sync"
	"testing"
	"time"

	"github.com/frag2win/TelemetryHealth/processor/tracechain"
	"go.uber.org/zap"
)

func TestOrphanDetector_ConcurrentObserveAndCheck_NoRace(t *testing.T) {
	detector := tracechain.NewOrphanDetector(zap.NewNop())
	now := time.Now()

	var wg sync.WaitGroup
	const numRoutines = 50
	wg.Add(numRoutines * 2)

	// Concurrently call ObserveSpan
	for i := 0; i < numRoutines; i++ {
		go func(id int) {
			defer wg.Done()
			detector.ObserveSpan("span-123", now)
		}(i)
	}

	// Concurrently call CheckOrphans
	for i := 0; i < numRoutines; i++ {
		go func(id int) {
			defer wg.Done()
			tuples := []tracechain.SpanTuple{
				{TraceID: "trace-1", SpanID: "span-abc", ParentSpanID: "span-123", Timestamp: now.Add(-time.Hour)},
			}
			detector.CheckOrphans(tuples, now, time.Minute)
		}(i)
	}

	wg.Wait()
}
