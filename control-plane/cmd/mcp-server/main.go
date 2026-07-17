package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/mcp"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/remediation"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage"
	ch "github.com/frag2win/TelemetryHealth/control-plane/internal/storage/clickhouse"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type sseSession struct {
	id       string
	writeCh  chan []byte
	closedCh chan struct{}
}

type SseHandler struct {
	mcpServer *mcp.Server
	logger    *zap.Logger
	mu        sync.RWMutex
	sessions  map[string]*sseSession
}

func NewSseHandler(mcpServer *mcp.Server, logger *zap.Logger) *SseHandler {
	return &SseHandler{
		mcpServer: mcpServer,
		logger:    logger,
		sessions:  make(map[string]*sseSession),
	}
}

func (h *SseHandler) HandleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	sessionID := uuid.New().String()
	session := &sseSession{
		id:       sessionID,
		writeCh:  make(chan []byte, 100),
		closedCh: make(chan struct{}),
	}

	h.mu.Lock()
	h.sessions[sessionID] = session
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.sessions, sessionID)
		h.mu.Unlock()
		close(session.closedCh)
		h.logger.Info("SSE connection closed", zap.String("session_id", sessionID))
	}()

	h.logger.Info("New SSE connection established", zap.String("session_id", sessionID))

	// Send endpoint event to client
	fmt.Fprintf(w, "event: endpoint\ndata: /message?session=%s\n\n", sessionID)
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-session.writeCh:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(msg))
			flusher.Flush()
		}
	}
}

func (h *SseHandler) HandleMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Error(w, "Missing session parameter", http.StatusBadRequest)
		return
	}

	h.mu.RLock()
	session, exists := h.sessions[sessionID]
	h.mu.RUnlock()

	if !exists {
		http.Error(w, "Session not found or expired", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	go func() {
		respBytes, err := h.mcpServer.HandleJSONRPCMessage(r.Context(), body)
		if err != nil {
			h.logger.Error("Failed to handle JSON-RPC message", zap.Error(err))
			return
		}
		if respBytes == nil {
			return // Notification, no response
		}

		select {
		case <-session.closedCh:
			h.logger.Warn("Session closed before message response could be sent", zap.String("session_id", sessionID))
		case session.writeCh <- respBytes:
		}
	}()

	w.WriteHeader(http.StatusAccepted)
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

	// If stdio mode is explicitly set or sse is explicitly disabled
	if *stdioFlag {
		logger.Info("Starting MCP server in stdio mode...")
		runStdioServer(mcpServer, logger)
		return
	}

	// Default to SSE mode
	logger.Info("Starting MCP server in HTTP/SSE mode...", zap.Int("port", *portFlag))
	runSseServer(mcpServer, *portFlag, logger)
}

func runStdioServer(mcpServer *mcp.Server, logger *zap.Logger) {
	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		respBytes, err := mcpServer.HandleJSONRPCMessage(ctx, line)
		if err != nil {
			logger.Error("failed to handle message", zap.Error(err))
			continue
		}
		if respBytes != nil {
			fmt.Printf("%s\n", string(respBytes))
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Error("stdin read error", zap.Error(err))
	}
}

func runSseServer(mcpServer *mcp.Server, port int, logger *zap.Logger) {
	handler := NewSseHandler(mcpServer, logger)
	mux := http.NewServeMux()

	mux.HandleFunc("/sse", handler.HandleSSE)
	mux.HandleFunc("/message", handler.HandleMessage)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("TelemetryHealth MCP Server (Model Context Protocol)"))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
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
		logger.Fatal("MCP SSE server failed", zap.Error(err))
	case <-sigChan:
		logger.Info("Shutdown signal received, stopping MCP SSE server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("MCP SSE server shutdown error", zap.Error(err))
		}
		logger.Info("MCP SSE server stopped cleanly")
	}
}
