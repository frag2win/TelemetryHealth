package kafka

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage/clickhouse"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// WorkerSet runs all three consumer goroutines and writes results to ClickHouse.
type WorkerSet struct {
	brokers    []string
	chClient   *clickhouse.Client
	logger     *zap.Logger
}

func NewWorkerSet(brokers []string, chClient *clickhouse.Client, logger *zap.Logger) *WorkerSet {
	return &WorkerSet{brokers: brokers, chClient: chClient, logger: logger}
}

// Run starts all three consumer goroutines. Blocks until ctx is cancelled.
func (w *WorkerSet) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		w.runCardinalityWorker(ctx)
	}()
	go func() {
		defer wg.Done()
		w.runOrphanWorker(ctx)
	}()
	go func() {
		defer wg.Done()
		w.runCoverageWorker(ctx)
	}()
	go func() {
		defer wg.Done()
		w.runRawSpanWorker(ctx)
	}()

	<-ctx.Done()
	w.logger.Info("WorkerSet shutting down")
	wg.Wait()
	w.logger.Info("WorkerSet shutdown complete")
}

func (w *WorkerSet) runCardinalityWorker(ctx context.Context) {
	consumer := NewConsumer(
		w.brokers, TopicCardinality, "cardinality-worker",
		func(ctx context.Context, events []CardinalityEvent) error {
			batch, err := w.chClient.Conn().PrepareBatch(ctx, `INSERT INTO telemetry_health.cardinality_signal
				(tenant_id, service, attribute_key, window_start, unique_estimate)`)
			if err != nil {
				return fmt.Errorf("prepare cardinality batch: %w", err)
			}
			for _, event := range events {
				if err := batch.Append(
					event.TenantID,
					event.Service,
					event.AttributeKey,
					event.Timestamp,
					event.UniqueValues,
				); err != nil {
					return fmt.Errorf("append cardinality: %w", err)
				}
			}
			if err := batch.Send(); err != nil {
				return fmt.Errorf("send cardinality: %w", err)
			}
			w.logger.Debug("wrote cardinality batch", zap.Int("count", len(events)))
			return nil
		},
		w.logger,
	)
	defer consumer.Close()
	_ = consumer.Run(ctx)
}

func (w *WorkerSet) runOrphanWorker(ctx context.Context) {
	consumer := NewConsumer(
		w.brokers, TopicOrphan, "orphan-worker",
		func(ctx context.Context, events []OrphanEvent) error {
			batch, err := w.chClient.Conn().PrepareBatch(ctx, `INSERT INTO telemetry_health.orphan_signal
				(tenant_id, trace_id, span_id, parent_span_id, collector_id, detected_at)`)
			if err != nil {
				return fmt.Errorf("prepare orphan batch: %w", err)
			}
			for _, event := range events {
				if err := batch.Append(
					event.TenantID,
					event.TraceID,
					event.SpanID,
					event.ParentSpanID,
					event.CollectorID,
					event.DetectedAt,
				); err != nil {
					return fmt.Errorf("append orphan: %w", err)
				}
			}
			if err := batch.Send(); err != nil {
				return fmt.Errorf("send orphan: %w", err)
			}
			w.logger.Debug("wrote orphan batch", zap.Int("count", len(events)))
			return nil
		},
		w.logger,
	)
	defer consumer.Close()
	_ = consumer.Run(ctx)
}

func (w *WorkerSet) runCoverageWorker(ctx context.Context) {
	consumer := NewConsumer(
		w.brokers, TopicCoverage, "coverage-worker",
		func(ctx context.Context, events []CoverageEvent) error {
			batch, err := w.chClient.Conn().PrepareBatch(ctx, `INSERT INTO telemetry_health.coverage_signal
				(tenant_id, service, last_seen_at, baseline_expected)`)
			if err != nil {
				return fmt.Errorf("prepare coverage batch: %w", err)
			}
			for _, event := range events {
				if err := batch.Append(
					event.TenantID,
					event.Service,
					event.LastSeenAt,
					uint8(1),
				); err != nil {
					return fmt.Errorf("append coverage: %w", err)
				}
			}
			if err := batch.Send(); err != nil {
				return fmt.Errorf("send coverage: %w", err)
			}
			w.logger.Debug("wrote coverage batch", zap.Int("count", len(events)))
			return nil
		},
		w.logger,
	)
	defer consumer.Close()
	_ = consumer.Run(ctx)
}

func (w *WorkerSet) runRawSpanWorker(ctx context.Context) {
	consumer := NewConsumer(
		w.brokers, TopicRawSpan, "rawspan-worker",
		func(ctx context.Context, events []RawSpanEvent) error {
			batch, err := w.chClient.Conn().PrepareBatch(ctx, `INSERT INTO telemetry_health.telemetryhealth_trace_index_spans
				(trace_id, span_id, parent_span_id, service_name, operation_name, start_time, end_time, status, attributes, tenant_id)`)
			if err != nil {
				return fmt.Errorf("prepare rawspan batch: %w", err)
			}
			for _, event := range events {
				if err := batch.Append(
					event.TraceID,
					event.SpanID,
					event.ParentSpanID,
					event.ServiceName,
					event.OperationName,
					event.StartTime,
					event.EndTime,
					event.Status,
					event.Attributes,
					event.TenantID,
				); err != nil {
					return fmt.Errorf("append rawspan: %w", err)
				}
			}
			if err := batch.Send(); err != nil {
				return fmt.Errorf("send rawspan: %w", err)
			}
			w.logger.Debug("wrote rawspan batch", zap.Int("count", len(events)))
			return nil
		},
		w.logger,
	)
	defer consumer.Close()
	_ = consumer.Run(ctx)
}

// Ensure kafkago is used.
var _ = kafkago.Message{}

// reuse strconv here instead.
var _ = strconv.Itoa
var _ = time.Now
