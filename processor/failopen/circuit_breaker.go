package failopen

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// State represents the circuit breaker state.
type State int32

const (
	StateClosed   State = iota // Normal operation, tracking enabled
	StateOpen                  // Tripped, fast fail (bypass tracking)
	StateHalfOpen              // Testing if system has recovered
)

// CircuitBreaker prevents panics or continuous errors from blocking the telemetry pipeline.
// When tripped, it bypasses the internal processing and simply passes the telemetry through.
type CircuitBreaker struct {
	mu           sync.Mutex
	state        State
	failureCount int64
	failureLimit int64
	resetTimeout time.Duration
	lastFailure  time.Time
	logger       *zap.Logger
}

func NewCircuitBreaker(limit int64, timeout time.Duration, logger *zap.Logger) *CircuitBreaker {
	if logger == nil {
		logger = zap.NewNop()
	}
	if limit <= 0 {
		limit = 5 // default fallback limit
	}
	return &CircuitBreaker{
		state:        StateClosed,
		failureLimit: limit,
		resetTimeout: timeout,
		logger:       logger,
	}
}

// Execute runs the given function safely. If the circuit is open, it immediately returns
// without executing the function. If the function panics, it catches the panic, increments
// the failure count, and returns an error (which the processor can log, but must not fail the pipeline).
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	cb.mu.Lock()
	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) >= cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.failureCount = 0
			cb.logger.Warn("Circuit breaker half-open, testing recovery")
		} else {
			cb.mu.Unlock()
			return nil // Fail open
		}
	}
	stateAtStart := cb.state
	cb.mu.Unlock()

	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic caught in fail-open execution: %v", r)
				cb.logger.Error("Processor panic recovered", zap.Any("panic", r))
			}
		}()
		err = fn(ctx)
	}()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailure = time.Now()
		if cb.failureCount >= cb.failureLimit {
			if cb.state != StateOpen {
				cb.state = StateOpen
				cb.logger.Error("Circuit breaker tripped, bypassing internal processing", zap.Int64("failures", cb.failureCount))
			}
		}
		return err
	}

	if stateAtStart == StateHalfOpen && cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.failureCount = 0
		cb.logger.Info("Circuit breaker reset, internal processing restored")
	}

	return nil
}

