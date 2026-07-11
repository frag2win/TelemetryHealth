package streaming

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestCoverageJob() *CoverageJob {
	return NewCoverageJob(zap.NewNop())
}

// ── Observe() tests ───────────────────────────────────────────────────────────

func TestCoverageJob_Observe_RegistersService(t *testing.T) {
	job := newTestCoverageJob()
	ctx := context.Background()
	now := time.Now()

	err := job.Observe(ctx, "tenant-1", "payments-api", false, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job.mu.Lock()
	defer job.mu.Unlock()
	act, ok := job.registry["tenant-1:payments-api"]
	if !ok {
		t.Fatal("expected service to be registered")
	}
	if !act.BaselineExpected {
		t.Error("expected BaselineExpected to be true for auto-registered service")
	}
	if act.TotalCount != 1 {
		t.Errorf("expected TotalCount=1, got %d", act.TotalCount)
	}
}

func TestCoverageJob_Observe_CountsErrors(t *testing.T) {
	job := newTestCoverageJob()
	ctx := context.Background()
	now := time.Now()

	_ = job.Observe(ctx, "tenant-1", "svc", false, now)
	_ = job.Observe(ctx, "tenant-1", "svc", true, now)
	_ = job.Observe(ctx, "tenant-1", "svc", true, now)

	job.mu.Lock()
	defer job.mu.Unlock()
	act := job.registry["tenant-1:svc"]
	if act.TotalCount != 3 {
		t.Errorf("expected TotalCount=3, got %d", act.TotalCount)
	}
	if act.ErrorCount != 2 {
		t.Errorf("expected ErrorCount=2, got %d", act.ErrorCount)
	}
}

func TestCoverageJob_Observe_UpdatesLastSeenAt(t *testing.T) {
	job := newTestCoverageJob()
	ctx := context.Background()
	t1 := time.Now()
	t2 := t1.Add(5 * time.Second)

	_ = job.Observe(ctx, "t1", "svc", false, t1)
	_ = job.Observe(ctx, "t1", "svc", false, t2)

	job.mu.Lock()
	defer job.mu.Unlock()
	act := job.registry["t1:svc"]
	if !act.LastSeenAt.Equal(t2) {
		t.Errorf("expected LastSeenAt=%v, got %v", t2, act.LastSeenAt)
	}
}

// ── CheckCoverageGaps() tests ─────────────────────────────────────────────────

func TestCoverageJob_CheckCoverageGaps_SilentService(t *testing.T) {
	job := newTestCoverageJob()
	ctx := context.Background()

	// Service that last emitted 20 minutes ago (beyond 10 min grace period).
	past := time.Now().Add(-20 * time.Minute)
	_ = job.Observe(ctx, "tenant-1", "silent-svc", false, past)

	gaps := job.CheckCoverageGaps(ctx, time.Now())
	if len(gaps) != 1 {
		t.Errorf("expected 1 gap, got %d: %v", len(gaps), gaps)
	}
	if gaps[0] != "tenant-1:silent-svc" {
		t.Errorf("unexpected gap key: %s", gaps[0])
	}
}

func TestCoverageJob_CheckCoverageGaps_ActiveService(t *testing.T) {
	job := newTestCoverageJob()
	ctx := context.Background()

	// Service that emitted 30 seconds ago (within grace period).
	recent := time.Now().Add(-30 * time.Second)
	_ = job.Observe(ctx, "tenant-1", "active-svc", false, recent)

	gaps := job.CheckCoverageGaps(ctx, time.Now())
	if len(gaps) != 0 {
		t.Errorf("expected 0 gaps for active service, got %d", len(gaps))
	}
}

func TestCoverageJob_CheckCoverageGaps_ExactlyAtGracePeriod(t *testing.T) {
	job := newTestCoverageJob()
	ctx := context.Background()

	now := time.Now()
	// Service that emitted exactly at the grace period boundary.
	exactly := now.Add(-10 * time.Minute)
	_ = job.Observe(ctx, "tenant-1", "boundary-svc", false, exactly)

	// Should NOT be flagged (must be strictly greater than grace period).
	gaps := job.CheckCoverageGaps(ctx, now)
	if len(gaps) != 0 {
		t.Errorf("expected 0 gaps at boundary, got %d", len(gaps))
	}
}

func TestCoverageJob_CheckCoverageGaps_MultiTenant(t *testing.T) {
	job := newTestCoverageJob()
	ctx := context.Background()
	past := time.Now().Add(-20 * time.Minute)

	// Two different tenants, each with a silent service.
	_ = job.Observe(ctx, "tenant-A", "svc", false, past)
	_ = job.Observe(ctx, "tenant-B", "svc", false, past)

	gaps := job.CheckCoverageGaps(ctx, time.Now())
	if len(gaps) != 2 {
		t.Errorf("expected 2 gaps (one per tenant), got %d", len(gaps))
	}
}

// ── DetectSamplingDrift() tests ────────────────────────────────────────────────

func TestCoverageJob_DetectSamplingDrift_DetectsDrop(t *testing.T) {
	job := newTestCoverageJob()
	ctx := context.Background()
	now := time.Now()

	// Simulate 200 spans: 1% error rate (2 errors), but expected 10% error rate.
	for i := 0; i < 200; i++ {
		isError := i < 2 // first 2 are errors
		_ = job.Observe(ctx, "tenant-1", "checkout", isError, now)
	}

	drifted := job.DetectSamplingDrift(ctx, 0.10) // expect 10% error rate
	if len(drifted) == 0 {
		t.Error("expected sampling drift to be detected when actual rate (1%) << expected (10%)")
	}
}

func TestCoverageJob_DetectSamplingDrift_NoDrift(t *testing.T) {
	job := newTestCoverageJob()
	ctx := context.Background()
	now := time.Now()

	// Simulate 200 spans: 10% error rate, expected 10% — no drift.
	for i := 0; i < 200; i++ {
		isError := i < 20
		_ = job.Observe(ctx, "tenant-1", "payments", isError, now)
	}

	drifted := job.DetectSamplingDrift(ctx, 0.10)
	if len(drifted) != 0 {
		t.Errorf("expected no drift when actual matches expected, got %v", drifted)
	}
}

func TestCoverageJob_DetectSamplingDrift_SkipsSmallSample(t *testing.T) {
	job := newTestCoverageJob()
	ctx := context.Background()
	now := time.Now()

	// Only 50 observations — below the 100-span minimum for meaningful drift detection.
	for i := 0; i < 50; i++ {
		_ = job.Observe(ctx, "tenant-1", "small-svc", false, now)
	}

	drifted := job.DetectSamplingDrift(ctx, 0.10)
	if len(drifted) != 0 {
		t.Errorf("expected no drift detection for small samples, got %v", drifted)
	}
}
