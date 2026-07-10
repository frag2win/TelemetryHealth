package streaming

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// serviceActivity tracks status and baseline configuration for a service (PRD §8.3).
type serviceActivity struct {
	LastSeenAt        time.Time
	BaselineExpected  bool
	ErrorCount        uint64
	TotalCount        uint64
}

// CoverageJob detects telemetry coverage holes and sampling-rate drift (PRD §8.3, Improvement #14).
type CoverageJob struct {
	logger           *zap.Logger
	mu               sync.Mutex
	registry         map[string]serviceActivity // key: tenantID + ":" + serviceName
	gracePeriod      time.Duration             // default 10 min
}

func NewCoverageJob(logger *zap.Logger) *CoverageJob {
	return &CoverageJob{
		logger:      logger,
		registry:    make(map[string]serviceActivity),
		gracePeriod: 10 * time.Minute,
	}
}

// Observe records service activity and tracks counts for sampling drift detection.
func (j *CoverageJob) Observe(ctx context.Context, tenantID, service string, isError bool, ts time.Time) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	key := tenantID + ":" + service
	act, exists := j.registry[key]
	if !exists {
		act = serviceActivity{
			BaselineExpected: true, // Auto-register as baseline (PRD §8.3)
		}
	}

	act.LastSeenAt = ts
	act.TotalCount++
	if isError {
		act.ErrorCount++
	}

	j.registry[key] = act
	return nil
}

// CheckCoverageGaps identifies services that have gone silent beyond the grace period (PRD §8.3).
func (j *CoverageJob) CheckCoverageGaps(ctx context.Context, now time.Time) []string {
	j.mu.Lock()
	defer j.mu.Unlock()

	var silentServices []string
	for key, act := range j.registry {
		if act.BaselineExpected && now.Sub(act.LastSeenAt) > j.gracePeriod {
			j.logger.Warn("Coverage gap detected: baseline service has stopped emitting telemetry",
				zap.String("tenant_service", key),
				zap.Time("last_seen", act.LastSeenAt),
				zap.Duration("duration_silent", now.Sub(act.LastSeenAt)),
			)
			silentServices = append(silentServices, key)
		}
	}
	return silentServices
}

// DetectSamplingDrift compares expected error rates to actual error rates and warns on disproportionate drops.
func (j *CoverageJob) DetectSamplingDrift(ctx context.Context, expectedErrorRate float64) []string {
	j.mu.Lock()
	defer j.mu.Unlock()

	var driftedServices []string
	for key, act := range j.registry {
		if act.TotalCount < 100 {
			continue // skip small sample sizes
		}
		actualRate := float64(act.ErrorCount) / float64(act.TotalCount)
		// If actual error rate is significantly lower than baseline/expected rate, sampling is dropping errors.
		if actualRate < expectedErrorRate*0.5 {
			j.logger.Warn("Sampling gap detected: error traces are dropped disproportionately",
				zap.String("tenant_service", key),
				zap.Float64("expected_rate", expectedErrorRate),
				zap.Float64("actual_rate", actualRate),
			)
			driftedServices = append(driftedServices, key)
		}
	}
	return driftedServices
}
