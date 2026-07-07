package ingest

import (
	"context"
	"net"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/authz"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/kafka"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"go.uber.org/zap"
)

const defaultCollectorID = "ingest-gateway"

// Server represents the Ingest Gateway gRPC server.
type Server struct {
	grpcServer *grpc.Server
	logger     *zap.Logger
}

type receiver struct {
	ptraceotlp.UnimplementedGRPCServer
	producer *kafka.Producer
	logger   *zap.Logger
}

func (r *receiver) Export(ctx context.Context, req ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	spans := req.Traces().ResourceSpans()
	for i := 0; i < spans.Len(); i++ {
		rs := spans.At(i)
		tenantID := rs.Resource().Attributes().AsRaw()["tenant_id"]
		tenantStr, _ := tenantID.(string)
		if tenantStr == "" {
			tenantStr = "unknown"
		}

		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			for k := 0; k < rs.ScopeSpans().At(j).Spans().Len(); k++ {
				span := rs.ScopeSpans().At(j).Spans().At(k)

				// Publish orphan candidate — cross-collector correlation happens in the worker
				if err := r.producer.PublishOrphan(ctx, kafka.OrphanEvent{
					TenantID:     tenantStr,
					TraceID:      span.TraceID().String(),
					SpanID:       span.SpanID().String(),
					ParentSpanID: span.ParentSpanID().String(),
					CollectorID:  defaultCollectorID,
					DetectedAt:   time.Now(),
				}); err != nil {
					r.logger.Error("failed to publish orphan event", zap.Error(err))
					return ptraceotlp.NewExportResponse(), status.Errorf(codes.Unavailable, "failed to publish telemetry: %v", err)
				}
			}
		}
	}

	r.logger.Debug("Traces exported to Kafka", zap.Int("resource_spans", spans.Len()))
	return ptraceotlp.NewExportResponse(), nil
}

type metricsReceiver struct {
	pmetricotlp.UnimplementedGRPCServer
	producer *kafka.Producer
	logger   *zap.Logger
}

func (r *metricsReceiver) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	rms := req.Metrics().ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		tenantID, _ := rm.Resource().Attributes().AsRaw()["tenant_id"].(string)
		service, _ := rm.Resource().Attributes().AsRaw()["service.name"].(string)

		// Publish coverage heartbeat per service
		if err := r.producer.PublishCoverage(ctx, kafka.CoverageEvent{
			TenantID:   tenantID,
			Service:    service,
			LastSeenAt: time.Now(),
		}); err != nil {
			r.logger.Error("failed to publish coverage event", zap.Error(err))
			return pmetricotlp.NewExportResponse(), status.Errorf(codes.Unavailable, "failed to publish telemetry: %v", err)
		}
	}

	r.logger.Debug("Metrics exported to Kafka", zap.Int("resource_metrics", rms.Len()))
	return pmetricotlp.NewExportResponse(), nil
}

type logsReceiver struct {
	plogotlp.UnimplementedGRPCServer
	logger *zap.Logger
}

func (r *logsReceiver) Export(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	r.logger.Debug("Logs received", zap.Int("resource_logs", req.Logs().ResourceLogs().Len()))
	return plogotlp.NewExportResponse(), nil
}

// NewServer creates a new Ingest Gateway server with tenant verification and Kafka producer.
func NewServer(logger *zap.Logger, producer *kafka.Producer) *Server {
	if logger == nil {
		logger = zap.NewNop()
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(authz.TenantAuthInterceptor()),
	}
	grpcServer := grpc.NewServer(opts...)

	ptraceotlp.RegisterGRPCServer(grpcServer, &receiver{producer: producer, logger: logger})
	pmetricotlp.RegisterGRPCServer(grpcServer, &metricsReceiver{producer: producer, logger: logger})
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
