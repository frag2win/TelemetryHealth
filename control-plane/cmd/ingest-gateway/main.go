package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/ingest"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/kafka"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	brokers := []string{"localhost:9092"}

	// Bootstrap Kafka topics
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := kafka.EnsureTopics(ctx, brokers[0], logger); err != nil {
		logger.Warn("topic bootstrap warning (may already exist)", zap.Error(err))
	}
	cancel()

	// Create Kafka producer
	producer := kafka.NewProducer(brokers, logger)
	defer producer.Close()

	server := ingest.NewServer(logger, producer)

	// Start server in background
	go func() {
		addr := ":4317" // Default OTLP gRPC port
		if err := server.Start(addr); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	logger.Info("Ingest Gateway started on :4317")

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	server.Stop()
	logger.Info("Server stopped cleanly")
}
