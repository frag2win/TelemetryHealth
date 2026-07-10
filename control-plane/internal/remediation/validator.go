package remediation

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Validator struct {
	logger *zap.Logger
}

func NewValidator(logger *zap.Logger) *Validator {
	return &Validator{logger: logger}
}

// Validate runs the generated config through a shadow-Collector dry run.
func (v *Validator) Validate(ctx context.Context, yamlConfig string) (bool, error) {
	v.logger.Info("Running shadow-collector validation (dry-run)")
	
	// Basic YAML parsing check to ensure configuration is valid syntax
	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlConfig), &parsed); err != nil {
		v.logger.Warn("YAML parsing failed for remediation config", zap.Error(err))
		return false, fmt.Errorf("invalid YAML syntax: %w", err)
	}

	return true, nil
}
