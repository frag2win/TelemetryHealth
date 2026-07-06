package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/ingest"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	server := ingest.NewServer(logger)

	// Start server in background
	go func() {
		addr := ":4317" // Default OTLP gRPC port
		if err := server.Start(addr); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	server.Stop()
	logger.Info("Server stopped cleanly")
}
