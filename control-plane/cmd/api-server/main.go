package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/api/rest"
	ch "github.com/frag2win/TelemetryHealth/control-plane/internal/storage/clickhouse"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer logger.Sync()

	// Attempt ClickHouse connection (optional — graceful fallback to mock if unavailable)
	var healthRepo *ch.HealthRepository
	client, err := ch.NewClient(
		[]string{"localhost:9000"},
		"telemetry_health", "default", "",
		logger,
	)
	if err != nil {
		logger.Warn("ClickHouse unavailable, using mock data", zap.Error(err))
	} else {
		defer client.Close()
		healthRepo = ch.NewHealthRepository(client.Conn(), logger)
		logger.Info("ClickHouse connected — using real data")
	}

	server := rest.NewServer(logger, healthRepo)

	go func() {
		if err := server.Start(":8080"); err != nil {
			logger.Fatal("API server failed", zap.Error(err))
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	logger.Info("API server stopped cleanly")
}
