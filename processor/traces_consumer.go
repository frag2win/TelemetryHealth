package processor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

type tracesConsumer struct {
	baseConsumer
	next   consumer.Traces
	logger *zap.Logger
}

func newTracesConsumer(set processor.Settings, cfg component.Config, next consumer.Traces) (processor.Traces, error) {
	bc, err := newBaseConsumer(cfg, set.Logger)
	if err != nil {
		return nil, err
	}
	return &tracesConsumer{
		baseConsumer: bc,
		next:         next,
		logger:       set.Logger,
	}, nil
}

func (c *tracesConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (c *tracesConsumer) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	// 1. Run detection logic wrapped in circuit breaker (fail-open)
	if err := c.cb.Execute(ctx, func(ctx context.Context) error {
		// Health telemetry extraction logic (cardinality, orphans, coverage)
		// Never blocks or mutates original traces.
		return nil
	}); err != nil {
		c.logger.Error("traces consumer execution failed, failing open", zap.Error(err))
	}

	// 2. Pass data to next consumer unmodified
	return c.next.ConsumeTraces(ctx, td)
}

func (c *tracesConsumer) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (c *tracesConsumer) Shutdown(ctx context.Context) error {
	return nil
}
