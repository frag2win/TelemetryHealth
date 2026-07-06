package failopen

import (
	"context"
	"fmt"
	"sync/atomic"
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
	state          int32 // atomic
	failureCount   int64 // atomic
	failureLimit   int64
	resetTimeout   time.Duration
	lastFailure    int64 // unix nano, atomic
	logger         *zap.Logger
}

func NewCircuitBreaker(limit int64, timeout time.Duration, logger *zap.Logger) *CircuitBreaker {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CircuitBreaker{
		state:        int32(StateClosed),
		failureLimit: limit,
		resetTimeout: timeout,
		logger:       logger,
	}
}

// Execute runs the given function safely. If the circuit is open, it immediately returns
// without executing the function. If the function panics, it catches the panic, increments
// the failure count, and returns an error (which the processor can log, but must not fail the pipeline).
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	currentState := State(atomic.LoadInt32(&cb.state))

	if currentState == StateOpen {
		last := atomic.LoadInt64(&cb.lastFailure)
		if time.Since(time.Unix(0, last)) >= cb.resetTimeout {
			// Transition to HalfOpen to test
			if atomic.CompareAndSwapInt32(&cb.state, int32(StateOpen), int32(StateHalfOpen)) {
				cb.logger.Warn("Circuit breaker half-open, testing recovery")
				currentState = StateHalfOpen
			} else {
				// Someone else changed it, bypass for now
				return nil
			}
		} else {
			// Still open, bypass
			return nil
		}
	}

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

	if err != nil {
		cb.recordFailure()
		return err
	}

	if currentState == StateHalfOpen {
		cb.reset()
	}
	return nil
}

func (cb *CircuitBreaker) recordFailure() {
	count := atomic.AddInt64(&cb.failureCount, 1)
	atomic.StoreInt64(&cb.lastFailure, time.Now().UnixNano())

	if count >= cb.failureLimit {
		if atomic.CompareAndSwapInt32(&cb.state, int32(StateClosed), int32(StateOpen)) ||
			atomic.CompareAndSwapInt32(&cb.state, int32(StateHalfOpen), int32(StateOpen)) {
			cb.logger.Error("Circuit breaker tripped, bypassing internal processing", zap.Int64("failures", count))
		}
	}
}

func (cb *CircuitBreaker) reset() {
	atomic.StoreInt64(&cb.failureCount, 0)
	if atomic.CompareAndSwapInt32(&cb.state, int32(StateHalfOpen), int32(StateClosed)) {
		cb.logger.Info("Circuit breaker reset, internal processing restored")
	}
}
