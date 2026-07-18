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

func (s *Simulator) InjectAgenticWorkflow(ctx context.Context, tenantID string) error {
	client, conn, err := s.getClient(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	s.logger.Info("Injecting agentic workflow spans")

	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("tenant_id", tenantID)
	rs.Resource().Attributes().PutStr("service.name", "ai-agent-service")

	ss := rs.ScopeSpans().AppendEmpty()
	
	// 1. Planner Span
	traceID := pcommon.TraceID(uuid.New())
	
	plannerUUID := uuid.New()
	var plannerIDBytes [8]byte
	copy(plannerIDBytes[:], plannerUUID[:8])
	plannerID := pcommon.SpanID(plannerIDBytes)
	
	planner := ss.Spans().AppendEmpty()
	planner.SetTraceID(traceID)
	planner.SetSpanID(plannerID)
	planner.SetName("PlanExecution")
	planner.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	planner.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(100 * time.Millisecond)))
	planner.Attributes().PutStr("llm.role", "planner")
	
	// 2. Retriever Span
	retrieverUUID := uuid.New()
	var retrieverIDBytes [8]byte
	copy(retrieverIDBytes[:], retrieverUUID[:8])
	retrieverID := pcommon.SpanID(retrieverIDBytes)

	retriever := ss.Spans().AppendEmpty()
	retriever.SetTraceID(traceID)
	retriever.SetSpanID(retrieverID)
	retriever.SetParentSpanID(plannerID)
	retriever.SetName("FetchContext")
	retriever.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(100 * time.Millisecond)))
	retriever.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(200 * time.Millisecond)))
	retriever.Attributes().PutStr("vector.search", "true")

	// 3. Tool Span
	toolUUID := uuid.New()
	var toolIDBytes [8]byte
	copy(toolIDBytes[:], toolUUID[:8])
	toolID := pcommon.SpanID(toolIDBytes)

	tool := ss.Spans().AppendEmpty()
	tool.SetTraceID(traceID)
	tool.SetSpanID(toolID)
	tool.SetParentSpanID(plannerID)
	tool.SetName("ExecuteCommand")
	tool.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(200 * time.Millisecond)))
	tool.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(300 * time.Millisecond)))
	tool.Attributes().PutStr("tool.name", "bash")

	req := ptraceotlp.NewExportRequestFromTraces(traces)
	outCtx := metadata.AppendToOutgoingContext(ctx, "x-tenant-id", tenantID)
	_, err = client.Export(outCtx, req)
	return err
}

func (s *Simulator) InjectAgenticFailure(ctx context.Context, tenantID string) error {
	client, conn, err := s.getClient(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	s.logger.Info("Injecting failed/retry agentic workflow spans")

	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("tenant_id", tenantID)
	rs.Resource().Attributes().PutStr("service.name", "ai-agent-service")

	ss := rs.ScopeSpans().AppendEmpty()

	// 1. Planner/Workflow Span
	traceID := pcommon.TraceID(uuid.New())

	plannerUUID := uuid.New()
	var plannerIDBytes [8]byte
	copy(plannerIDBytes[:], plannerUUID[:8])
	plannerID := pcommon.SpanID(plannerIDBytes)

	planner := ss.Spans().AppendEmpty()
	planner.SetTraceID(traceID)
	planner.SetSpanID(plannerID)
	planner.SetName("PlanExecution")
	planner.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	planner.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(3000 * time.Millisecond)))
	planner.Attributes().PutStr("llm.role", "planner")
	planner.Attributes().PutStr("workflow.topic", "Observability best practices")
	planner.Status().SetCode(ptrace.StatusCodeError)
	planner.Status().SetMessage("Agent workflow execution failed")

	// 2. Failed Tool Span (web_search timeout)
	toolUUID1 := uuid.New()
	var toolIDBytes1 [8]byte
	copy(toolIDBytes1[:], toolUUID1[:8])
	toolID1 := pcommon.SpanID(toolIDBytes1)

	tool1 := ss.Spans().AppendEmpty()
	tool1.SetTraceID(traceID)
	tool1.SetSpanID(toolID1)
	tool1.SetParentSpanID(plannerID)
	tool1.SetName("agent.research")
	tool1.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(100 * time.Millisecond)))
	tool1.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(1100 * time.Millisecond)))
	tool1.Attributes().PutStr("llm.tool_name", "web_search")
	tool1.Attributes().PutStr("llm.tool_call.error", "TimeoutError: connection refused")
	tool1.Status().SetCode(ptrace.StatusCodeError)

	// 3. Retry Tool Span (web_search retry success)
	toolUUID2 := uuid.New()
	var toolIDBytes2 [8]byte
	copy(toolIDBytes2[:], toolUUID2[:8])
	toolID2 := pcommon.SpanID(toolIDBytes2)

	tool2 := ss.Spans().AppendEmpty()
	tool2.SetTraceID(traceID)
	tool2.SetSpanID(toolID2)
	tool2.SetParentSpanID(plannerID)
	tool2.SetName("agent.research")
	tool2.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(1200 * time.Millisecond)))
	tool2.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(2000 * time.Millisecond)))
	tool2.Attributes().PutStr("llm.tool_name", "web_search")
	tool2.Status().SetCode(ptrace.StatusCodeOk)

	// 4. Summarize LLM Span
	sumUUID := uuid.New()
	var sumIDBytes [8]byte
	copy(sumIDBytes[:], sumUUID[:8])
	sumID := pcommon.SpanID(sumIDBytes)

	sum := ss.Spans().AppendEmpty()
	sum.SetTraceID(traceID)
	sum.SetSpanID(sumID)
	sum.SetParentSpanID(plannerID)
	sum.SetName("agent.summarize")
	sum.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(2100 * time.Millisecond)))
	sum.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(3300 * time.Millisecond)))
	sum.Attributes().PutStr("llm.model", "claude-3-5-sonnet")
	sum.Attributes().PutStr("llm.token_usage", "8450")
	sum.Status().SetCode(ptrace.StatusCodeOk)

	req := ptraceotlp.NewExportRequestFromTraces(traces)
	outCtx := metadata.AppendToOutgoingContext(ctx, "x-tenant-id", tenantID)
	_, err = client.Export(outCtx, req)
	return err
}
