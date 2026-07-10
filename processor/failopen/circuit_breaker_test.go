package failopen

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)


func TestCircuitBreaker_PanicRecovery(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Second, zap.NewNop())

	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		panic("test panic")
	})

	if err == nil {
		t.Fatal("expected error from panic, got nil")
	}
	if cb.failureCount != 1 {
		t.Fatalf("expected 1 failure, got %d", cb.failureCount)
	}
}

func TestCircuitBreaker_TripAndReset(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond, zap.NewNop())

	// Fail 1
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("err 1")
	})
	if State(cb.state) != StateClosed {
		t.Fatal("expected closed state")
	}

	// Fail 2 - should trip
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("err 2")
	})
	if State(cb.state) != StateOpen {
		t.Fatal("expected open state")
	}

	// Execution should bypass
	executed := false
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		executed = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error when bypassed, got %v", err)
	}
	if executed {
		t.Fatal("expected function to not execute when circuit is open")
	}

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Next execution should transition to half-open and run
	executed = false
	err = cb.Execute(context.Background(), func(ctx context.Context) error {
		executed = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error on success, got %v", err)
	}
	if !executed {
		t.Fatal("expected function to execute in half-open state")
	}
	if State(cb.state) != StateClosed {
		t.Fatalf("expected closed state after successful execution, got %d", cb.state)
	}
}

func TestCircuitBreaker_StartsInClosedState(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second, zap.NewNop())
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state != StateClosed {
		t.Errorf("expected initial state=Closed, got %d", cb.state)
	}
}

func TestCircuitBreaker_DefaultLimitIsApplied(t *testing.T) {
	cb := NewCircuitBreaker(0, 30*time.Second, zap.NewNop())
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.failureLimit != 5 {
		t.Errorf("expected default failureLimit=5, got %d", cb.failureLimit)
	}
}

func TestCircuitBreaker_PanicCountsAsFailure(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second, zap.NewNop())
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			panic("controlled test panic")
		})
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state != StateOpen {
		t.Errorf("expected state=Open after 3 panics, got %d", cb.state)
	}
}

func TestCircuitBreaker_RemainsOpenIfHalfOpenFails(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Millisecond, zap.NewNop())
	ctx := context.Background()
	_ = cb.Execute(ctx, func(ctx context.Context) error { return errors.New("trip") })
	time.Sleep(20 * time.Millisecond)
	_ = cb.Execute(ctx, func(ctx context.Context) error { return errors.New("still failing") })
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state != StateOpen {
		t.Errorf("expected state=Open after half-open failure, got %d", cb.state)
	}
}

func TestCircuitBreaker_SuccessKeepsClosed(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second, zap.NewNop())
	for i := 0; i < 10; i++ {
		err := cb.Execute(context.Background(), func(ctx context.Context) error { return nil })
		if err != nil {
			t.Fatalf("unexpected error on successful execution: %v", err)
		}
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state != StateClosed {
		t.Errorf("expected state=Closed after all successes, got %d", cb.state)
	}
}

func TestCircuitBreaker_ConcurrentAccess_NoPanic(t *testing.T) {
	cb := NewCircuitBreaker(100, 30*time.Second, zap.NewNop())
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = cb.Execute(ctx, func(ctx context.Context) error {
				if n%7 == 0 {
					return errors.New("occasional failure")
				}
				return nil
			})
		}(i)
	}
	wg.Wait()
}
