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
	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
	"github.com/frag2win/TelemetryHealth/control-plane/pkg/models"
	"go.uber.org/zap"
)

// MockHealthRepository implements the full footprint of storage.HealthRepository.
type MockHealthRepository struct{}

func (m *MockHealthRepository) QueryHealthMetrics(ctx context.Context, tenantID string) (*storage.HealthMetrics, error) {
	return &storage.HealthMetrics{
		TenantID:            tenantID,
		CardinalityMax:      0,
		OrphanCount:         0,
		PreviousOrphanCount: 0,
		ActiveServices:      0,
		CompositeScore:      100.0,
		RemediationIssue:    "",
		Window:              time.Now(),
	}, nil
}

func (m *MockHealthRepository) QueryAgentTraces(ctx context.Context) ([]storage.AgentTrace, error) {
	return []storage.AgentTrace{}, nil
}

func (m *MockHealthRepository) GetTenantWeights(ctx context.Context, tenantID string) (telemetry.TenantWeights, error) {
	return telemetry.TenantWeights{
		CardinalityWeight: 0.20,
		OrphanWeight:      0.30,
		CoverageWeight:    0.50,
	}, nil
}

func (m *MockHealthRepository) SaveTenantConfig(ctx context.Context, tenantID string, weights telemetry.TenantWeights) error {
	return nil
}

func (m *MockHealthRepository) LogRemediationEvent(ctx context.Context, tenantID string, issueType string, yamlConfig string, validated, applied bool, actorID, actorRole, sourceIP, action, resourceID string) error {
	return nil
}

func (m *MockHealthRepository) QuerySpansByTraceID(ctx context.Context, traceID string) ([]models.SpanData, error) {
	return []models.SpanData{}, nil
}

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
	client, err := ch.NewClient(
		ctx,
		[]string{"127.0.0.1:9000"},
		"telemetry_health", "default", "",
		logger,
	)
	cancel()
	if err != nil {
		logger.Warn("ClickHouse unavailable — falling back to safe in-memory repository store to block runtime panics", zap.Error(err))
		healthRepo = &MockHealthRepository{}
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
