package kafka

import (
	"context"
	"encoding/json"

	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Handler is a function that processes a decoded Kafka message.
type Handler[T any] func(ctx context.Context, event T) error

// Consumer reads from a single Kafka topic and dispatches to a typed handler.
type Consumer[T any] struct {
	reader  *kafkago.Reader
	handler Handler[T]
	logger  *zap.Logger
}

func NewConsumer[T any](brokers []string, topic, groupID string, handler Handler[T], logger *zap.Logger) *Consumer[T] {
	return &Consumer[T]{
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       1e6, // 1 MB
			CommitInterval: 0,   // explicit commit after processing
		}),
		handler: handler,
		logger:  logger,
	}
}

// Run starts consuming messages until ctx is cancelled.
func (c *Consumer[T]) Run(ctx context.Context) error {
	c.logger.Info("Kafka consumer started", zap.String("topic", c.reader.Config().Topic))

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // normal shutdown
			}
			c.logger.Error("fetch message failed", zap.Error(err))
			continue
		}

		var event T
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.Error("decode failed, skipping", zap.Error(err))
			_ = c.reader.CommitMessages(ctx, msg)
			continue
		}

		if err := c.handler(ctx, event); err != nil {
			c.logger.Error("handler error", zap.Error(err))
			// Don't commit — message will be re-delivered
			continue
		}

		_ = c.reader.CommitMessages(ctx, msg)
	}
}

func (c *Consumer[T]) Close() {
	c.reader.Close()
}
