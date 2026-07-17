package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/authz"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/alerting"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/api/rest"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/engine"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage"
	ch "github.com/frag2win/TelemetryHealth/control-plane/internal/storage/clickhouse"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage/mock"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
	"go.uber.org/zap"
)

func main() {
	if os.Getenv("ENV") != "production" && os.Getenv("INSECURE_DEV_MODE") == "" {
		os.Setenv("INSECURE_DEV_MODE", "true")
	}
	// PRD §10 Security, Improvement #2.1: panic immediately if INSECURE_DEV_MODE is set in production.
	authz.ValidateStartupConfig()

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer logger.Sync()

	// Initialize OTel Self-Instrumentation SDK (PRD §10, Improvement #16)
	otelShutdown, err := telemetry.InitOTelSDK(context.Background(), "api-server")
	if err != nil {
		logger.Warn("Failed to initialize OTel self-instrumentation SDK", zap.Error(err))
	} else {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = otelShutdown(ctx)
		}()
	}

	// Attempt ClickHouse connection (optional — graceful fallback to mock if unavailable)
	var healthRepo storage.HealthRepository
	var replayRepo engine.ReplayRepository
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	client, err := ch.NewClient(
		ctx,
		[]string{"127.0.0.1:9000"},
		"telemetry_health", "default", "",
		logger,
	)
	cancel()
	if err != nil {
		logger.Warn("ClickHouse unavailable, using mock data", zap.Error(err))
		mockRepo := mock.NewRepository()
		healthRepo = mockRepo
		replayRepo = mockRepo
	} else {
		defer client.Close()
		healthRepo = ch.NewHealthRepository(client.Conn(), logger)
		replayRepo = ch.NewReplayRepository(client.Conn(), logger)
		logger.Info("ClickHouse connected — using real data")
	}

	server := rest.NewServer(logger, healthRepo, replayRepo)

	// Initialize and start Telemetry Poller (Alertmanager Bridge)
	bridge := alerting.NewSigNozBridge(logger)
	poller := alerting.NewTelemetryPoller(logger, healthRepo, bridge, 10*time.Second, 80.0, "00000000-0000-0000-0000-000000000001")
	poller.Start(context.Background())

	// Use channel to handle errors from API server start
	errChan := make(chan error, 1)
	go func() {
		if err := server.Start(":8080"); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logger.Error("API server failed on startup", zap.Error(err))
	case <-sigChan:
		logger.Info("Shutdown signal received, shutting down API server...")
		ctx, cancelCtx := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelCtx()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("API server shutdown failed", zap.Error(err))
		}
		logger.Info("API server stopped cleanly")
	}
}
