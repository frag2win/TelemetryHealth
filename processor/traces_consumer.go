package processor

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type tracesConsumer struct {
	baseConsumer
	next    consumer.Traces
	logger  *zap.Logger
	metrics *metricsHelper

	controlPlaneEndpoint string
	tenantID             string
	conn                 *grpc.ClientConn
	client               ptraceotlp.GRPCClient
	stopChan             chan struct{}
}

func newTracesConsumer(set processor.Settings, cfg component.Config, next consumer.Traces) (processor.Traces, error) {
	bc, err := newBaseConsumer(cfg, set.Logger)
	if err != nil {
		return nil, err
	}
	mh, err := newMetricsHelper(set.TelemetrySettings.MeterProvider, set.Logger)
	if err != nil {
		return nil, err
	}
	procCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type: expected *Config, got %T", cfg)
	}
	return &tracesConsumer{
		baseConsumer:         bc,
		next:                 next,
		logger:               set.Logger,
		metrics:              mh,
		controlPlaneEndpoint: procCfg.ControlPlaneEndpoint,
		tenantID:             procCfg.TenantID,
		stopChan:             make(chan struct{}),
	}, nil
}

func (c *tracesConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (c *tracesConsumer) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	// Step 1: Extract structural tuples BEFORE sampling decisions are applied.
	// Ship [trace_id, span_id, parent_span_id] to EXP2 regardless of sampling outcome.
	// This prevents the sampling correlation paradox (PRD §8.2).
	if err := c.cb.Execute(ctx, func(ctx context.Context) error {
		rss := td.ResourceSpans()
		for i := 0; i < rss.Len(); i++ {
			rs := rss.At(i)
			serviceName := ""
			if v, ok := rs.Resource().Attributes().Get("service.name"); ok {
				serviceName = v.Str()
			}
			sss := rs.ScopeSpans()
			for j := 0; j < sss.Len(); j++ {
				ss := sss.At(j)
				spans := ss.Spans()
				for k := 0; k < spans.Len(); k++ {
					span := spans.At(k)
					sig := HealthSignal{
						TraceID:      span.TraceID().String(),
						SpanID:       span.SpanID().String(),
						ParentSpanID: span.ParentSpanID().String(),
						ServiceName:  serviceName,
					}
					// EmitHealthSignal drops if queue is full — never blocks (PRD §10)
					c.EmitHealthSignal(ctx, sig)
				}
			}
		}
		return nil
	}); err != nil {
		c.logger.Error("traces consumer health extraction failed, failing open", zap.Error(err))
	}

	// Step 2: Enrich and filter trace data (built defensively to fail-open)
	c.processTracesDefensively(ctx, td)

	// Step 3: Pass the processed trace data to the next consumer (EXP1)
	return c.next.ConsumeTraces(ctx, td)
}

func (c *tracesConsumer) processTracesDefensively(ctx context.Context, td ptrace.Traces) {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Error("Panic recovered during traces processing, failing open", zap.Any("panic", r))
		}
	}()

	// Pass 1: Identify AI agent traces and map trace_id -> agent_id
	aiAgentTraces := make(map[string]bool)
	agentIDs := make(map[string]string)
	traceHasErrors := make(map[string]bool)

	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		sss := rs.ScopeSpans()
		for j := 0; j < sss.Len(); j++ {
			ss := sss.At(j)
			spans := ss.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				traceIDStr := span.TraceID().String()

				// Track if span has error
				if span.Status().Code() == ptrace.StatusCodeError {
					traceHasErrors[traceIDStr] = true
				}

				hasAIContext := false
				span.Attributes().Range(func(key string, val pcommon.Value) bool {
					if key == "agent_id" {
						agentIDs[traceIDStr] = val.Str()
						hasAIContext = true
					}
					if strings.HasPrefix(key, "llm.") {
						hasAIContext = true
					}
					if key == "telemetry.health.ai_agent" {
						if (val.Type() == pcommon.ValueTypeBool && val.Bool()) ||
							(val.Type() == pcommon.ValueTypeStr && val.Str() == "true") {
							hasAIContext = true
						}
					}
					return true
				})
				if hasAIContext {
					aiAgentTraces[traceIDStr] = true
				}
			}
		}
	}

	// Pass 2: Enrich spans & perform cardinality filtering
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		serviceName := "unknown-service"
		if v, ok := rs.Resource().Attributes().Get("service.name"); ok {
			serviceName = v.Str()
		}
		sss := rs.ScopeSpans()
		for j := 0; j < sss.Len(); j++ {
			ss := sss.At(j)
			spans := ss.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				traceIDStr := span.TraceID().String()

				isAIAgentTrace := aiAgentTraces[traceIDStr]
				agentID := agentIDs[traceIDStr]
				if agentID == "" {
					agentID = "unknown-agent"
				}

				// Automatically tag traces belonging to AI agents
				if isAIAgentTrace {
					span.Attributes().PutBool("telemetry.health.ai_agent", true)
				}

				// Cardinality filter: cap distinct attribute keys per service at 100
				var keysToRemove []string
				var tokensBurned int64

				span.Attributes().Range(func(key string, val pcommon.Value) bool {
					// Track token burn attributes if present using exact OTel GenAI semantic convention keys (Finding 9.1)
					isTokenKey := false
					for _, tk := range []string{
						"gen_ai.usage.input_tokens",
						"gen_ai.usage.output_tokens",
						"gen_ai.usage.total_tokens",
						"llm.usage.total_tokens",
					} {
						if key == tk {
							isTokenKey = true
							break
						}
					}
					if isTokenKey {
						if val.Type() == pcommon.ValueTypeInt {
							tokensBurned += val.Int()
						}
					}

					// Never drop vital identification or service tags
					if key == "service.name" || key == "telemetry.health.ai_agent" || key == "agent_id" {
						return true
					}

					valStr := val.AsString()
					allowed := c.tracker.Observe(serviceName, key, valStr)
					if !allowed {
						keysToRemove = append(keysToRemove, key)
					}
					return true
				})

				for _, key := range keysToRemove {
					span.Attributes().Remove(key)
				}

				// Record real-time metrics for AI agent traces
				if isAIAgentTrace {
					// Token burn rate
					if tokensBurned > 0 {
						c.metrics.RecordTokenBurn(ctx, serviceName, agentID, tokensBurned)
					}

					// Trace errors count (per error span)
					if span.Status().Code() == ptrace.StatusCodeError {
						c.metrics.RecordTraceError(ctx, serviceName, agentID)
					}
				}
			}
		}
	}

	traceToServiceMap := make(map[string]string, td.SpanCount())
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		serviceName := "unknown-service"
		if v, ok := rs.Resource().Attributes().Get("service.name"); ok {
			serviceName = v.Str()
		}
		sss := rs.ScopeSpans()
		for j := 0; j < sss.Len(); j++ {
			ss := sss.At(j)
			spans := ss.Spans()
			for k := 0; k < spans.Len(); k++ {
				traceToServiceMap[spans.At(k).TraceID().String()] = serviceName
			}
		}
	}

	// Record health scores per trace/agent
	for traceIDStr, isAIAgent := range aiAgentTraces {
		if isAIAgent {
			agentID := agentIDs[traceIDStr]
			if agentID == "" {
				agentID = "unknown-agent"
			}
			score := 1.0
			if traceHasErrors[traceIDStr] {
				score = 0.0
			}
			
			serviceName, exists := traceToServiceMap[traceIDStr]
			if !exists {
				serviceName = "unknown-service"
			}

			c.metrics.RecordAgentHealth(ctx, serviceName, agentID, score)
		}
	}
}

