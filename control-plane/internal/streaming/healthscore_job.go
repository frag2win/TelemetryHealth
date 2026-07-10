package streaming

import (
	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
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
func (j *HealthScoreJob) Compute(cardinalityViolation, orphanRate, coverageDrop float64, weights telemetry.TenantWeights) float64 {
	return telemetry.CalculateHealthScoreFromViolations(cardinalityViolation, orphanRate, coverageDrop, weights)
}
