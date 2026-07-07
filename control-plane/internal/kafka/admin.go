package kafka

import (
	"context"
	"net"
	"strconv"

	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// EnsureTopics creates the required Kafka topics if they don't exist.
func EnsureTopics(ctx context.Context, broker string, logger *zap.Logger) error {
	topics := []string{TopicCardinality, TopicOrphan, TopicCoverage}

	conn, err := kafkago.DialContext(ctx, "tcp", broker)
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}

	controllerConn, err := kafkago.DialContext(ctx, "tcp", net.JoinHostPort(controller.Host, itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer controllerConn.Close()

	topicConfigs := make([]kafkago.TopicConfig, len(topics))
	for i, t := range topics {
		topicConfigs[i] = kafkago.TopicConfig{
			Topic:             t,
			NumPartitions:     1,
			ReplicationFactor: 1,
		}
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		// Ignore "topic already exists" errors
		logger.Debug("CreateTopics result (may include already-exists)", zap.Error(err))
	}

	logger.Info("Kafka topics ensured", zap.Strings("topics", topics))
	return nil
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
