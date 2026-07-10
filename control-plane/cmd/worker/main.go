package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/kafka"
	ch "github.com/frag2win/TelemetryHealth/control-plane/internal/storage/clickhouse"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer logger.Sync()

	// Initialize OTel Self-Instrumentation SDK (PRD §10, Improvement #16)
	otelShutdown, err := telemetry.InitOTelSDK(context.Background(), "stream-worker")
	if err != nil {
		logger.Warn("Failed to initialize OTel self-instrumentation SDK", zap.Error(err))
	} else {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = otelShutdown(ctx)
		}()
	}

	brokers := []string{"localhost:9092"}

	// --- Ensure Kafka topics exist ---
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := kafka.EnsureTopics(ctx, brokers[0], logger); err != nil {
		logger.Warn("topic bootstrap failed (may already exist)", zap.Error(err))
	}

	// --- Connect to ClickHouse ---
	chCtx, chCancel := context.WithTimeout(context.Background(), 10*time.Second)
	chClient, err := ch.NewClient(
		chCtx,
		[]string{"localhost:9000"},
		"telemetry_health", "telemetry", "",
		logger,
	)
	chCancel()
	if err != nil {
		logger.Fatal("clickhouse connect failed", zap.Error(err))
	}
	defer chClient.Close()

	// --- Start consumer workers ---
	workers := kafka.NewWorkerSet(brokers, chClient, logger)

	runCtx, runCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer runCancel()

	errChan := make(chan error, 1)

	// Start Prometheus metrics server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{
		Addr:    ":9091",
		Handler: metricsMux,
	}
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("metrics server failed: %w", err)
		}
	}()

	logger.Info("Stream worker started — consuming from Kafka, writing to ClickHouse, metrics on :9091")

	// Start workers in a goroutine so we can wait for shutdown signal
	workerDone := make(chan struct{})
	go func() {
		workers.Run(runCtx)
		close(workerDone)
	}()

	select {
	case err := <-errChan:
		logger.Error("Metrics server startup failed", zap.Error(err))
	case <-runCtx.Done():
		logger.Info("Shutdown signal received, stopping workers...")
		<-workerDone

		metricsCtx, metricsCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer metricsCancel()
		if err := metricsServer.Shutdown(metricsCtx); err != nil {
			logger.Error("Metrics server shutdown failed", zap.Error(err))
		}
		logger.Info("Stream worker stopped cleanly")
	}
}
