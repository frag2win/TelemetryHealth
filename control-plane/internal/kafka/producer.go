package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

const (
	TopicCardinality = "telemetry.cardinality"
	TopicOrphan      = "telemetry.orphan"
	TopicCoverage    = "telemetry.coverage"
	TopicRawSpan     = "telemetry.rawspan" // Phase 3 Mock Data
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

// RawSpanEvent is published for building the mock topology (Phase 3).
type RawSpanEvent struct {
	TenantID      string    `json:"tenant_id"`
	TraceID       string    `json:"trace_id"`
	SpanID        string    `json:"span_id"`
	ParentSpanID  string    `json:"parent_span_id"`
	ServiceName   string    `json:"service_name"`
	OperationName string    `json:"operation_name"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Status        string    `json:"status"`
	Attributes    string    `json:"attributes"`
}

// Producer wraps kafka-go writers for each topic.
type Producer struct {
	cardinality *kafkago.Writer
	orphan      *kafkago.Writer
	coverage    *kafkago.Writer
	rawspan     *kafkago.Writer
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
		rawspan:     makeWriter(TopicRawSpan),
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

func (p *Producer) PublishRawSpan(ctx context.Context, event RawSpanEvent) error {
	return p.publish(ctx, p.rawspan, event.TenantID, event)
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

func (p *Producer) Close() error {
	var errs []error
	if err := p.cardinality.Close(); err != nil {
		p.logger.Error("failed to close cardinality writer", zap.Error(err))
		errs = append(errs, err)
	}
	if err := p.orphan.Close(); err != nil {
		p.logger.Error("failed to close orphan writer", zap.Error(err))
		errs = append(errs, err)
	}
	if err := p.coverage.Close(); err != nil {
		p.logger.Error("failed to close coverage writer", zap.Error(err))
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing producer writers: %v", errs)
	}
	return nil
}
