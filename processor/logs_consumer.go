package processor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

type logsConsumer struct {
	baseConsumer
	next   consumer.Logs
	logger *zap.Logger
}

func newLogsConsumer(set processor.Settings, cfg component.Config, next consumer.Logs) (processor.Logs, error) {
	bc, err := newBaseConsumer(cfg, set.Logger)
	if err != nil {
		return nil, err
	}
	return &logsConsumer{
		baseConsumer: bc,
		next:         next,
		logger:       set.Logger,
	}, nil
}

func (c *logsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (c *logsConsumer) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	if err := c.cb.Execute(ctx, func(ctx context.Context) error {
		// Coverage tracking for logs
		return nil
	}); err != nil {
		c.logger.Error("logs consumer execution failed, failing open", zap.Error(err))
	}

	return c.next.ConsumeLogs(ctx, ld)
}

func (c *logsConsumer) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (c *logsConsumer) Shutdown(ctx context.Context) error {
	return nil
}
