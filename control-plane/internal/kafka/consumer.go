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
	reader    *kafkago.Reader
	dlqWriter *kafkago.Writer
	handler   Handler[T]
	logger    *zap.Logger
}

func NewConsumer[T any](brokers []string, topic, groupID string, handler Handler[T], logger *zap.Logger) *Consumer[T] {
	// Validate groupID is non-empty (Finding 8.2)
	if groupID == "" {
		groupID = topic + "-group-default"
		logger.Warn("NewConsumer called with empty groupID, falling back to default", zap.String("groupID", groupID))
	}

	return &Consumer[T]{
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       1e6, // 1 MB
			CommitInterval: 0,   // explicit commit after processing
		}),
		dlqWriter: &kafkago.Writer{
			Addr:     kafkago.TCP(brokers...),
			Topic:    topic + ".dlq",
			Balancer: &kafkago.LeastBytes{},
		},
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
					select {
					case <-timer.C:
					default:
					}
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
	retries := 0
	maxRetries := 5

	for {
		if err := c.handler(ctx, batch); err != nil {
			retries++
			if retries > maxRetries {
				c.logger.Error("handler failed persistently, writing batch to DLQ and dropping to fail open",
					zap.Int("size", len(batch)),
					zap.Error(err),
				)
				c.writeToDLQ(ctx, msgs)
				return
			}

			c.logger.Error("handler error, retrying batch", 
				zap.Int("size", len(batch)),
				zap.Int("retry", retries),
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

func (c *Consumer[T]) writeToDLQ(ctx context.Context, msgs []kafkago.Message) {
	dlqMsgs := make([]kafkago.Message, len(msgs))
	for i, m := range msgs {
		dlqMsgs[i] = kafkago.Message{
			Key:   m.Key,
			Value: m.Value,
			Time:  time.Now(),
		}
	}
	writeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := c.dlqWriter.WriteMessages(writeCtx, dlqMsgs...); err != nil {
		c.logger.Error("failed to write messages to DLQ", zap.Error(err))
	} else {
		c.logger.Info("successfully wrote failed batch to DLQ", zap.Int("count", len(msgs)), zap.String("dlqTopic", c.dlqWriter.Topic))
		if err := c.reader.CommitMessages(ctx, msgs...); err != nil {
			c.logger.Error("failed to commit messages after sending to DLQ", zap.Error(err))
		}
	}
}

func (c *Consumer[T]) Close() {
	_ = c.reader.Close()
	_ = c.dlqWriter.Close()
}
