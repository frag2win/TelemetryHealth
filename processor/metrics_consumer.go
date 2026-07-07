package processor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
)

type metricsConsumer struct {
	baseConsumer
	next consumer.Metrics
}

func newMetricsConsumer(set processor.Settings, cfg component.Config, next consumer.Metrics) (processor.Metrics, error) {
	return &metricsConsumer{
		baseConsumer: newBaseConsumer(cfg, set.Logger),
		next:         next,
	}, nil
}

func (c *metricsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (c *metricsConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	_ = c.cb.Execute(ctx, func(ctx context.Context) error {
		// Coverage tracking for metrics
		return nil
	})

	return c.next.ConsumeMetrics(ctx, md)
}

func (c *metricsConsumer) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (c *metricsConsumer) Shutdown(ctx context.Context) error {
	return nil
}
