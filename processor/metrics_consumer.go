package processor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

type metricsConsumer struct {
	baseConsumer
	next   consumer.Metrics
	logger *zap.Logger
}

func newMetricsConsumer(set processor.Settings, cfg component.Config, next consumer.Metrics) (processor.Metrics, error) {
	bc, err := newBaseConsumer(cfg, set.Logger)
	if err != nil {
		return nil, err
	}
	return &metricsConsumer{
		baseConsumer: bc,
		next:         next,
		logger:       set.Logger,
	}, nil
}

func (c *metricsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (c *metricsConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	if err := c.cb.Execute(ctx, func(ctx context.Context) error {
		// Coverage tracking for metrics
		return nil
	}); err != nil {
		c.logger.Error("metrics consumer execution failed, failing open", zap.Error(err))
	}

	return c.next.ConsumeMetrics(ctx, md)
}

func (c *metricsConsumer) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (c *metricsConsumer) Shutdown(ctx context.Context) error {
	return nil
}
