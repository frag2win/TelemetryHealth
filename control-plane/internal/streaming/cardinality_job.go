package streaming

import (
	"context"
	"time"

	"github.com/axiomhq/hyperloglog"
	"go.uber.org/zap"
)

// CardinalityJob merges HLL sketches arriving from the collector fleet.
type CardinalityJob struct {
	logger *zap.Logger
	// State store for windowed sketches
	state map[string]*hyperloglog.Sketch
}

func NewCardinalityJob(logger *zap.Logger) *CardinalityJob {
	return &CardinalityJob{
		logger: logger,
		state:  make(map[string]*hyperloglog.Sketch),
	}
}

// Process merges an incoming sketch into the central state.
func (j *CardinalityJob) Process(ctx context.Context, tenantID, service, key string, sketch *hyperloglog.Sketch, timestamp time.Time) error {
	stateKey := tenantID + ":" + service + ":" + key
	if existing, ok := j.state[stateKey]; ok {
		// Exact-merge centrally as required by PRD §8.1
		existing.Merge(sketch)
	} else {
		j.state[stateKey] = sketch
	}
	return nil
}
