package ingest

import (
	"context"
	"net"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/authz"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"google.golang.org/grpc"
	"go.uber.org/zap"
)

// Server represents the Ingest Gateway gRPC server.
type Server struct {
	grpcServer *grpc.Server
	logger     *zap.Logger
}

type receiver struct {
	ptraceotlp.UnimplementedGRPCServer
	logger *zap.Logger
}

func (r *receiver) Export(ctx context.Context, req ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	// TODO: Route traces to Kafka/Stream processing
	r.logger.Debug("Received traces export request")
	return ptraceotlp.NewExportResponse(), nil
}

type metricsReceiver struct {
	pmetricotlp.UnimplementedGRPCServer
	logger *zap.Logger
}

func (r *metricsReceiver) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	r.logger.Debug("Received metrics export request")
	return pmetricotlp.NewExportResponse(), nil
}

type logsReceiver struct {
	plogotlp.UnimplementedGRPCServer
	logger *zap.Logger
}

func (r *logsReceiver) Export(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	r.logger.Debug("Received logs export request")
	return plogotlp.NewExportResponse(), nil
}

// NewServer creates a new Ingest Gateway server with tenant verification.
func NewServer(logger *zap.Logger) *Server {
	if logger == nil {
		logger = zap.NewNop()
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(authz.TenantAuthInterceptor()),
	}
	grpcServer := grpc.NewServer(opts...)

	ptraceotlp.RegisterGRPCServer(grpcServer, &receiver{logger: logger})
	pmetricotlp.RegisterGRPCServer(grpcServer, &metricsReceiver{logger: logger})
	plogotlp.RegisterGRPCServer(grpcServer, &logsReceiver{logger: logger})

	return &Server{
		grpcServer: grpcServer,
		logger:     logger,
	}
}

// Start begins listening on the specified address.
func (s *Server) Start(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.logger.Info("Starting Ingest Gateway", zap.String("address", addr))
	return s.grpcServer.Serve(listener)
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() {
	s.logger.Info("Stopping Ingest Gateway")
	s.grpcServer.GracefulStop()
}
