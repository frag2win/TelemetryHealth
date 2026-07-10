package processor

import (
	"context"
	"time"

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
	// Step 1: Extract structural tuples BEFORE sampling decisions are applied.
	// Ship [trace_id, span_id, parent_span_id] to EXP2 regardless of sampling outcome.
	// This prevents the sampling correlation paradox (PRD §8.2).
	if err := c.cb.Execute(ctx, func(ctx context.Context) error {
		rss := td.ResourceSpans()
		for i := 0; i < rss.Len(); i++ {
			rs := rss.At(i)
			serviceName := ""
			if v, ok := rs.Resource().Attributes().Get("service.name"); ok {
				serviceName = v.Str()
			}
			sss := rs.ScopeSpans()
			for j := 0; j < sss.Len(); j++ {
				ss := sss.At(j)
				spans := ss.Spans()
				for k := 0; k < spans.Len(); k++ {
					span := spans.At(k)
					sig := HealthSignal{
						TraceID:      span.TraceID().String(),
						SpanID:       span.SpanID().String(),
						ParentSpanID: span.ParentSpanID().String(),
						ServiceName:  serviceName,
					}
					// EmitHealthSignal drops if queue is full — never blocks (PRD §10)
					c.EmitHealthSignal(ctx, sig)
				}
			}
		}
		return nil
	}); err != nil {
		c.logger.Error("traces consumer health extraction failed, failing open", zap.Error(err))
	}

	// Step 2: Pass the original trace data to the next consumer (EXP1) unmodified.
	// Sampling, if any, is applied downstream — health extraction already happened above.
	return c.next.ConsumeTraces(ctx, td)
}

func (c *tracesConsumer) Start(ctx context.Context, host component.Host) error {
	c.logger.Info("TracesConsumer started", zap.Time("at", time.Now()))
	return nil
}

func (c *tracesConsumer) Shutdown(ctx context.Context) error {
	c.logger.Info("TracesConsumer shutting down")
	return nil
}
