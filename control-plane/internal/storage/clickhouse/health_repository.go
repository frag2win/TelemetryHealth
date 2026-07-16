package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
	"github.com/frag2win/TelemetryHealth/control-plane/pkg/models"
	"go.uber.org/zap"
)

// HealthMetrics is the structured result returned from the health query.
type HealthMetrics struct {
	TenantID            string
	CardinalityMax      uint64
	OrphanCount         uint64
	PreviousOrphanCount uint64
	ActiveServices      uint64
	CompositeScore   float64
	RemediationIssue string
	Window           time.Time
}

// HealthRepository handles all read queries for the health dashboard.
type HealthRepository struct {
	conn   driver.Conn
	logger *zap.Logger
}

// NewHealthRepository creates a new repository.
func NewHealthRepository(conn driver.Conn, logger *zap.Logger) *HealthRepository {
	return &HealthRepository{conn: conn, logger: logger}
}

// DB exposes the underlying driver.Conn for other components (like GraphEngine)
func (r *HealthRepository) DB() driver.Conn {
	return r.conn
}

// QueryHealthMetrics fetches aggregated telemetry signals for a tenant.
func (r *HealthRepository) QueryHealthMetrics(ctx context.Context, tenantID string) (*HealthMetrics, error) {
	metrics := &HealthMetrics{TenantID: tenantID}

	// --- 1. Cardinality: peak unique estimate in last 30 min ---
	cardQuery := `
		SELECT max(unique_estimate) AS max_cardinality
		FROM telemetry_health.cardinality_signal
		WHERE tenant_id = {tenant_id:UUID}
		  AND window_start >= now() - INTERVAL 30 MINUTE`

	row := r.conn.QueryRow(ctx, cardQuery, ch.Named("tenant_id", tenantID))
	if err := row.Scan(&metrics.CardinalityMax); err != nil {
		r.logger.Warn("cardinality query failed, defaulting to 0", zap.Error(err))
		metrics.CardinalityMax = 0
	}

	// --- 2. Orphaned Traces: count in last 30 min ---
	orphanQuery := `
		SELECT count() AS orphan_count
		FROM telemetry_health.orphan_signal
		WHERE tenant_id = {tenant_id:UUID}
		  AND detected_at >= now() - INTERVAL 30 MINUTE`

	row2 := r.conn.QueryRow(ctx, orphanQuery, ch.Named("tenant_id", tenantID))
	if err := row2.Scan(&metrics.OrphanCount); err != nil {
		r.logger.Warn("orphan query failed, defaulting to 0", zap.Error(err))
		metrics.OrphanCount = 0
	}

	orphanPrevQuery := `
		SELECT count() AS orphan_count
		FROM telemetry_health.orphan_signal
		WHERE tenant_id = {tenant_id:UUID}
		  AND detected_at >= now() - INTERVAL 60 MINUTE
		  AND detected_at < now() - INTERVAL 30 MINUTE`
	rowPrev := r.conn.QueryRow(ctx, orphanPrevQuery, ch.Named("tenant_id", tenantID))
	if err := rowPrev.Scan(&metrics.PreviousOrphanCount); err != nil {
		metrics.PreviousOrphanCount = 0
	}

	// --- 3. Active Services: distinct services seen in last 10 min ---
	coverageQuery := `
		SELECT count() AS active_services
		FROM telemetry_health.coverage_signal
		WHERE tenant_id = {tenant_id:UUID}
		  AND last_seen_at >= now() - INTERVAL 10 MINUTE`

	row3 := r.conn.QueryRow(ctx, coverageQuery, ch.Named("tenant_id", tenantID))
	if err := row3.Scan(&metrics.ActiveServices); err != nil {
		r.logger.Warn("coverage query failed, defaulting to 0", zap.Error(err))
		metrics.ActiveServices = 0
	}

	// --- 4. Composite Health Score ---
	weights, err := r.GetTenantWeights(ctx, tenantID)
	if err != nil {
		r.logger.Warn("Failed to fetch tenant weights, using defaults", zap.String("tenant_id", tenantID), zap.Error(err))
		weights = telemetry.DefaultWeights()
	}
	metrics.CompositeScore = telemetry.CalculateHealthScore(metrics.CardinalityMax, metrics.OrphanCount, metrics.ActiveServices, weights)

	if metrics.CardinalityMax > 1_000_000 {
		metrics.RemediationIssue = fmt.Sprintf("High cardinality detected: %d unique values", metrics.CardinalityMax)
	} else if metrics.OrphanCount > 100 {
		metrics.RemediationIssue = fmt.Sprintf("Elevated orphan spans: %d in last 30m", metrics.OrphanCount)
	}

	return metrics, nil
}

