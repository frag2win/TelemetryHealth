package simulator

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Simulator struct {
	logger   *zap.Logger
	endpoint string
}

func NewSimulator(logger *zap.Logger, endpoint string) *Simulator {
	return &Simulator{
		logger:   logger,
		endpoint: endpoint,
	}
}

func (s *Simulator) getClient(ctx context.Context) (ptraceotlp.GRPCClient, *grpc.ClientConn, error) {
	conn, err := grpc.DialContext(ctx, s.endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to ingest-gateway: %w", err)
	}
	return ptraceotlp.NewGRPCClient(conn), conn, nil
}

func (s *Simulator) InjectHighCardinality(ctx context.Context, tenantID string) error {
	client, conn, err := s.getClient(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	s.logger.Info("Injecting high cardinality telemetry burst", zap.Int("count", 1000))

	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("tenant_id", tenantID)
	rs.Resource().Attributes().PutStr("service.name", "checkout-service")

	ss := rs.ScopeSpans().AppendEmpty()

	// Generate 1000 spans with completely unique attributes
	for i := 0; i < 1000; i++ {
		span := ss.Spans().AppendEmpty()
		traceID := pcommon.TraceID(uuid.New())
		spanID := pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, byte(i % 256)})

		span.SetTraceID(traceID)
		span.SetSpanID(spanID)
		span.SetName("ProcessOrder")
		span.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		span.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(10 * time.Millisecond)))

		// THIS IS THE ANOMALY: Highly unique values for a single attribute key
		span.Attributes().PutStr("db.statement", fmt.Sprintf("SELECT * FROM orders WHERE id = '%s'", uuid.New().String()))
		span.Attributes().PutStr("user_id", uuid.New().String())
	}

	req := ptraceotlp.NewExportRequestFromTraces(traces)
	outCtx := metadata.AppendToOutgoingContext(ctx, "x-tenant-id", tenantID)
	_, err = client.Export(outCtx, req)
	return err
}

func (s *Simulator) InjectDroppedSpans(ctx context.Context, tenantID string) error {
	client, conn, err := s.getClient(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	s.logger.Info("Injecting dropped/orphaned spans")

	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("tenant_id", tenantID)
	rs.Resource().Attributes().PutStr("service.name", "payment-service")

	ss := rs.ScopeSpans().AppendEmpty()

	traceID := pcommon.TraceID(uuid.New())
	parentIDUUID := uuid.New()
	var parentIDBytes [8]byte
	copy(parentIDBytes[:], parentIDUUID[:8])
	parentSpanID := pcommon.SpanID(parentIDBytes) // This span will NOT be included in the export!

	// Generate 5 child spans pointing to a parent that doesn't exist
	for i := 0; i < 5; i++ {
		span := ss.Spans().AppendEmpty()
		span.SetTraceID(traceID)
		childIDUUID := uuid.New()
		var childIDBytes [8]byte
		copy(childIDBytes[:], childIDUUID[:8])
		span.SetSpanID(pcommon.SpanID(childIDBytes))
		span.SetParentSpanID(parentSpanID) // Orphaned!
		span.SetName("ChargeCreditCard")
		span.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		span.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(50 * time.Millisecond)))
	}

	req := ptraceotlp.NewExportRequestFromTraces(traces)
	outCtx := metadata.AppendToOutgoingContext(ctx, "x-tenant-id", tenantID)
	_, err = client.Export(outCtx, req)
	return err
}
