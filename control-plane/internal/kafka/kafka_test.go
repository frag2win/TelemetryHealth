package kafka_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/kafka"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// TestProducerEncoding verifies events marshal cleanly to JSON as Kafka values.
func TestProducerEncoding(t *testing.T) {
	events := []any{
		kafka.CardinalityEvent{
			TenantID: "tenant-1", Service: "checkout",
			AttributeKey: "user_id", UniqueValues: 1_500_000,
			Timestamp: time.Now(),
		},
		kafka.OrphanEvent{
			TenantID: "tenant-1", TraceID: "abc123", SpanID: "def456",
			ParentSpanID: "000000", CollectorID: "gw-01",
			DetectedAt: time.Now(),
		},
		kafka.CoverageEvent{
			TenantID: "tenant-1", Service: "payments",
			LastSeenAt: time.Now(),
		},
	}

	for _, e := range events {
		data, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("marshal failed for %T: %v", e, err)
		}
		if len(data) == 0 {
			t.Fatalf("empty JSON for %T", e)
		}
	}
}

// TestConsumerOfflineGraceful verifies consumer exits cleanly when context is cancelled.
func TestConsumerOfflineGraceful(t *testing.T) {
	consumer := kafka.NewConsumer(
		[]string{"localhost:19092"}, // unreachable
		"test.topic", "test-group",
		func(ctx context.Context, e kafka.CoverageEvent) error { return nil },
		zap.NewNop(),
	)
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Should return when ctx is cancelled, not panic.
	_ = consumer.Run(ctx)
}

// Ensure kafkago used.
var _ = kafkago.Message{}