// Named wraps clickhouse.Named for query parameterization.
func Named(name string, value interface{}) interface{} {
	return ch.Named(name, value)
}

// AgentDecision represents a single decision step in an agent trace.
type AgentDecision struct {
	Step   string `json:"step"`
	Tool   string `json:"tool"`
	Status string `json:"status"`
}

// AgentTrace represents the execution details of an LLM agent query.
type AgentTrace struct {
	ID                string          `json:"id"`
	Model             string          `json:"model"`
	Tokens            int             `json:"tokens"`
	Cost              float64         `json:"cost"`
	Latency           string          `json:"latency"`
	HallucinationRisk string          `json:"hallucinationRisk"`
	Decisions         []AgentDecision `json:"decisions"`
}

// QueryAgentTraces queries ClickHouse for spans with gen_ai.* attributes to reconstruct agent traces,
// falling back to rich mock data if no spans are found.
func (r *HealthRepository) QueryAgentTraces(ctx context.Context) ([]AgentTrace, error) {
	// Attempt to query SigNoz traces index if it exists
	query := `
		SELECT 
			trace_id,
			attributes_map['gen_ai.request.model'] AS model,
			attributes_map['gen_ai.usage.total_tokens'] AS tokens,
			attributes_map['gen_ai.usage.cost'] AS cost,
			duration_nano
		FROM signoz_traces.signoz_index_v2
		WHERE attributes_map['gen_ai.system'] != ''
		ORDER BY timestamp DESC
		LIMIT 10`

	var traces []AgentTrace
	rows, err := r.conn.Query(ctx, query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var traceID, model, tokensStr, costStr string
			var durationNano int64
			if err := rows.Scan(&traceID, &model, &tokensStr, &costStr, &durationNano); err == nil {
				traceSuffix := traceID
				if len(traceID) >= 6 {
					traceSuffix = traceID[:6]
				}
				traces = append(traces, AgentTrace{
					ID:                "trace-" + traceSuffix,
					Model:             model,
					Tokens:            4120, // default placeholder
					Cost:              0.035,
					Latency:           fmt.Sprintf("%.1fs", float64(durationNano)/1e9),
					HallucinationRisk: "Low",
					Decisions: []AgentDecision{
						{Step: "Retrieved OTel spans from ClickHouse index", Tool: "query_clickhouse", Status: "success"},
						{Step: "Inferred prompt template and resolved trace context", Tool: "resolve_spans", Status: "success"},
					},
				})
			}
		}
	}

	// Fallback to rich, realistic traces if ClickHouse returned nothing or errored out
	if len(traces) == 0 {
		traces = []AgentTrace{
			{
				ID:                "trace-991",
				Model:             "gpt-4o",
				Tokens:            4120,
				Cost:              0.041,
				Latency:           "3.2s",
				HallucinationRisk: "Low",
				Decisions: []AgentDecision{
					{Step: "Retrieved 15 similar spans from ClickHouse (gen_ai.system)", Tool: "query_clickhouse", Status: "success"},
					{Step: "Analyzed cardinality distribution for user_id", Tool: "python_eval", Status: "success"},
					{Step: "Generated remediation YAML via SigNoz MCP tool", Tool: "generate_yaml", Status: "success"},
				},
			},
			{
				ID:                "trace-992",
				Model:             "claude-3-5-sonnet",
				Tokens:            8450,
				Cost:              0.025,
				Latency:           "6.1s",
				HallucinationRisk: "High",
				Decisions: []AgentDecision{
					{Step: "Attempted to query missing index (gen_ai.request.model)", Tool: "query_clickhouse", Status: "error"},
					{Step: "Retried with full table scan (token limit warning)", Tool: "query_clickhouse", Status: "warning"},
					{Step: "Formulated remediation with unverified field names", Tool: "generate_yaml", Status: "warning"},
				},
			},
		}
	}

	return traces, nil
}

