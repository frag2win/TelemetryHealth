package kafka

import (
	"context"
	"encoding/json"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

const (
	TopicCardinality = "telemetry.cardinality"
	TopicOrphan      = "telemetry.orphan"
	TopicCoverage    = "telemetry.coverage"
)

// CardinalityEvent is the message published when a cardinality observation is made.
type CardinalityEvent struct {
	TenantID     string    `json:"tenant_id"`
	Service      string    `json:"service"`
	AttributeKey string    `json:"attribute_key"`
	UniqueValues uint64    `json:"unique_values"`
	Timestamp    time.Time `json:"timestamp"`
}

// OrphanEvent is published when a trace span has no matching parent.
type OrphanEvent struct {
	TenantID     string    `json:"tenant_id"`
	TraceID      string    `json:"trace_id"`
	SpanID       string    `json:"span_id"`
	ParentSpanID string    `json:"parent_span_id"`
	CollectorID  string    `json:"collector_id"`
	DetectedAt   time.Time `json:"detected_at"`
}

// CoverageEvent is published per heartbeat for each active service.
type CoverageEvent struct {
	TenantID   string    `json:"tenant_id"`
	Service    string    `json:"service"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

// Producer wraps kafka-go writers for each topic.
type Producer struct {
	cardinality *kafkago.Writer
	orphan      *kafkago.Writer
	coverage    *kafkago.Writer
	logger      *zap.Logger
}

func NewProducer(brokers []string, logger *zap.Logger) *Producer {
	makeWriter := func(topic string) *kafkago.Writer {
		return &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafkago.LeastBytes{},
			BatchTimeout: 10 * time.Millisecond,
			RequiredAcks: kafkago.RequireOne,
		}
	}

	return &Producer{
		cardinality: makeWriter(TopicCardinality),
		orphan:      makeWriter(TopicOrphan),
		coverage:    makeWriter(TopicCoverage),
		logger:      logger,
	}
}

func (p *Producer) PublishCardinality(ctx context.Context, event CardinalityEvent) error {
	return p.publish(ctx, p.cardinality, event.TenantID, event)
}

func (p *Producer) PublishOrphan(ctx context.Context, event OrphanEvent) error {
	return p.publish(ctx, p.orphan, event.TenantID, event)
}

func (p *Producer) PublishCoverage(ctx context.Context, event CoverageEvent) error {
	return p.publish(ctx, p.coverage, event.TenantID, event)
}

func (p *Producer) publish(ctx context.Context, w *kafkago.Writer, key string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	err = w.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(key),
		Value: data,
		Time:  time.Now(),
	})
	if err != nil {
		p.logger.Error("kafka publish failed", zap.String("topic", w.Topic), zap.Error(err))
		return err
	}
	return nil
}

func (p *Producer) Close() {
	p.cardinality.Close()
	p.orphan.Close()
	p.coverage.Close()
}
