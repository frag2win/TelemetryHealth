package storage

import (
	"context"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
	"github.com/frag2win/TelemetryHealth/control-plane/pkg/models"
)

// HealthMetrics is the structured result returned from the health query.
type HealthMetrics struct {
	TenantID            string
	CardinalityMax      uint64
	OrphanCount         uint64
	PreviousOrphanCount uint64
	ActiveServices      uint64
	CompositeScore      float64
	RemediationIssue    string
	Window              time.Time
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

// HealthRepository abstracts all database queries for the health dashboard.
type HealthRepository interface {
	QueryHealthMetrics(ctx context.Context, tenantID string) (*HealthMetrics, error)
	QueryAgentTraces(ctx context.Context) ([]AgentTrace, error)
	GetTenantWeights(ctx context.Context, tenantID string) (telemetry.TenantWeights, error)
	SaveTenantConfig(ctx context.Context, tenantID string, weights telemetry.TenantWeights) error
	LogRemediationEvent(ctx context.Context, tenantID string, issueType string, yamlConfig string, validated, applied bool, actorID, actorRole, sourceIP, action, resourceID string) error
	QuerySpansByTraceID(ctx context.Context, traceID string) ([]models.SpanData, error)
}
