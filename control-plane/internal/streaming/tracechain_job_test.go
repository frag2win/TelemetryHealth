package streaming

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestTraceJob() *TraceChainJob {
	return NewTraceChainJob(zap.NewNop())
}

// ── Process() tests ───────────────────────────────────────────────────────────

func TestTraceChainJob_Process_BuffersSpan(t *testing.T) {
	job := newTestTraceJob()
	ctx := context.Background()
	now := time.Now()

	err := job.Process(ctx, "tenant-1", "trace-a", "span-1", "parent-0", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job.window.mu.Lock()
	defer job.window.mu.Unlock()
	if len(job.window.byTrace["trace-a"]) != 1 {
		t.Errorf("expected 1 buffered tuple, got %d", len(job.window.byTrace["trace-a"]))
	}
}

func TestTraceChainJob_Process_DropsLateData(t *testing.T) {
	job := newTestTraceJob()
	ctx := context.Background()

	// Advance watermark with a recent event.
	recentTime := time.Now()
	_ = job.Process(ctx, "tenant-1", "trace-new", "span-new", "", recentTime)

	// Submit a span 60s in the past (beyond the 30s lateness window).
	lateTime := recentTime.Add(-60 * time.Second)
	err := job.Process(ctx, "tenant-1", "trace-old", "span-old", "p", lateTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job.window.mu.Lock()
	defer job.window.mu.Unlock()
	// The late trace should have been dropped, not buffered.
	if _, exists := job.window.byTrace["trace-old"]; exists {
		t.Error("expected late span to be dropped, but it was buffered")
	}
}

func TestTraceChainJob_Process_AcceptsWithinLatenessWindow(t *testing.T) {
	job := newTestTraceJob()
	ctx := context.Background()

	now := time.Now()
	// Submit a recent span to advance the watermark.
	_ = job.Process(ctx, "tenant-1", "trace-ref", "span-ref", "", now)

	// Submit a span 10s in the past — within the 30s window.
	slightlyLate := now.Add(-10 * time.Second)
	err := job.Process(ctx, "tenant-1", "trace-a", "span-a", "parent-x", slightlyLate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job.window.mu.Lock()
	defer job.window.mu.Unlock()
	if _, exists := job.window.byTrace["trace-a"]; !exists {
		t.Error("expected span within lateness window to be buffered")
	}
}

func TestTraceChainJob_Process_AdvancesWatermark(t *testing.T) {
	job := newTestTraceJob()
	ctx := context.Background()

	t1 := time.Now()
	t2 := t1.Add(5 * time.Second)

	_ = job.Process(ctx, "t", "trace-1", "s1", "", t1)
	_ = job.Process(ctx, "t", "trace-2", "s2", "", t2)

	job.window.mu.Lock()
	defer job.window.mu.Unlock()
	// Watermark should reflect t2 + clockSkewTolerance (5s).
	expected := t2.Add(5 * time.Second)
	if !job.window.watermark.Equal(expected) {
		t.Errorf("expected watermark %v, got %v", expected, job.window.watermark)
	}
}

// ── FlushOrphans() tests ──────────────────────────────────────────────────────

func TestTraceChainJob_FlushOrphans_DetectsOrphan(t *testing.T) {
	job := newTestTraceJob()
	ctx := context.Background()

	// Use a past event time so the window expires immediately.
	past := time.Now().Add(-120 * time.Second)

	// A span that references a parent that is never submitted.
	_ = job.Process(ctx, "tenant-1", "trace-x", "child-span", "missing-parent", past)

	// Advance watermark with a very recent event to make 'past' expire.
	_ = job.Process(ctx, "tenant-1", "trace-ref", "anchor", "", time.Now())

	orphans := job.FlushOrphans(ctx)
	found := false
	for _, o := range orphans {
		if o.SpanID == "child-span" && o.ParentSpanID == "missing-parent" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected orphan child-span to be detected, orphans=%+v", orphans)
	}
}

func TestTraceChainJob_FlushOrphans_SkipsRootSpans(t *testing.T) {
	job := newTestTraceJob()
	ctx := context.Background()

	past := time.Now().Add(-120 * time.Second)
	// Root span has empty parent_span_id.
	_ = job.Process(ctx, "tenant-1", "trace-root", "root-span", "", past)
	_ = job.Process(ctx, "tenant-1", "trace-ref", "anchor", "", time.Now())

	orphans := job.FlushOrphans(ctx)
	for _, o := range orphans {
		if o.SpanID == "root-span" {
			t.Error("root span should never be classified as an orphan")
		}
	}
}

func TestTraceChainJob_FlushOrphans_SkipsCompleteChain(t *testing.T) {
	job := newTestTraceJob()
	ctx := context.Background()

	past := time.Now().Add(-120 * time.Second)
	// Parent and child both submitted — chain is complete, no orphan.
	_ = job.Process(ctx, "tenant-1", "trace-full", "parent-span", "", past)
	_ = job.Process(ctx, "tenant-1", "trace-full", "child-span", "parent-span", past)
	_ = job.Process(ctx, "tenant-1", "trace-ref", "anchor", "", time.Now())

	orphans := job.FlushOrphans(ctx)
	for _, o := range orphans {
		if o.TraceID == "trace-full" {
			t.Errorf("no orphan expected for complete chain, got %+v", o)
		}
	}
}

func TestTraceChainJob_FlushOrphans_DoesNotFlushActiveWindow(t *testing.T) {
	job := newTestTraceJob()
	ctx := context.Background()

	// Submit a RECENT span — it should stay in the window, not be flushed.
	recent := time.Now()
	_ = job.Process(ctx, "tenant-1", "trace-active", "active-span", "missing-parent", recent)

	orphans := job.FlushOrphans(ctx)
	for _, o := range orphans {
		if o.TraceID == "trace-active" {
			t.Error("active (non-expired) window should not be flushed")
		}
	}
}

func TestTraceChainJob_FlushOrphans_CleansUpFlushedTraces(t *testing.T) {
	job := newTestTraceJob()
	ctx := context.Background()

	past := time.Now().Add(-120 * time.Second)
	_ = job.Process(ctx, "tenant-1", "trace-gone", "s1", "missing", past)
	_ = job.Process(ctx, "tenant-1", "trace-ref", "anchor", "", time.Now())

	_ = job.FlushOrphans(ctx)

	job.window.mu.Lock()
	defer job.window.mu.Unlock()
	if _, exists := job.window.byTrace["trace-gone"]; exists {
		t.Error("expected flushed trace to be removed from the window map")
	}
}

// ── Concurrency test ──────────────────────────────────────────────────────────

func TestTraceChainJob_ConcurrentAccess(t *testing.T) {
	job := newTestTraceJob()
	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 500; i++ {
			_ = job.Process(ctx, "t", "trace-concurrent", "s", "", time.Now())
		}
		close(done)
	}()
	go func() {
		for i := 0; i < 100; i++ {
			_ = job.FlushOrphans(ctx)
		}
	}()
	<-done
}
