package mcp

import (
	"context"
	"fmt"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/remediation"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage"
	"go.uber.org/zap"
)

type MetricValue struct {
	Value  string  `json:"value"`
	Change float64 `json:"change"`
}

type MetricsPayload struct {
	Cardinality MetricValue `json:"cardinality"`
	Orphans     MetricValue `json:"orphans"`
	Coverage    MetricValue `json:"coverage"`
}

type RemediationPayload struct {
	IssueType string `json:"issueType"`
	Yaml      string `json:"yaml"`
	Validated bool   `json:"validated"`
}

type HealthResponse struct {
	HealthScore float64            `json:"healthScore"`
	Metrics     MetricsPayload     `json:"metrics"`
	Remediation RemediationPayload `json:"remediation"`
	TenantId    string             `json:"tenantId"`
	Version     string             `json:"version"`
}

// Toolset holds the real dependencies the MCP tools need. Construct once at
// startup (same repo/generator/validator instances the REST server uses) and
// pass it into the tool functions.
type Toolset struct {
	HealthRepo storage.HealthRepository
	Generator  *remediation.Generator
	Validator  *remediation.Validator
	Logger     *zap.Logger
}

func NewToolset(repo storage.HealthRepository, gen *remediation.Generator, val *remediation.Validator, logger *zap.Logger) *Toolset {
	return &Toolset{HealthRepo: repo, Generator: gen, Validator: val, Logger: logger}
}

// GetTelemetryHealth queries real tenant health data instead of returning a
// static payload. Returns an error (not fabricated data) if ClickHouse isn't
// configured, matching the REST endpoint's 503 behavior.
func (t *Toolset) GetTelemetryHealth(ctx context.Context, tenantID string) (*HealthResponse, error) {
	if t.HealthRepo == nil {
		return nil, fmt.Errorf("health repository not configured — ClickHouse unavailable")
	}

	metrics, err := t.HealthRepo.QueryHealthMetrics(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("querying health metrics: %w", err)
	}

	issueType := metrics.RemediationIssue
	remediationYaml := ""
	validated := false
	if issueType != "" && t.Generator != nil {
		var genErr error
		remediationYaml, genErr = t.Generator.Generate(ctx, issueType)
		if genErr != nil {
			if t.Logger != nil {
				t.Logger.Error("failed to generate remediation yaml", zap.Error(genErr))
			}
		}
		if t.Validator != nil && remediationYaml != "" {
			validated, _ = t.Validator.Validate(ctx, remediationYaml)
		}
	}

	return &HealthResponse{
		HealthScore: metrics.CompositeScore,
		Metrics: MetricsPayload{
			Cardinality: MetricValue{Value: fmt.Sprintf("%d", metrics.CardinalityMax)},
			Orphans:     MetricValue{Value: fmt.Sprintf("%d", metrics.OrphanCount)},
			Coverage:    MetricValue{Value: fmt.Sprintf("%d", metrics.ActiveServices)},
		},
		Remediation: RemediationPayload{
			IssueType: issueType,
			Yaml:      remediationYaml,
			Validated: validated,
		},
		TenantId: tenantID,
		Version:  "v1.1.0-mcp",
	}, nil
}

// GenerateRemediation is unchanged in shape but now hangs off the injected
// Toolset instead of a package-level singleton, so it shares the same
// generator instance as the REST server (one source of truth).
func (t *Toolset) GenerateRemediation(ctx context.Context, issueType string) (string, error) {
	if t.Generator == nil {
		return "", fmt.Errorf("remediation generator not configured")
	}
	return t.Generator.Generate(ctx, issueType)
}
