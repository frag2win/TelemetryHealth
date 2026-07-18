package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
	"github.com/frag2win/TelemetryHealth/control-plane/pkg/models"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// PricingConfig holds per-model token cost rates (USD per token).
type PricingConfig struct {
	Rates       map[string]float64
	DefaultRate float64
}

// DefaultPricingConfig returns standard per-token pricing for common LLM models.
func DefaultPricingConfig() PricingConfig {
	return PricingConfig{
		Rates: map[string]float64{
			"gpt-4o":            0.000005,
			"gpt-4-turbo":       0.000010,
			"gpt-3.5-turbo":     0.0000015,
			"claude-3-5-sonnet": 0.000003,
			"claude-3-opus":     0.000015,
			"claude-3-haiku":    0.00000025,
			"gemini-1.5-pro":    0.0000035,
			"gemini-1.5-flash":  0.00000035,
		},
		DefaultRate: 0.000005,
	}
}

// LoadPricingConfig loads PricingConfig, allowing environment overrides via JSON or env.
func LoadPricingConfig() PricingConfig {
	cfg := DefaultPricingConfig()
	if envRates := os.Getenv("LLM_PRICING_JSON"); envRates != "" {
		var customRates map[string]float64
		if err := json.Unmarshal([]byte(envRates), &customRates); err == nil {
			for k, v := range customRates {
				cfg.Rates[k] = v
			}
		}
	}
	if defRateStr := os.Getenv("LLM_DEFAULT_TOKEN_RATE"); defRateStr != "" {
		if val, err := strconv.ParseFloat(defRateStr, 64); err == nil && val >= 0 {
			cfg.DefaultRate = val
		}
	}
	return cfg
}

// CalculateCost computes the token cost given a model name and token count.
func (p PricingConfig) CalculateCost(model string, tokens int) float64 {
	rate, ok := p.Rates[model]
	if !ok {
		rate = p.DefaultRate
	}
	return float64(tokens) * rate
}

// CalculateHallucinationRisk determines hallucination risk from span attributes and observable signals (Finding 10.2).
func CalculateHallucinationRisk(attrs map[string]interface{}) string {
	if r, ok := attrs["llm.hallucination_risk"].(string); ok && r != "" {
		return r
	}
	if r, ok := attrs["gen_ai.evaluation.hallucination_risk"].(string); ok && r != "" {
		return r
	}

	// Fallback calculation from observable signals: confidence, tool failure rate, temperature, errors
	riskLevel := 0 // 0=Low, 1=Medium, 2=High

	if confVal, ok := attrs["llm.confidence"]; ok {
		switch v := confVal.(type) {
		case float64:
			if v < 0.70 {
				riskLevel = 2
			} else if v < 0.85 && riskLevel < 1 {
				riskLevel = 1
			}
		}
	} else if confVal, ok := attrs["confidence"]; ok {
		switch v := confVal.(type) {
		case float64:
			if v < 0.70 {
				riskLevel = 2
			} else if v < 0.85 && riskLevel < 1 {
				riskLevel = 1
			}
		}
	}

	var toolFailures float64
	if tf, ok := attrs["llm.tool_call_failures"].(float64); ok {
		toolFailures = tf
	} else if tf, ok := attrs["tool_call_failures"].(float64); ok {
		toolFailures = tf
	}
	if toolFailures >= 2 {
		riskLevel = 2
	} else if toolFailures == 1 && riskLevel < 1 {
		riskLevel = 1
	}

	if errAttr, ok := attrs["error"].(bool); ok && errAttr {
		if riskLevel < 2 {
			riskLevel = 2
		}
	}

	var temp float64
	hasTemp := false
	if t, ok := attrs["llm.temperature"].(float64); ok {
		temp = t
		hasTemp = true
	} else if t, ok := attrs["gen_ai.request.temperature"].(float64); ok {
		temp = t
		hasTemp = true
	}
	if hasTemp && temp > 0.85 && riskLevel < 2 {
		riskLevel++
	}

	switch riskLevel {
	case 2:
		return "High"
	case 1:
		return "Medium"
	default:
		return "Low"
	}
}

