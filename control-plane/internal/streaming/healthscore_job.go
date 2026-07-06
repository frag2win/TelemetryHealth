package streaming

import (
	"go.uber.org/zap"
)

// HealthScoreJob computes the composite Telemetry Health Score.
type HealthScoreJob struct {
	logger *zap.Logger
}

func NewHealthScoreJob(logger *zap.Logger) *HealthScoreJob {
	return &HealthScoreJob{logger: logger}
}

// Compute computes the score using the PRD §8.4 formula:
// HealthScore = 100 - Σ(weight_i × normalized_signal_i)
func (j *HealthScoreJob) Compute(cardinalityViolation, orphanRate, coverageDrop float64) float64 {
	// Default weights: cardinality 20%, orphan 30%, coverage 50%
	score := 100.0 - (0.20*cardinalityViolation + 0.30*orphanRate + 0.50*coverageDrop)
	if score < 0 {
		return 0
	}
	return score
}
