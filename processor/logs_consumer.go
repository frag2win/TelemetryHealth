package processor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
)

type logsConsumer struct {
	baseConsumer
	next consumer.Logs
}

func newLogsConsumer(set processor.Settings, cfg component.Config, next consumer.Logs) (processor.Logs, error) {
	return &logsConsumer{
		baseConsumer: newBaseConsumer(cfg, set.Logger),
		next:         next,
	}, nil
}

func (c *logsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (c *logsConsumer) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	_ = c.cb.Execute(ctx, func(ctx context.Context) error {
		// Coverage tracking for logs
		return nil
	})

	return c.next.ConsumeLogs(ctx, ld)
}

func (c *logsConsumer) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (c *logsConsumer) Shutdown(ctx context.Context) error {
	return nil
}
