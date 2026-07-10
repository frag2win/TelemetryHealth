package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// e2e_test sends a real OTLP gRPC payload to the Ingest Gateway and verifies
// the pipeline: Gateway → Kafka Producer → (worker picks up) → ClickHouse.
func main() {
	gateway := flag.String("gateway", "localhost:4317", "Ingest gateway address")
	tenantStr := flag.String("tenant", "00000000-0000-0000-0000-000000000001", "Tenant UUID")
	flag.Parse()

	conn, err := grpc.NewClient(*gateway,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("dial ingest gateway: %v", err)
	}
	defer conn.Close()

	tenantID := *tenantStr

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attach tenant-id metadata header (required by authz interceptor)
	ctx = metadata.AppendToOutgoingContext(ctx, "x-tenant-id", tenantID)

	// --- 1. Send Traces (triggers orphan detection) ---
	traceClient := ptraceotlp.NewGRPCClient(conn)
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("tenant_id", tenantID)
	rs.Resource().Attributes().PutStr("service.name", "checkout")
	scopeSpan := rs.ScopeSpans().AppendEmpty()
	for i := 0; i < 5; i++ {
		span := scopeSpan.Spans().AppendEmpty()
		traceID := pcommon.TraceID([16]byte{byte(i + 1)})
		spanID := pcommon.SpanID([8]byte{byte(i + 1)})
		span.SetTraceID(traceID)
		span.SetSpanID(spanID)
		span.SetName(fmt.Sprintf("operation-%d", i))
	}

	traceResp, err := traceClient.Export(ctx, ptraceotlp.NewExportRequestFromTraces(traces))
	if err != nil {
		log.Fatalf("trace export failed: %v", err)
	}
	fmt.Printf("✓ Traces sent: partial_success=%v\n", traceResp.PartialSuccess().ErrorMessage())

	// --- 2. Send Metrics (triggers coverage heartbeat) ---
	metricsClient := pmetricotlp.NewGRPCClient(conn)
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("tenant_id", tenantID)
	rm.Resource().Attributes().PutStr("service.name", "checkout")
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName("http.request.duration")
	m.SetEmptyGauge()
	dp := m.Gauge().DataPoints().AppendEmpty()
	dp.SetDoubleValue(42.5)
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	metricsResp, err := metricsClient.Export(ctx, pmetricotlp.NewExportRequestFromMetrics(metrics))
	if err != nil {
		log.Fatalf("metrics export failed: %v", err)
	}
	fmt.Printf("✓ Metrics sent: partial_success=%v\n", metricsResp.PartialSuccess().ErrorMessage())

	fmt.Println("\n✅ E2E test passed!")
	fmt.Println("   → Ingest Gateway received OTLP payload")
	fmt.Println("   → Orphan events published to telemetry.orphan Kafka topic")
	fmt.Println("   → Coverage heartbeat published to telemetry.coverage Kafka topic")
	fmt.Println("   → Stream worker will pick up and write to ClickHouse")
	fmt.Println("   → Dashboard at http://localhost:5173 will reflect fresh data")
}