// GetTenantWeights fetches the configurable health score weights for a tenant, falling back to defaults if not configured (PRD §8.4).
func (r *HealthRepository) GetTenantWeights(ctx context.Context, tenantID string) (telemetry.TenantWeights, error) {
	weights := telemetry.DefaultWeights()
	query := `
		SELECT cardinality_weight, orphan_weight, coverage_weight
		FROM telemetry_health.tenant_config
		WHERE tenant_id = {tenant_id:UUID}
		LIMIT 1`
	row := r.conn.QueryRow(ctx, query, ch.Named("tenant_id", tenantID))
	if err := row.Scan(&weights.CardinalityWeight, &weights.OrphanWeight, &weights.CoverageWeight); err != nil {
		// Log but return defaults
		r.logger.Debug("Using default weights for tenant", zap.String("tenant_id", tenantID), zap.Error(err))
	}
	return weights, nil
}

// SaveTenantConfig saves the health score weights for a tenant (PRD §8.4).
func (r *HealthRepository) SaveTenantConfig(ctx context.Context, tenantID string, weights telemetry.TenantWeights) error {
	query := `
		INSERT INTO telemetry_health.tenant_config (tenant_id, cardinality_weight, orphan_weight, coverage_weight, updated_at)
		VALUES ({tenant_id:UUID}, {card_w:Float64}, {orphan_w:Float64}, {cov_w:Float64}, {now:DateTime64})`
	err := r.conn.Exec(ctx, query,
		ch.Named("tenant_id", tenantID),
		ch.Named("card_w", weights.CardinalityWeight),
		ch.Named("orphan_w", weights.OrphanWeight),
		ch.Named("cov_w", weights.CoverageWeight),
		ch.Named("now", time.Now()),
	)
	if err != nil {
		r.logger.Error("Failed to save tenant config", zap.String("tenant_id", tenantID), zap.Error(err))
		return err
	}
	return nil
}

// LogRemediationEvent logs a remediation event to the ClickHouse audit table (PRD §10, Improvement #19).
func (r *HealthRepository) LogRemediationEvent(ctx context.Context, tenantID string, issueType string, yamlConfig string, validated, applied bool, actorID, actorRole, sourceIP, action, resourceID string) error {
	query := `
		INSERT INTO telemetry_health.remediation_event (
			tenant_id, issue_type, generated_yaml, validated, applied, actor_id, actor_role, source_ip, action, resource_id, ts
		) VALUES (
			{tenant_id:UUID}, {issue_type:LowCardinality(String)}, {yaml:String}, {validated:UInt8}, {applied:UInt8},
			{actor_id:String}, {actor_role:LowCardinality(String)}, {source_ip:String}, {action:LowCardinality(String)}, {resource_id:String}, {ts:DateTime64}
		)`
	
	valVal := uint8(0)
	if validated {
		valVal = 1
	}
	appVal := uint8(0)
	if applied {
		appVal = 1
	}

	err := r.conn.Exec(ctx, query,
		ch.Named("tenant_id", tenantID),
		ch.Named("issue_type", issueType),
		ch.Named("yaml", yamlConfig),
		ch.Named("validated", valVal),
		ch.Named("applied", appVal),
		ch.Named("actor_id", actorID),
		ch.Named("actor_role", actorRole),
		ch.Named("source_ip", sourceIP),
		ch.Named("action", action),
		ch.Named("resource_id", resourceID),
		ch.Named("ts", time.Now()),
	)
	if err != nil {
		r.logger.Error("Failed to log remediation audit event", zap.String("tenant_id", tenantID), zap.Error(err))
		return err
	}
	return nil
}

// Ensure driver package is used.
var _ driver.Conn

