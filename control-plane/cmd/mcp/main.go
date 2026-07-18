package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/mcp"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/remediation"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage"
	ch "github.com/frag2win/TelemetryHealth/control-plane/internal/storage/clickhouse"
	"go.uber.org/zap"
)

func main() {
	stdioFlag := flag.Bool("stdio", false, "Start MCP server in stdio mode")
	_ = flag.Bool("sse", true, "Start MCP server in SSE mode (default)")
	portFlag := flag.Int("port", 8081, "Port to run the SSE server on")
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer logger.Sync()

	// Initialize ClickHouse repository (fallback to mock if unavailable)
	var healthRepo storage.HealthRepository
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	chHost := os.Getenv("CH_HOST")
	if chHost == "" {
		chHost = "127.0.0.1"
	}
	chPort := os.Getenv("CH_PORT")
	if chPort == "" {
		chPort = "9000"
	}
	chAddr := chHost + ":" + chPort

	client, err := ch.NewClient(
		ctx,
		[]string{chAddr},
		"telemetry_health", "default", "",
		logger,
	)
	cancel()
	if err != nil {
		logger.Warn("ClickHouse unavailable — MCP queries will fallback or fail", zap.Error(err))
	} else {
		defer client.Close()
		healthRepo = ch.NewHealthRepository(client.Conn(), logger)
		logger.Info("ClickHouse connected successfully for MCP server")
	}

	generator := remediation.NewGenerator(logger)
	validator := remediation.NewValidator(logger)
	toolset := mcp.NewToolset(healthRepo, generator, validator, logger)
	mcpServer := mcp.NewServer(toolset)

	// If stdio mode is explicitly set
	if *stdioFlag {
		logger.Info("Starting MCP server in stdio mode...")
		if err := mcp.ServeStdio(context.Background(), os.Stdin, os.Stdout, mcpServer, logger); err != nil {
			logger.Error("MCP stdio server error", zap.Error(err))
		}
		return
	}

	// Default to HTTP/SSE mode
	logger.Info("Starting MCP server in HTTP/SSE mode...", zap.Int("port", *portFlag))
	handler := mcp.NewHTTPHandler(mcpServer, logger)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *portFlag),
		Handler: mux,
	}

	errChan := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logger.Fatal("MCP HTTP/SSE server failed", zap.Error(err))
	case <-sigChan:
		logger.Info("Shutdown signal received, stopping MCP HTTP/SSE server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("MCP HTTP/SSE server shutdown error", zap.Error(err))
		}
		logger.Info("MCP HTTP/SSE server stopped cleanly")
	}
}
