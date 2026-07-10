package ingest

// grpc_server.go — Ingest Gateway: receives OTLP spans/metrics/logs, verifies tenant identity,
// and publishes structured events to Kafka for stream processing.
//
// PRD §8.1: Cardinality signal extraction at ingest (Improvement #2.2).
// PRD §8.2: Structural span tuples extracted BEFORE sampling (orphan correlation).
// PRD §10:  Fail-open: gRPC handler errors are logged but do not block the OTLP export response.

import (
	"context"
	"net"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/authz"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/kafka"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"google.golang.org/grpc"
	"go.uber.org/zap"
)

const (
	defaultCollectorID = "ingest-gateway"
	// maxCardinalityKeysPerService caps the distinct attribute keys tracked per service
	// to prevent key-space explosion (PRD §8.1, Improvement #2.2).
	maxCardinalityKeysPerService = 100
)

// Server represents the Ingest Gateway gRPC server.
type Server struct {
	grpcServer *grpc.Server
	logger     *zap.Logger
}

// ── Traces receiver ───────────────────────────────────────────────────────────

type receiver struct {
	ptraceotlp.UnimplementedGRPCServer
	producer *kafka.Producer
	logger   *zap.Logger
}

// Export handles incoming OTLP trace exports.
// For each span it:
//  1. Publishes an orphan candidate (structural tuple: trace_id, span_id, parent_span_id).
//  2. Extracts attribute key/value pairs and publishes CardianlityEvents (PRD §8.1, Improvement #2.2).
func (r *receiver) Export(ctx context.Context, req ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	spans := req.Traces().ResourceSpans()
	for i := 0; i < spans.Len(); i++ {
		rs := spans.At(i)
		tenantStr := extractTenantID(rs.Resource())
		serviceName := extractServiceName(rs.Resource())

		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			for k := 0; k < rs.ScopeSpans().At(j).Spans().Len(); k++ {
				span := rs.ScopeSpans().At(j).Spans().At(k)

				telemetry.IngestedSpansTotal.WithLabelValues(tenantStr).Inc()

				// Publish orphan candidate — cross-collector correlation happens in the stream worker.
				if err := r.producer.PublishOrphan(ctx, kafka.OrphanEvent{
					TenantID:     tenantStr,
					TraceID:      span.TraceID().String(),
					SpanID:       span.SpanID().String(),
					ParentSpanID: span.ParentSpanID().String(),
					CollectorID:  defaultCollectorID,
					DetectedAt:   time.Now(),
				}); err != nil {
					// Fail-open: log warning but do NOT block the OTLP export response.
					r.logger.Warn("failed to publish orphan event, skipping", zap.Error(err))
				}

				// Extract cardinality signals from span attributes (PRD §8.1, Improvement #2.2).
				// Key-space explosion protection: track at most maxCardinalityKeysPerService distinct keys.
				r.publishCardinalityEvents(ctx, tenantStr, serviceName, span.Attributes())
			}
		}
	}

	r.logger.Debug("Traces exported to Kafka", zap.Int("resource_spans", spans.Len()))
	return ptraceotlp.NewExportResponse(), nil
}

// publishCardinalityEvents scans span attributes and publishes a CardinalityEvent per attribute key.
// Enforces the key-space cap (PRD §8.1) by stopping after maxCardinalityKeysPerService distinct keys.
func (r *receiver) publishCardinalityEvents(ctx context.Context, tenantID, service string, attrs pcommon.Map) {
	keyCount := 0
	attrs.Range(func(k string, v pcommon.Value) bool {
		if keyCount >= maxCardinalityKeysPerService {
			r.logger.Warn("Key-space cap reached at ingest, skipping remaining attributes",
				zap.String("tenant_id", tenantID),
				zap.String("service", service),
				zap.Int("cap", maxCardinalityKeysPerService),
			)
			return false // Stop iteration
		}

		// Publish one cardinality event per attribute key observed.
		// The unique_values field is 1 for each individual span — the stream worker
		// merges these with HLL sketches to get the true cross-collector estimate.
		event := kafka.CardinalityEvent{
			TenantID:     tenantID,
			Service:      service,
			AttributeKey: k,
			UniqueValues: 1,
			Timestamp:    time.Now(),
		}

		if err := r.producer.PublishCardinality(ctx, event); err != nil {
			// Fail-open: do not block on Kafka errors.
			r.logger.Warn("failed to publish cardinality event, skipping",
				zap.String("key", k),
				zap.Error(err),
			)
		}
		keyCount++
		return true
	})
}

// ── Metrics receiver ──────────────────────────────────────────────────────────

type metricsReceiver struct {
	pmetricotlp.UnimplementedGRPCServer
	producer *kafka.Producer
	logger   *zap.Logger
}

// Export handles incoming OTLP metrics exports.
// Publishes a CoverageEvent heartbeat per active service (PRD §8.3).
func (r *metricsReceiver) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	rms := req.Metrics().ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		tenantID := extractTenantID(rm.Resource())
		service := extractServiceName(rm.Resource())

		// Publish coverage heartbeat per service (PRD §8.3).
		if err := r.producer.PublishCoverage(ctx, kafka.CoverageEvent{
			TenantID:   tenantID,
			Service:    service,
			LastSeenAt: time.Now(),
		}); err != nil {
			r.logger.Warn("failed to publish coverage event, skipping", zap.Error(err))
		}
	}

	r.logger.Debug("Metrics exported to Kafka", zap.Int("resource_metrics", rms.Len()))
	return pmetricotlp.NewExportResponse(), nil
}

// ── Logs receiver ─────────────────────────────────────────────────────────────

type logsReceiver struct {
	plogotlp.UnimplementedGRPCServer
	producer *kafka.Producer
	logger   *zap.Logger
}

// Export handles incoming OTLP log exports.
// Currently publishes a coverage heartbeat per service (logs prove the service is alive).
// Full log-based coverage analysis is planned for M2.
func (r *logsReceiver) Export(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	rls := req.Logs().ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rl := rls.At(i)
		tenantID := extractTenantID(rl.Resource())
		service := extractServiceName(rl.Resource())

		// Publish coverage heartbeat — log emission proves the service is alive (PRD §8.3).
		if r.producer != nil && service != "" {
			if err := r.producer.PublishCoverage(ctx, kafka.CoverageEvent{
				TenantID:   tenantID,
				Service:    service,
				LastSeenAt: time.Now(),
			}); err != nil {
				r.logger.Warn("failed to publish logs-based coverage event, skipping", zap.Error(err))
			}
		}
	}

	r.logger.Debug("Logs received", zap.Int("resource_logs", rls.Len()))
	return plogotlp.NewExportResponse(), nil
}

// ── Helper functions ──────────────────────────────────────────────────────────

// extractTenantID reads the tenant_id resource attribute, defaulting to "unknown".
func extractTenantID(res pcommon.Resource) string {
	if v, ok := res.Attributes().Get("tenant_id"); ok {
		if s := v.Str(); s != "" {
			return s
		}
	}
	return "unknown"
}

// extractServiceName reads the service.name resource attribute, defaulting to "unknown".
func extractServiceName(res pcommon.Resource) string {
	if v, ok := res.Attributes().Get("service.name"); ok {
		if s := v.Str(); s != "" {
			return s
		}
	}
	return "unknown"
}

// ── Server construction ───────────────────────────────────────────────────────

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
	plogotlp.RegisterGRPCServer(grpcServer, &logsReceiver{producer: producer, logger: logger})

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
