package remediation

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"
)

var testLogger = zap.NewNop()

// ── Validator tests ───────────────────────────────────────────────────────────

func TestValidator_ValidYAML_Passes(t *testing.T) {
	v := NewValidator(testLogger)
	yaml := `
processors:
  attributes/remediation:
    actions:
      - key: user_id
        action: hash
`
	ok, err := v.Validate(context.Background(), yaml)
	if !ok || err != nil {
		t.Errorf("expected valid config to pass, got ok=%v err=%v", ok, err)
	}
}

func TestValidator_InvalidYAML_Fails(t *testing.T) {
	v := NewValidator(testLogger)
	badYaml := `{this is: [not valid yaml`
	ok, err := v.Validate(context.Background(), badYaml)
	if ok {
		t.Error("expected invalid YAML to fail validation")
	}
	if err == nil {
		t.Error("expected non-nil error for invalid YAML")
	}
}

func TestValidator_EmptyYAML_Passes(t *testing.T) {
	v := NewValidator(testLogger)
	ok, err := v.Validate(context.Background(), "")
	if !ok || err != nil {
		t.Errorf("expected empty YAML to pass (no components), got ok=%v err=%v", ok, err)
	}
}

func TestValidator_BlockedComponent_Fails(t *testing.T) {
	v := NewValidator(testLogger)
	// filelog is blocked (FS access capable).
	yaml := `
receivers:
  filelog:
    include:
      - /var/log/*.log
`
	ok, err := v.Validate(context.Background(), yaml)
	if ok {
		t.Error("expected blocked component 'filelog' to fail validation")
	}
	if err == nil || !strings.Contains(err.Error(), "filelog") {
		t.Errorf("expected error mentioning 'filelog', got: %v", err)
	}
}

func TestValidator_BlockedExporter_Fails(t *testing.T) {
	v := NewValidator(testLogger)
	// Exporters are not on the allowlist by design.
	yaml := `
exporters:
  otlphttp:
    endpoint: http://malicious.host:4317
`
	ok, err := v.Validate(context.Background(), yaml)
	if ok {
		t.Error("expected blocked exporter to fail validation")
	}
	if err == nil {
		t.Error("expected non-nil error for blocked exporter")
	}
}

func TestValidator_MultipleAllowedComponents_Passes(t *testing.T) {
	v := NewValidator(testLogger)
	yaml := `
processors:
  batch:
    timeout: 200ms
  memory_limiter:
    limit_mib: 512
  filter/errors:
    error_mode: ignore
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
`
	ok, err := v.Validate(context.Background(), yaml)
	if !ok || err != nil {
		t.Errorf("expected multi-component valid config to pass, got ok=%v err=%v", ok, err)
	}
}

func TestValidator_ComponentWithSuffix_Passes(t *testing.T) {
	v := NewValidator(testLogger)
	// "attributes/remediation" should pass — type "attributes" is allowed.
	yaml := `
processors:
  attributes/remediation:
    actions:
      - key: credit_card
        action: delete
`
	ok, err := v.Validate(context.Background(), yaml)
	if !ok || err != nil {
		t.Errorf("expected suffixed allowed component to pass, got ok=%v err=%v", ok, err)
	}
}

func TestValidator_InvalidSectionType_Fails(t *testing.T) {
	v := NewValidator(testLogger)
	// processors must be a map, not a list.
	yaml := `
processors:
  - attributes
  - batch
`
	ok, err := v.Validate(context.Background(), yaml)
	if ok {
		t.Error("expected invalid section structure (list instead of map) to fail")
	}
	if err == nil {
		t.Error("expected non-nil error for invalid section type")
	}
}

// ── Generator tests ───────────────────────────────────────────────────────────

func TestGenerator_Cardinality_ProducesYAML(t *testing.T) {
	g := NewGenerator(testLogger)
	yaml, err := g.Generate(context.Background(), "High Cardinality (user_id)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if yaml == "" {
		t.Error("expected non-empty YAML for cardinality issue")
	}
	if !strings.Contains(yaml, "user_id") {
		t.Errorf("expected generated YAML to contain 'user_id', got:\n%s", yaml)
	}
}

func TestGenerator_Sampling_ProducesYAML(t *testing.T) {
	g := NewGenerator(testLogger)
	yaml, err := g.Generate(context.Background(), "sampling drift detected")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if yaml == "" {
		t.Error("expected non-empty YAML for sampling issue")
	}
}

func TestGenerator_Coverage_ProducesYAML(t *testing.T) {
	g := NewGenerator(testLogger)
	yaml, err := g.Generate(context.Background(), "coverage gap on inventory-worker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if yaml == "" {
		t.Error("expected non-empty YAML for coverage issue")
	}
}

func TestGenerator_BrokenTraceChain_ProducesYAML(t *testing.T) {
	g := NewGenerator(testLogger)
	yaml, err := g.Generate(context.Background(), "broken_trace detected on payments-api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if yaml == "" {
		t.Error("expected non-empty YAML for broken trace chain issue")
	}
}

func TestGenerator_OrphanAlias_ProducesYAML(t *testing.T) {
	g := NewGenerator(testLogger)
	yaml, err := g.Generate(context.Background(), "orphan spans detected")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if yaml == "" {
		t.Error("expected non-empty YAML for orphan issue")
	}
}

func TestGenerator_UnknownIssueType_ReturnsError(t *testing.T) {
	g := NewGenerator(testLogger)
	yaml, err := g.Generate(context.Background(), "flux capacitor overload")
	if err == nil {
		t.Error("expected error for unknown issue type, got nil")
	}
	if yaml != "" {
		t.Errorf("expected empty YAML for unknown issue, got: %s", yaml)
	}
}

func TestGenerator_EmptyIssueType_ReturnsError(t *testing.T) {
	g := NewGenerator(testLogger)
	yaml, err := g.Generate(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty issue type, got nil")
	}
	_ = yaml
}

func TestGenerator_Cardinality_OutputPassesValidator(t *testing.T) {
	g := NewGenerator(testLogger)
	v := NewValidator(testLogger)

	yaml, err := g.Generate(context.Background(), "High Cardinality (session_id)")
	if err != nil {
		t.Fatalf("generator error: %v", err)
	}

	ok, err := v.Validate(context.Background(), yaml)
	if !ok || err != nil {
		t.Errorf("generated cardinality YAML failed validation: ok=%v err=%v\nYAML:\n%s", ok, err, yaml)
	}
}