func (c *tracesConsumer) Start(ctx context.Context, host component.Host) error {
	c.logger.Info("TracesConsumer started")
	if c.controlPlaneEndpoint != "" {
		c.logger.Info("Connecting to control plane", zap.String("endpoint", c.controlPlaneEndpoint))
		conn, err := grpc.Dial(c.controlPlaneEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			c.logger.Error("Failed to connect to control plane endpoint", zap.Error(err))
			return err
		}
		c.conn = conn
		c.client = ptraceotlp.NewGRPCClient(conn)
		go c.exportLoop()
	} else {
		c.logger.Warn("control_plane_endpoint not configured; health signals will not be exported")
	}
	return nil
}

func (c *tracesConsumer) Shutdown(ctx context.Context) error {
	c.logger.Info("TracesConsumer shutting down")
	close(c.stopChan)
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

func (c *tracesConsumer) exportLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.exportPendingSignals()
		case <-c.stopChan:
			return
		}
	}
}

func (c *tracesConsumer) exportPendingSignals() {
	signals := c.DrainHealthSignals()
	if len(signals) == 0 {
		return
	}

	td := ptrace.NewTraces()

	type groupKey struct {
		tenantID    string
		serviceName string
	}

	grouped := make(map[groupKey][]HealthSignal)
	for _, sig := range signals {
		tenant := sig.TenantID
		if tenant == "" {
			tenant = c.tenantID
		}
		if tenant == "" {
			tenant = "unknown"
		}

		svc := sig.ServiceName
		if svc == "" {
			svc = "unknown"
		}

		k := groupKey{tenantID: tenant, serviceName: svc}
		grouped[k] = append(grouped[k], sig)
	}

	for key, sigs := range grouped {
		rs := td.ResourceSpans().AppendEmpty()
		rs.Resource().Attributes().PutStr("tenant_id", key.tenantID)
		rs.Resource().Attributes().PutStr("service.name", key.serviceName)

		scopeSpans := rs.ScopeSpans().AppendEmpty()
		scopeSpans.Scope().SetName("telemetryhealth-exporter")

		for _, sig := range sigs {
			span := scopeSpans.Spans().AppendEmpty()

			var tid pcommon.TraceID
			if b, err := hex.DecodeString(sig.TraceID); err == nil && len(b) == 16 {
				copy(tid[:], b)
			}
			span.SetTraceID(tid)

			var sid pcommon.SpanID
			if b, err := hex.DecodeString(sig.SpanID); err == nil && len(b) == 8 {
				copy(sid[:], b)
			}
			span.SetSpanID(sid)

			var psid pcommon.SpanID
			if b, err := hex.DecodeString(sig.ParentSpanID); err == nil && len(b) == 8 {
				copy(psid[:], b)
			}
			span.SetParentSpanID(psid)

			span.SetName("health-signal")
			span.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
			span.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		}
	}

	tenantToSend := c.tenantID
	if tenantToSend == "" {
		tenantToSend = "unknown"
	}
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-tenant-id", tenantToSend))

	req := ptraceotlp.NewExportRequestFromTraces(td)
	_, err := c.client.Export(ctx, req)
	if err != nil {
		c.logger.Error("Failed to export health signals to control plane", zap.Error(err))
	} else {
		c.logger.Debug("Successfully exported health signals", zap.Int("count", len(signals)))
	}
}