// QuerySpansByTraceID fetches spans for a given traceID from ClickHouse.
// Falls back to mock data if no spans are found.
func (r *HealthRepository) QuerySpansByTraceID(ctx context.Context, traceID string) ([]models.SpanData, error) {
	// Attempt to query clickhouse first
	query := `
		SELECT 
			trace_id,
			span_id,
			parent_span_id,
			service_name,
			name,
			duration_nano,
			timestamp,
			attributes_map,
			status_code
		FROM signoz_traces.signoz_index_v2
		WHERE trace_id = {trace_id:String}
		ORDER BY timestamp ASC`

	var spans []models.SpanData
	if r.conn != nil {
		rows, err := r.conn.Query(ctx, query, ch.Named("trace_id", traceID))
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var span models.SpanData
				var statusVal uint8
				if err := rows.Scan(
					&span.TraceID,
					&span.SpanID,
					&span.ParentSpanID,
					&span.ServiceName,
					&span.Name,
					&span.DurationNano,
					&span.Timestamp,
					&span.Attributes,
					&statusVal,
				); err == nil {
					if statusVal == 2 { // OpenTelemetry ERROR is status code 2
						span.StatusCode = "ERROR"
					} else {
						span.StatusCode = "OK"
					}
					spans = append(spans, span)
				}
			}
		} else {
			r.logger.Warn("Failed to query ClickHouse for spans, falling back to mock trace data", zap.String("trace_id", traceID), zap.Error(err))
		}
	} else {
		r.logger.Info("ClickHouse connection is nil, skipping query and falling back to mock trace data", zap.String("trace_id", traceID))
	}

	// Fallback to mock data if empty or error
	if len(spans) == 0 {
		r.logger.Info("No spans found in ClickHouse, generating mock trace", zap.String("trace_id", traceID))
		spans = r.generateMockSpans(traceID)
	}

	return spans, nil
}

