package streaming

import (
	"testing"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
	"go.uber.org/zap"
)

func newTestHealthScoreJob() *HealthScoreJob {
	return NewHealthScoreJob(zap.NewNop())
}

var defaultWeights = telemetry.DefaultWeights()

// ── Compute() correctness tests ───────────────────────────────────────────────

func TestHealthScoreJob_Compute_PerfectHealth(t *testing.T) {
	job := newTestHealthScoreJob()
	score := job.Compute(0, 0, 0, defaultWeights)
	if score != 100.0 {
		t.Errorf("expected score=100, got %.2f", score)
	}
}

func TestHealthScoreJob_Compute_FullCardinality(t *testing.T) {
	job := newTestHealthScoreJob()
	// cardinality=100% violation, no orphan, no coverage drop.
	score := job.Compute(1.0, 0, 0, defaultWeights)
	// Cardinality weight=0.20, so score = 100 - (0.20*100*1.0) = 80.
	expected := 80.0
	if score != expected {
		t.Errorf("expected score=%.1f, got %.2f", expected, score)
	}
}

func TestHealthScoreJob_Compute_FullOrphanRate(t *testing.T) {
	job := newTestHealthScoreJob()
	// orphan=100%, no cardinality, no coverage.
	score := job.Compute(0, 1.0, 0, defaultWeights)
	// Orphan weight=0.30, so score = 100 - (0.30*100*1.0) = 70.
	expected := 70.0
	if score != expected {
		t.Errorf("expected score=%.1f, got %.2f", expected, score)
	}
}

func TestHealthScoreJob_Compute_FullCoverageDrop(t *testing.T) {
	job := newTestHealthScoreJob()
	// coverage=100%, no cardinality, no orphan.
	score := job.Compute(0, 0, 1.0, defaultWeights)
	// Coverage weight=0.50, so score = 100 - (0.50*100*1.0) = 50.
	expected := 50.0
	if score != expected {
		t.Errorf("expected score=%.1f, got %.2f", expected, score)
	}
}

func TestHealthScoreJob_Compute_AllViolations_ClampedToZero(t *testing.T) {
	job := newTestHealthScoreJob()
	// All at 100% violation — score should be clamped to 0.
	score := job.Compute(1.0, 1.0, 1.0, defaultWeights)
	if score < 0 {
		t.Errorf("score should be clamped to 0, got %.2f", score)
	}
	// With default weights: 100 - (20+30+50) = 0
	if score != 0.0 {
		t.Errorf("expected score=0 with all 100%% violations, got %.2f", score)
	}
}

func TestHealthScoreJob_Compute_CustomWeights(t *testing.T) {
	job := newTestHealthScoreJob()
	weights := telemetry.TenantWeights{
		CardinalityWeight: 0.10,
		OrphanWeight:      0.10,
		CoverageWeight:    0.80,
	}
	// 100% coverage drop with custom weights: 100 - (0.80*100) = 20.
	score := job.Compute(0, 0, 1.0, weights)
	expected := 20.0
	if score != expected {
		t.Errorf("expected score=%.1f with custom weights, got %.2f", expected, score)
	}
}

func TestHealthScoreJob_Compute_PartialViolations(t *testing.T) {
	job := newTestHealthScoreJob()
	// 50% orphan rate: 100 - (0.30*100*0.5) = 85.
	score := job.Compute(0, 0.5, 0, defaultWeights)
	expected := 85.0
	if score != expected {
		t.Errorf("expected score=%.1f, got %.2f", expected, score)
	}
}

func TestHealthScoreJob_Compute_NeverExceeds100(t *testing.T) {
	job := newTestHealthScoreJob()
	score := job.Compute(0, 0, 0, telemetry.TenantWeights{
		CardinalityWeight: 0,
		OrphanWeight:      0,
		CoverageWeight:    0,
	})
	if score > 100.0 {
		t.Errorf("score should never exceed 100, got %.2f", score)
	}
}

func TestHealthScoreJob_Compute_NeverNegative(t *testing.T) {
	job := newTestHealthScoreJob()
	// Over-weighted violations — score must not go below 0.
	weights := telemetry.TenantWeights{
		CardinalityWeight: 0.5,
		OrphanWeight:      0.5,
		CoverageWeight:    0.5,
	}
	score := job.Compute(1.0, 1.0, 1.0, weights)
	if score < 0 {
		t.Errorf("score should never be negative, got %.2f", score)
	}
}