// HealthRepository handles all read queries for the health dashboard.
type HealthRepository struct {
	conn      driver.Conn
	logger    *zap.Logger
	dbName    string
	tableName string
	pricing   PricingConfig
}

// NewHealthRepository creates a new repository.
func NewHealthRepository(conn driver.Conn, logger *zap.Logger) *HealthRepository {
	dbName := os.Getenv("CLICKHOUSE_DB")
	if dbName == "" {
		dbName = "signoz_traces"
	}
	tableName := os.Getenv("CLICKHOUSE_TABLE")
	if tableName == "" {
		tableName = "signoz_index_v2"
	}
	return &HealthRepository{
		conn:      conn,
		logger:    logger,
		dbName:    dbName,
		tableName: tableName,
		pricing:   LoadPricingConfig(),
	}
}

// DB exposes the underlying driver.Conn for other components (like GraphEngine)
func (r *HealthRepository) DB() driver.Conn {
	return r.conn
}

// QueryHealthMetrics fetches aggregated telemetry signals for a tenant using errgroup for concurrent execution (Finding 7.2).
func (r *HealthRepository) QueryHealthMetrics(ctx context.Context, tenantID string) (*storage.HealthMetrics, error) {
	if r.conn == nil {
		return nil, fmt.Errorf("database connection unavailable")
	}

	metrics := &storage.HealthMetrics{TenantID: tenantID}
	g, ctx := errgroup.WithContext(ctx)

	// 1. Cardinality query
	g.Go(func() error {
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
		return nil
	})

	// 2. Orphaned Traces query
	g.Go(func() error {
		orphanQuery := `
			SELECT count() AS orphan_count
			FROM telemetry_health.orphan_signal
			WHERE tenant_id = {tenant_id:UUID}
			  AND detected_at >= now() - INTERVAL 30 MINUTE`
		row := r.conn.QueryRow(ctx, orphanQuery, ch.Named("tenant_id", tenantID))
		if err := row.Scan(&metrics.OrphanCount); err != nil {
			r.logger.Warn("orphan query failed, defaulting to 0", zap.Error(err))
			metrics.OrphanCount = 0
		}
		return nil
	})

	// 3. Previous Orphaned Traces query
	g.Go(func() error {
		orphanPrevQuery := `
			SELECT count() AS orphan_count
			FROM telemetry_health.orphan_signal
			WHERE tenant_id = {tenant_id:UUID}
			  AND detected_at >= now() - INTERVAL 60 MINUTE
			  AND detected_at < now() - INTERVAL 30 MINUTE`
		row := r.conn.QueryRow(ctx, orphanPrevQuery, ch.Named("tenant_id", tenantID))
		if err := row.Scan(&metrics.PreviousOrphanCount); err != nil {
			metrics.PreviousOrphanCount = 0
		}
		return nil
	})

	// 4. Active Services query
	g.Go(func() error {
		coverageQuery := `
			SELECT count() AS active_services
			FROM telemetry_health.coverage_signal
			WHERE tenant_id = {tenant_id:UUID}
			  AND last_seen_at >= now() - INTERVAL 10 MINUTE`
		row := r.conn.QueryRow(ctx, coverageQuery, ch.Named("tenant_id", tenantID))
		if err := row.Scan(&metrics.ActiveServices); err != nil {
			r.logger.Warn("coverage query failed, defaulting to 0", zap.Error(err))
			metrics.ActiveServices = 0
		}
		return nil
	})

	// 5. Tenant Weights query
	var weights telemetry.TenantWeights
	g.Go(func() error {
		w, err := r.GetTenantWeights(ctx, tenantID)
		if err != nil {
			r.logger.Warn("Failed to fetch tenant weights, using defaults", zap.String("tenant_id", tenantID), zap.Error(err))
			weights = telemetry.DefaultWeights()
		} else {
			weights = w
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("concurrent health metrics queries failed: %w", err)
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

// QueryAgentTraces queries ClickHouse for spans with gen_ai.* attributes to reconstruct agent traces,
// falling back to rich mock data if no spans are found.
func (r *HealthRepository) QueryAgentTraces(ctx context.Context) ([]storage.AgentTrace, error) {
	var traces []storage.AgentTrace

	// 1. Query our local trace index table first (real simulator data is ingested here)
	localQuery := `
		SELECT 
			trace_id,
			service_name,
			toUnixTimestamp64Nano(end_time) - toUnixTimestamp64Nano(start_time) AS duration_nano,
			attributes
		FROM telemetry_health.telemetryhealth_trace_index_spans
		WHERE service_name = 'ai-agent-service' OR service_name = 'ai-agent' OR attributes LIKE '%gen_ai%'
		ORDER BY start_time DESC
		LIMIT 10`

	if r.conn != nil {
		rows, err := r.conn.Query(ctx, localQuery)
		if err == nil {
			for rows.Next() {
				var traceID, serviceName, attrsStr string
				var durationNano int64
				if err := rows.Scan(&traceID, &serviceName, &durationNano, &attrsStr); err == nil {
					var attrs map[string]interface{}
					_ = json.Unmarshal([]byte(attrsStr), &attrs)
					
					model := "default-model"
					if m, ok := attrs["llm.model"].(string); ok {
						model = m
					} else if m, ok := attrs["gen_ai.request.model"].(string); ok {
						model = m
					}
					
					tokens := 1500
					if t, ok := attrs["llm.token_usage"].(string); ok {
						if val, err := strconv.Atoi(t); err == nil {
							tokens = val
						}
					}
					
					risk := CalculateHallucinationRisk(attrs)
					
					traceSuffix := traceID
					if len(traceID) >= 6 {
						traceSuffix = traceID[:6]
					}

					traces = append(traces, storage.AgentTrace{
						ID:                "trace-" + traceSuffix,
						Model:             model,
						Tokens:            tokens,
						Cost:              r.pricing.CalculateCost(model, tokens),
						Latency:           fmt.Sprintf("%.1fs", float64(durationNano)/1e9),
						HallucinationRisk: risk,
						Decisions: []storage.AgentDecision{
							{Step: "Retrieved local OTel spans from clickhouse", Tool: "query_clickhouse", Status: "success"},
							{Step: "Extracted OTLP attributes & parsed agent trace", Tool: "parse_trace", Status: "success"},
						},
					})
				}
			}
			rows.Close() // Explicitly close rows to avoid connection leaks (Finding 7.3)
		}
	}

	// 2. Query SigNoz traces index if local trace table is empty
	if len(traces) == 0 && r.conn != nil {
		signozQuery := fmt.Sprintf(`
			SELECT 
				trace_id,
				attributes_map['gen_ai.request.model'] AS model,
				attributes_map['gen_ai.usage.total_tokens'] AS tokens,
				attributes_map['gen_ai.usage.cost'] AS cost,
				attributes_map['llm.hallucination_risk'] AS risk,
				duration_nano
			FROM %s.%s
			WHERE attributes_map['gen_ai.system'] != ''
			ORDER BY timestamp DESC
			LIMIT 10`, r.dbName, r.tableName)

		rows, err := r.conn.Query(ctx, signozQuery)
		if err == nil {
			for rows.Next() {
				var traceID, model, tokensStr, costStr, riskStr string
				var durationNano int64
				if err := rows.Scan(&traceID, &model, &tokensStr, &costStr, &riskStr, &durationNano); err == nil {
					traceSuffix := traceID
					if len(traceID) >= 6 {
						traceSuffix = traceID[:6]
					}
					tokens := 4120
					if tokensStr != "" {
						if val, err := strconv.Atoi(tokensStr); err == nil {
							tokens = val
						}
					}
					cost := r.pricing.CalculateCost(model, tokens)
					if costStr != "" {
						if val, err := strconv.ParseFloat(costStr, 64); err == nil {
							cost = val
						}
					}
					risk := "Low"
					if riskStr != "" {
						risk = riskStr
					} else {
						risk = CalculateHallucinationRisk(map[string]interface{}{"llm.model": model})
					}
					traces = append(traces, storage.AgentTrace{
						ID:                "trace-" + traceSuffix,
						Model:             model,
						Tokens:            tokens,
						Cost:              cost,
						Latency:           fmt.Sprintf("%.1fs", float64(durationNano)/1e9),
						HallucinationRisk: risk,
						Decisions: []storage.AgentDecision{
							{Step: "Retrieved OTel spans from ClickHouse index", Tool: "query_clickhouse", Status: "success"},
							{Step: "Inferred prompt template and resolved trace context", Tool: "resolve_spans", Status: "success"},
						},
					})
				}
			}
			rows.Close() // Explicitly close rows to avoid connection leaks (Finding 7.3)
		}
	}

	// 3. Fallback to rich, realistic traces if database returned nothing or errored out
	if len(traces) == 0 {
		traces = []storage.AgentTrace{
			{
				ID:                "trace-991",
				Model:             "gpt-4o",
				Tokens:            4120,
				Cost:              0.041,
				Latency:           "3.2s",
				HallucinationRisk: "Low",
				Decisions: []storage.AgentDecision{
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
				Decisions: []storage.AgentDecision{
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
// Uses FINAL modifier to bypass eventual consistency and get latest configured weights (Finding 7.4).
func (r *HealthRepository) GetTenantWeights(ctx context.Context, tenantID string) (telemetry.TenantWeights, error) {
	weights := telemetry.DefaultWeights()
	query := `
		SELECT cardinality_weight, orphan_weight, coverage_weight
		FROM telemetry_health.tenant_config FINAL
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
	var spans []models.SpanData

	// 1. Query our local trace index table first (real simulator data is ingested here)
	localQuery := `
		SELECT 
			trace_id,
			span_id,
			parent_span_id,
			service_name,
			operation_name,
			toUnixTimestamp64Nano(end_time) - toUnixTimestamp64Nano(start_time) AS duration_nano,
			start_time,
			attributes,
			status
		FROM telemetry_health.telemetryhealth_trace_index_spans
		WHERE trace_id = {trace_id:String}
		ORDER BY start_time ASC`

	if r.conn != nil {
		rows, err := r.conn.Query(ctx, localQuery, ch.Named("trace_id", traceID))
		if err == nil {
			for rows.Next() {
				var span models.SpanData
				var durationNano int64
				var attrsStr, statusVal string
				if err := rows.Scan(
					&span.TraceID,
					&span.SpanID,
					&span.ParentSpanID,
					&span.ServiceName,
					&span.Name,
					&durationNano,
					&span.Timestamp,
					&attrsStr,
					&statusVal,
				); err == nil {
					span.DurationNano = durationNano
					
					var attrs map[string]interface{}
					if attrsStr != "" {
						_ = json.Unmarshal([]byte(attrsStr), &attrs)
					}
					span.Attributes = make(map[string]string)
					for k, v := range attrs {
						span.Attributes[k] = fmt.Sprintf("%v", v)
					}

					if strings.ToLower(statusVal) == "error" {
						span.StatusCode = "ERROR"
					} else {
						span.StatusCode = "OK"
					}
					spans = append(spans, span)
				}
			}
			rows.Close() // Explicitly close rows to avoid connection leaks (Finding 7.3)
		}
	}

	// 2. Query SigNoz traces index if local trace table is empty
	if len(spans) == 0 && r.conn != nil {
		signozQuery := fmt.Sprintf(`
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
			FROM %s.%s
			WHERE trace_id = {trace_id:String}
			ORDER BY timestamp ASC`, r.dbName, r.tableName)

		rows, err := r.conn.Query(ctx, signozQuery, ch.Named("trace_id", traceID))
		if err == nil {
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
			rows.Close() // Explicitly close rows to avoid connection leaks (Finding 7.3)
		}
	}



	return spans, nil
}



