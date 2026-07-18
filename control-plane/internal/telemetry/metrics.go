package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// IngestedSpansTotal tracks the number of individual spans received via OTLP gRPC.
	IngestedSpansTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "telemetryhealth_ingested_spans_total",
		Help: "Total number of spans received by the ingest gateway",
	}, []string{"tenant_id"})

	// KafkaMessagesProcessedTotal tracks the number of messages successfully processed by stream workers.
	KafkaMessagesProcessedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "telemetryhealth_kafka_messages_processed_total",
		Help: "Total number of Kafka messages processed by the stream workers",
	}, []string{"topic"})

	// ApiRequestsTotal tracks the number of HTTP requests to the REST API.
	ApiRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "telemetryhealth_api_requests_total",
		Help: "Total number of HTTP requests to the REST API",
	}, []string{"method", "path", "status"})

	// ApiRequestDuration tracks the latency of API requests.
	ApiRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "telemetryhealth_api_request_duration_seconds",
		Help:    "Latency of API requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	// ClickHouseWriteDuration tracks the latency of inserts to ClickHouse.
	ClickHouseWriteDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "telemetryhealth_clickhouse_write_duration_seconds",
		Help:    "Latency of batch inserts to ClickHouse",
		Buckets: prometheus.DefBuckets,
	}, []string{"table"})

	// PipelineHealthScore tracks the composite health score.
	PipelineHealthScore = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "telemetryhealth_pipeline_health_score",
		Help: "Composite pipeline health score for a tenant",
	}, []string{"tenant_id"})

	// AgentHealthScore tracks the composite health score per agent.
	AgentHealthScore = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "telemetryhealth_agent_health_score",
		Help: "Health score of the AI agent",
	}, []string{"service_name", "agent_id"})

	// AgentTokenBurnRate tracks the LLM token consumption.
	AgentTokenBurnRate = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "telemetryhealth_agent_token_burn_total",
		Help: "Total tokens consumed by the AI agent",
	}, []string{"service_name", "agent_id"})

	// AgentTraceErrorCount tracks the number of trace errors per agent.
	AgentTraceErrorCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "telemetryhealth_agent_trace_error_count",
		Help: "Total trace errors encountered by the AI agent",
	}, []string{"service_name", "agent_id"})
)
