package failopen

import (
	"context"
	"errors"
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
