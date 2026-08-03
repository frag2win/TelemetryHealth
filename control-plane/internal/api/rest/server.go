package rest

// @title TelemetryHealth API
// @version 1.0
// @description REST API for TelemetryHealth control plane
// @host localhost:8080
// @BasePath /api/v1

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/engine"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/remediation"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage/signoz"

	_ "github.com/frag2win/TelemetryHealth/control-plane/docs" // imported for swagger
)

// Server is the REST API server for the control plane dashboard.
type Server struct {
	logger       *zap.Logger
	healthRepo   storage.HealthRepository
	validator    *remediation.Validator
	generator    *remediation.Generator
	graphEngine  *engine.Engine
	httpServer   *http.Server
	signozClient *signoz.QueryClient
}

// NewServer creates a new instance of the REST API server.
func NewServer(logger *zap.Logger, healthRepo storage.HealthRepository, replayRepo engine.ReplayRepository) *Server {
	return &Server{
		logger:       logger,
		healthRepo:   healthRepo,
		validator:    remediation.NewValidator(logger),
		generator:    remediation.NewGenerator(logger),
		graphEngine:  engine.NewEngine(replayRepo),
		signozClient: signoz.NewQueryClient(logger),
	}
}

func (s *Server) Start(addr string) error {
	r := s.routes()

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	s.logger.Info("Starting API Server", zap.String("addr", addr))
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	s.logger.Info("Shutting down API Server")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) livezHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}


