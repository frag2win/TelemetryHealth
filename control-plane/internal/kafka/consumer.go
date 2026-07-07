package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

const (
	defaultBatchSize = 1000
	defaultBatchTime = 3 * time.Second
)

// Handler processes a batch of decoded Kafka messages.
type Handler[T any] func(ctx context.Context, events []T) error

// Consumer reads from a Kafka topic and dispatches batches to a typed handler.
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

// Run starts consuming messages, buffering them, and calling the handler.
// It blocks until ctx is cancelled.
func (c *Consumer[T]) Run(ctx context.Context) error {
	c.logger.Info("Kafka consumer started", zap.String("topic", c.reader.Config().Topic))

	batch := make([]T, 0, defaultBatchSize)
	msgs := make([]kafkago.Message, 0, defaultBatchSize)
	timer := time.NewTimer(defaultBatchTime)
	defer timer.Stop()

	for {
		// Read a message with a short timeout so we can also check the batch timer and context
		fetchCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		msg, err := c.reader.FetchMessage(fetchCtx)
		cancel()

		if err == nil {
			var event T
			if decodeErr := json.Unmarshal(msg.Value, &event); decodeErr != nil {
				c.logger.Error("decode failed, dropping message", zap.Error(decodeErr))
				_ = c.reader.CommitMessages(ctx, msg)
			} else {
				batch = append(batch, event)
				msgs = append(msgs, msg)
			}
		} else if ctx.Err() != nil {
			// graceful shutdown — flush what we have
			c.flush(context.Background(), batch, msgs)
			return nil
		}

		// Flush if we hit size limit or timeout
		select {
		case <-timer.C:
			if len(batch) > 0 {
				c.flush(ctx, batch, msgs)
				batch = batch[:0]
				msgs = msgs[:0]
			}
			timer.Reset(defaultBatchTime)
		default:
			if len(batch) >= defaultBatchSize {
				c.flush(ctx, batch, msgs)
				batch = batch[:0]
				msgs = msgs[:0]
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(defaultBatchTime)
			}
		}
	}
}

// flush sends the batch to the handler, retrying with exponential backoff on failure.
func (c *Consumer[T]) flush(ctx context.Context, batch []T, msgs []kafkago.Message) {
	if len(batch) == 0 {
		return
	}

	backoff := 100 * time.Millisecond
	maxBackoff := 10 * time.Second

	for {
		if err := c.handler(ctx, batch); err != nil {
			c.logger.Error("handler error, retrying batch", 
				zap.Int("size", len(batch)),
				zap.Duration("backoff", backoff),
				zap.Error(err),
			)
			
			// If context is cancelled during backoff, we drop the uncommitted batch
			// to avoid blocking shutdown. It will be re-delivered on restart.
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}

			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Success! Commit all messages in the batch.
		telemetry.KafkaMessagesProcessedTotal.WithLabelValues(c.reader.Config().Topic).Add(float64(len(batch)))
		if err := c.reader.CommitMessages(ctx, msgs...); err != nil {
			c.logger.Error("failed to commit messages", zap.Error(err))
		}
		return // done
	}
}

func (c *Consumer[T]) Close() {
	c.reader.Close()
}
