package remediation

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type Generator struct {
	logger *zap.Logger
}

func NewGenerator(logger *zap.Logger) *Generator {
	return &Generator{logger: logger}
}

// Generate proposes an OTel config snippet to fix the issue.
func (g *Generator) Generate(ctx context.Context, issueType string) (string, error) {
	var yaml string
	switch issueType {
	case "cardinality_explosion":
		yaml = `
processors:
  attributes/remediation:
    actions:
      - key: "user_id"
        action: "delete"
`
	case "sampling_gap":
		yaml = `
processors:
  probabilistic_sampler/remediation:
    hash_seed: 22
    sampling_percentage: 100
`
	case "broken_trace_chain":
		yaml = `
processors:
  tail_sampling/repair:
    policies:
      [ { name: repair-chain, type: always_sample } ]
`
	case "coverage_gap":
		yaml = `
receivers:
  otlp/missing_service:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
`
	case "semantic_convention_violation":
		yaml = `
processors:
  transform/remediation:
    trace_statements:
      - context: span
        statements:
          - set(attributes["http.method"], attributes["http.request.method"])
`
	default:
		return "", fmt.Errorf("unknown issue type: %s", issueType)
	}

	return yaml, nil
}
