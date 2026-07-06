package cardinality

import (
	"testing"

	"go.uber.org/zap"
)

func TestTracker_ObserveAndFlush(t *testing.T) {
	tracker := NewTracker(1024*1024, 100, zap.NewNop())

	tracker.Observe("svc-a", "user_id", "u1")
	tracker.Observe("svc-a", "user_id", "u2")
	tracker.Observe("svc-a", "user_id", "u1") // duplicate

	sketches := tracker.Flush()
	if len(sketches) != 1 {
		t.Fatalf("expected 1 service, got %d", len(sketches))
	}
	
	sketch := sketches["svc-a"]["user_id"]
	if sketch == nil {
		t.Fatal("expected sketch for svc-a/user_id")
	}
	
	est := sketch.Estimate()
	if est != 2 {
		t.Fatalf("expected estimate 2, got %d", est)
	}

	// After flush, tracker should be empty
	sketches2 := tracker.Flush()
	if len(sketches2) != 0 {
		t.Fatalf("expected empty tracker after flush, got %d services", len(sketches2))
	}
}

func TestTracker_KeySpaceExplosion(t *testing.T) {
	tracker := NewTracker(1024*1024, 2, zap.NewNop())

	tracker.Observe("svc-a", "key1", "val")
	tracker.Observe("svc-a", "key2", "val")
	tracker.Observe("svc-a", "key3", "val") // should be dropped

	sketches := tracker.Flush()
	if len(sketches["svc-a"]) != 2 {
		t.Fatalf("expected 2 keys tracked, got %d", len(sketches["svc-a"]))
	}
	if _, ok := sketches["svc-a"]["key3"]; ok {
		t.Fatal("expected key3 to be dropped")
	}
}

func TestTracker_MemoryBound(t *testing.T) {
	// A single sketch takes ~16KB. Limit to 20KB to allow 1 sketch but not 2.
	tracker := NewTracker(20000, 100, zap.NewNop())

	tracker.Observe("svc-a", "key1", "val")
	tracker.Observe("svc-b", "key1", "val") // should be dropped due to memory

	sketches := tracker.Flush()
	count := 0
	for _, svcMap := range sketches {
		count += len(svcMap)
	}
	if count != 1 {
		t.Fatalf("expected 1 sketch total, got %d", count)
	}
}
