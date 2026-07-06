package streaming

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CoverageJob detects services that stop emitting telemetry.
type CoverageJob struct {
	logger   *zap.Logger
	registry map[string]time.Time
	mu       sync.Mutex
}

func NewCoverageJob(logger *zap.Logger) *CoverageJob {
	return &CoverageJob{
		logger:   logger,
		registry: make(map[string]time.Time),
	}
}

// Observe records a service as active.
func (j *CoverageJob) Observe(ctx context.Context, tenantID, service string, ts time.Time) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.registry[tenantID+":"+service] = ts
	return nil
}
