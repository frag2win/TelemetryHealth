package processor

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type metricsHelper struct {
	meter             metric.Meter
	agentHealthScore  metric.Float64Gauge
	agentTokenBurn    metric.Int64Counter
	agentTraceErrors  metric.Int64Counter
	logger            *zap.Logger
}

func newMetricsHelper(mp metric.MeterProvider, logger *zap.Logger) (*metricsHelper, error) {
	if mp == nil {
		return nil, nil
	}
	meter := mp.Meter("github.com/frag2win/TelemetryHealth/processor")
	
	healthScore, err := meter.Float64Gauge(
		"telemetryhealth_agent_health_score",
		metric.WithDescription("Composite health score of AI Agent traces"),
	)
	if err != nil {
		return nil, err
	}

	tokenBurn, err := meter.Int64Counter(
		"telemetryhealth_agent_token_burn_total",
		metric.WithDescription("Tracking AI Agent token consumption total"),
	)
	if err != nil {
		return nil, err
	}

	traceErrors, err := meter.Int64Counter(
		"telemetryhealth_agent_trace_error_count",
		metric.WithDescription("Tracking aggregate AI Agent trace failure trends"),
	)
	if err != nil {
		return nil, err
	}

	return &metricsHelper{
		meter:             meter,
		agentHealthScore:  healthScore,
		agentTokenBurn:    tokenBurn,
		agentTraceErrors:  traceErrors,
		logger:            logger,
	}, nil
}

func (h *metricsHelper) RecordAgentHealth(ctx context.Context, serviceName, agentID string, score float64) {
	if h == nil {
		return
	}
	h.agentHealthScore.Record(ctx, score, metric.WithAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("agent_id", agentID),
	))
}

func (h *metricsHelper) RecordTokenBurn(ctx context.Context, serviceName, agentID string, tokens int64) {
	if h == nil {
		return
	}
	h.agentTokenBurn.Add(ctx, tokens, metric.WithAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("agent_id", agentID),
	))
}

func (h *metricsHelper) RecordTraceError(ctx context.Context, serviceName, agentID string) {
	if h == nil {
		return
	}
	h.agentTraceErrors.Add(ctx, 1, metric.WithAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("agent_id", agentID),
	))
}
