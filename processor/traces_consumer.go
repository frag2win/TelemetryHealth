package processor

import (
	"context"
	"time"

	"github.com/frag2win/TelemetryHealth/processor/failopen"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
)

type tracesConsumer struct {
	next consumer.Traces
	cb   *failopen.CircuitBreaker
}

func newTracesConsumer(set processor.Settings, cfg component.Config, next consumer.Traces) (processor.Traces, error) {
	cb := failopen.NewCircuitBreaker(5, 30*time.Second, set.Logger)
	return &tracesConsumer{
		next: next,
		cb:   cb,
	}, nil
}

func (c *tracesConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (c *tracesConsumer) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	// 1. Run detection logic wrapped in circuit breaker (fail-open)
	_ = c.cb.Execute(ctx, func(ctx context.Context) error {
		// Health telemetry extraction logic (cardinality, orphans, coverage)
		// Never blocks or mutates original traces.
		return nil
	})

	// 2. Pass data to next consumer unmodified
	return c.next.ConsumeTraces(ctx, td)
}

func (c *tracesConsumer) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (c *tracesConsumer) Shutdown(ctx context.Context) error {
	return nil
}