// generateMockSpans creates realistic traces for local development and demos.
func (r *HealthRepository) generateMockSpans(traceID string) []models.SpanData {
	now := time.Now()

	// Pattern 1: Tool Timeout and Retry (trace-992 or similar)
	if strings.Contains(traceID, "992") || strings.Contains(traceID, "fail") || strings.Contains(traceID, "timeout") {
		return []models.SpanData{
			{
				TraceID:      traceID,
				SpanID:       "span-root",
				ParentSpanID: "",
				ServiceName:  "ai-agent",
				Name:         "agent.workflow",
				DurationNano: int64(3000 * time.Millisecond),
				Timestamp:    now,
				Attributes: map[string]string{
					"workflow.topic": "Observability best practices",
				},
				StatusCode: "ERROR",
			},
			{
				TraceID:      traceID,
				SpanID:       "span-tool-fail",
				ParentSpanID: "span-root",
				ServiceName:  "ai-agent",
				Name:         "agent.research",
				DurationNano: int64(1000 * time.Millisecond),
				Timestamp:    now.Add(100 * time.Millisecond),
				Attributes: map[string]string{
					"llm.tool_name":       "web_search",
					"llm.tool_call.error": "TimeoutError: connection refused",
				},
				StatusCode: "ERROR",
			},
			{
				TraceID:      traceID,
				SpanID:       "span-tool-retry",
				ParentSpanID: "span-root",
				ServiceName:  "ai-agent",
				Name:         "agent.research",
				DurationNano: int64(800 * time.Millisecond),
				Timestamp:    now.Add(1200 * time.Millisecond),
				Attributes: map[string]string{
					"llm.tool_name": "web_search",
				},
				StatusCode: "OK",
			},
			{
				TraceID:      traceID,
				SpanID:       "span-llm-summarize",
				ParentSpanID: "span-root",
				ServiceName:  "ai-agent",
				Name:         "agent.summarize",
				DurationNano: int64(1200 * time.Millisecond),
				Timestamp:    now.Add(2100 * time.Millisecond),
				Attributes: map[string]string{
					"llm.model":           "gpt-4o",
					"llm.token_usage":     "1250",
					"llm.prompt.raw_abcd": "Information about Observability best practices",
				},
				StatusCode: "OK",
			},
		}
	}

	// Pattern 2: Token Limit Exceeded
	if strings.Contains(traceID, "token") || strings.Contains(traceID, "limit") {
		return []models.SpanData{
			{
				TraceID:      traceID,
				SpanID:       "span-root",
				ParentSpanID: "",
				ServiceName:  "ai-agent",
				Name:         "agent.workflow",
				DurationNano: int64(2200 * time.Millisecond),
				Timestamp:    now,
				Attributes: map[string]string{
					"workflow.topic": "Vector databases",
				},
				StatusCode: "OK",
			},
			{
				TraceID:      traceID,
				SpanID:       "span-tool",
				ParentSpanID: "span-root",
				ServiceName:  "ai-agent",
				Name:         "agent.research",
				DurationNano: int64(900 * time.Millisecond),
				Timestamp:    now.Add(100 * time.Millisecond),
				Attributes: map[string]string{
					"llm.tool_name": "web_search",
				},
				StatusCode: "OK",
			},
			{
				TraceID:      traceID,
				SpanID:       "span-llm-heavy",
				ParentSpanID: "span-root",
				ServiceName:  "ai-agent",
				Name:         "agent.summarize",
				DurationNano: int64(1200 * time.Millisecond),
				Timestamp:    now.Add(1000 * time.Millisecond),
				Attributes: map[string]string{
					"llm.model":       "gpt-4o",
					"llm.token_usage": "5200",
				},
				StatusCode: "OK",
			},
		}
	}

	// Pattern 3: Retrieval Collapse
	if strings.Contains(traceID, "retrieve") || strings.Contains(traceID, "collapse") {
		return []models.SpanData{
			{
				TraceID:      traceID,
				SpanID:       "span-root",
				ParentSpanID: "",
				ServiceName:  "ai-agent",
				Name:         "agent.workflow",
				DurationNano: int64(1500 * time.Millisecond),
				Timestamp:    now,
				Attributes:   map[string]string{"workflow.topic": "nonexistent term"},
				StatusCode:   "OK",
			},
			{
				TraceID:      traceID,
				SpanID:       "span-retriever",
				ParentSpanID: "span-root",
				ServiceName:  "ai-agent",
				Name:         "agent.retrieve",
				DurationNano: int64(300 * time.Millisecond),
				Timestamp:    now.Add(100 * time.Millisecond),
				Attributes: map[string]string{
					"retriever.documents_count": "0",
				},
				StatusCode: "OK",
			},
			{
				TraceID:      traceID,
				SpanID:       "span-llm",
				ParentSpanID: "span-root",
				ServiceName:  "ai-agent",
				Name:         "agent.summarize",
				DurationNano: int64(1000 * time.Millisecond),
				Timestamp:    now.Add(400 * time.Millisecond),
				Attributes: map[string]string{
					"llm.model": "gpt-4o",
				},
				StatusCode: "OK",
			},
		}
	}

	// Pattern 4: Normal Flow (trace-991 or default)
	return []models.SpanData{
		{
			TraceID:      traceID,
			SpanID:       "span-root",
			ParentSpanID: "",
			ServiceName:  "ai-agent",
			Name:         "agent.workflow",
			DurationNano: int64(2500 * time.Millisecond),
			Timestamp:    now,
			Attributes: map[string]string{
				"workflow.topic": "LLM cost optimization",
			},
			StatusCode: "OK",
		},
		{
			TraceID:      traceID,
			SpanID:       "span-tool",
			ParentSpanID: "span-root",
			ServiceName:  "ai-agent",
			Name:         "agent.research",
			DurationNano: int64(1100 * time.Millisecond),
			Timestamp:    now.Add(100 * time.Millisecond),
			Attributes: map[string]string{
				"llm.tool_name": "web_search",
			},
			StatusCode: "OK",
		},
		{
			TraceID:      traceID,
			SpanID:       "span-llm",
			ParentSpanID: "span-root",
			ServiceName:  "ai-agent",
			Name:         "agent.summarize",
			DurationNano: int64(1200 * time.Millisecond),
			Timestamp:    now.Add(1250 * time.Millisecond),
			Attributes: map[string]string{
				"llm.model":       "gpt-4o",
				"llm.token_usage": "980",
			},
			StatusCode: "OK",
		},
	}
}

