package mcp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ServeStdio starts an MCP server loop using stdio transport (line-delimited JSON-RPC 2.0).
// It reads from r (e.g. os.Stdin) and writes responses to w (e.g. os.Stdout).
func ServeStdio(ctx context.Context, r io.Reader, w io.Writer, server *Server, logger *zap.Logger) error {
	scanner := bufio.NewScanner(r)
	const maxCapacity = 10 * 1024 * 1024 // 10MB capacity to handle large prompts/tool args
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		respBytes, err := server.HandleJSONRPCMessage(ctx, line)
		if err != nil {
			if logger != nil {
				logger.Error("failed to handle stdio message", zap.Error(err))
			}
			continue
		}

		if respBytes != nil {
			if _, err := w.Write(respBytes); err != nil {
				if logger != nil {
					logger.Error("failed to write stdio response", zap.Error(err))
				}
				return err
			}
			if _, err := w.Write([]byte("\n")); err != nil {
				if logger != nil {
					logger.Error("failed to write stdio newline", zap.Error(err))
				}
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		if logger != nil {
			logger.Error("stdio scan error", zap.Error(err))
		}
		return err
	}
	return nil
}

type sseSession struct {
	id       string
	writeCh  chan []byte
	closedCh chan struct{}
}

// HTTPHandler handles HTTP and SSE transports for the MCP server.
type HTTPHandler struct {
	server   *Server
	logger   *zap.Logger
	mu       sync.RWMutex
	sessions map[string]*sseSession
}

func NewHTTPHandler(server *Server, logger *zap.Logger) *HTTPHandler {
	return &HTTPHandler{
		server:   server,
		logger:   logger,
		sessions: make(map[string]*sseSession),
	}
}

// RegisterRoutes registers all MCP transport routes onto the provided ServeMux.
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/sse", h.HandleSSE)
	mux.HandleFunc("/message", h.HandleMessage)
	mux.HandleFunc("/jsonrpc", h.HandleMessage)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.HandleMessage(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("TelemetryHealth MCP Server (Model Context Protocol)"))
	})
}

func (h *HTTPHandler) HandleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
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
		if h.logger != nil {
			h.logger.Info("SSE connection closed", zap.String("session_id", sessionID))
		}
	}()

	if h.logger != nil {
		h.logger.Info("New SSE connection established", zap.String("session_id", sessionID))
	}

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

func (h *HTTPHandler) HandleMessage(w http.ResponseWriter, r *http.Request) {
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	sessionID := r.URL.Query().Get("session")
	if sessionID != "" {
		h.mu.RLock()
		session, exists := h.sessions[sessionID]
		h.mu.RUnlock()

		if !exists {
			http.Error(w, "Session not found or expired", http.StatusNotFound)
			return
		}

		go func() {
			respBytes, err := h.server.HandleJSONRPCMessage(r.Context(), body)
			if err != nil {
				if h.logger != nil {
					h.logger.Error("Failed to handle JSON-RPC message over SSE", zap.Error(err))
				}
				return
			}
			if respBytes == nil {
				return
			}

			select {
			case <-session.closedCh:
				if h.logger != nil {
					h.logger.Warn("Session closed before message response could be sent", zap.String("session_id", sessionID))
				}
			case session.writeCh <- respBytes:
			}
		}()

		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Direct HTTP JSON-RPC request without SSE session
	respBytes, err := h.server.HandleJSONRPCMessage(r.Context(), body)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to handle direct HTTP JSON-RPC message", zap.Error(err))
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if respBytes == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}
