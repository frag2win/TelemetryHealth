package remediation

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// otelAllowedComponents is the component allowlist for dry-run validation (PRD §8.5).
// Blocks receiver/exporter types capable of filesystem or host access.
var otelAllowedComponents = map[string]bool{
	// Allowed processors (config manipulation only)
	"attributes":             true,
	"transform":              true,
	"probabilistic_sampler":  true,
	"tail_sampling":          true,
	"filter":                 true,
	"resource":               true,
	"batch":                  true,
	"memory_limiter":         true,
	// Allowed receivers (network-based, no FS access)
	"otlp":           true,
	"jaeger":         true,
	"zipkin":         true,
	// Blocked by default: filelog, hostmetrics, any exporter
}

// Validator runs generated remediation configs through structural validation (PRD §8.5 Phase 1).
// Phase 1: in-process YAML parsing + OTel component allowlist check.
// Phase 2 (M3): ephemeral containerized otelcol dry-run with sandbox constraints.
type Validator struct {
	logger *zap.Logger
}

func NewValidator(logger *zap.Logger) *Validator {
	return &Validator{logger: logger}
}

// Validate performs two-phase validation (PRD §8.5):
// 1. YAML syntax parsing.
// 2. OTel component structural check: only allowed components may appear.
// Returns (true, nil) if valid, (false, err) if invalid.
func (v *Validator) Validate(ctx context.Context, yamlConfig string) (bool, error) {
	v.logger.Info("Running validation (Phase 1: structural check)")

	// Phase 1a: YAML syntax check
	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlConfig), &parsed); err != nil {
		v.logger.Warn("YAML parsing failed for remediation config", zap.Error(err))
		return false, fmt.Errorf("invalid YAML syntax: %w", err)
	}

	// Phase 1b: OTel component allowlist check (PRD §8.5 — block filesystem-capable components)
	if err := v.checkAllowlist(parsed); err != nil {
		v.logger.Warn("Component allowlist violation in remediation config", zap.Error(err))
		return false, err
	}

	v.logger.Info("Shadow-collector validation passed (structural)")
	return true, nil
}

// checkAllowlist walks the parsed YAML and ensures only allowed OTel components are present.
func (v *Validator) checkAllowlist(parsed map[string]interface{}) error {
	// Check each top-level OTel config section: processors, receivers, exporters, connectors
	sections := []string{"processors", "receivers", "exporters", "connectors"}
	for _, section := range sections {
		sectionData, ok := parsed[section]
		if !ok {
			continue
		}
		components, ok := sectionData.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid config: %s must be a map", section)
		}
		for componentKey := range components {
			// Strip any name suffix (e.g., "attributes/remediation" → "attributes")
			componentType := strings.SplitN(componentKey, "/", 2)[0]
			if !otelAllowedComponents[componentType] {
				return fmt.Errorf("component not allowed in remediation config: %s/%s (blocked by security allowlist)", section, componentKey)
			}
		}
	}
	return nil
}
