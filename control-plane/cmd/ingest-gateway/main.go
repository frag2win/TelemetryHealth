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

	"github.com/frag2win/TelemetryHealth/control-plane/internal/ingest"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/kafka"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	if os.Getenv("ENV") != "production" && os.Getenv("INSECURE_DEV_MODE") == "" {
		os.Setenv("INSECURE_DEV_MODE", "true")
	}
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Initialize OTel Self-Instrumentation SDK (PRD §10, Improvement #16)
	otelShutdown, err := telemetry.InitOTelSDK(context.Background(), "ingest-gateway")
	if err != nil {
		logger.Warn("Failed to initialize OTel self-instrumentation SDK", zap.Error(err))
	} else {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = otelShutdown(ctx)
		}()
	}

	brokers := []string{"127.0.0.1:9092"}

	// Bootstrap Kafka topics
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := kafka.EnsureTopics(ctx, brokers[0], logger); err != nil {
		logger.Warn("topic bootstrap warning (may already exist)", zap.Error(err))
	}

	// Create Kafka producer
	producer := kafka.NewProducer(brokers, logger)
	defer func() {
		if err := producer.Close(); err != nil {
			logger.Error("failed to close producer", zap.Error(err))
		}
	}()

	server := ingest.NewServer(logger, producer)

	errChan := make(chan error, 2)

	// Start server in background
	go func() {
		addr := ":4317" // Default OTLP gRPC port
		if err := server.Start(addr); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// Start Prometheus metrics server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{
		Addr:    ":9094",
		Handler: metricsMux,
	}
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("metrics server error: %w", err)
		}
	}()

	logger.Info("Ingest Gateway started on :4317, metrics on :9090")

	// Wait for termination signal or startup error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logger.Error("Server startup failed", zap.Error(err))
	case <-sigChan:
		logger.Info("Shutdown signal received, stopping Ingest Gateway...")
		server.Stop()

		metricsCtx, metricsCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer metricsCancel()
		if err := metricsServer.Shutdown(metricsCtx); err != nil {
			logger.Error("Metrics server shutdown failed", zap.Error(err))
		}
		logger.Info("Server stopped cleanly")
	}
}
