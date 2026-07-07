package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/kafka"
	ch "github.com/frag2win/TelemetryHealth/control-plane/internal/storage/clickhouse"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer logger.Sync()

	brokers := []string{"localhost:9092"}

	// --- Ensure Kafka topics exist ---
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := kafka.EnsureTopics(ctx, brokers[0], logger); err != nil {
		logger.Warn("topic bootstrap failed (may already exist)", zap.Error(err))
	}
	cancel()

	// --- Connect to ClickHouse ---
	chClient, err := ch.NewClient(
		[]string{"localhost:9000"},
		"telemetry_health", "telemetry", "",
		logger,
	)
	if err != nil {
		log.Fatalf("clickhouse connect: %v", err)
	}
	defer chClient.Close()

	// --- Start consumer workers ---
	workers := kafka.NewWorkerSet(brokers, chClient, logger)

	runCtx, runCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer runCancel()

	logger.Info("Stream worker started — consuming from Kafka, writing to ClickHouse")
	workers.Run(runCtx)
	logger.Info("Stream worker stopped cleanly")
}
