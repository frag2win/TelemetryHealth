package remediation

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"strings"
	"text/template"
	"time"

	"go.uber.org/zap"
)

//go:embed templates/*.yaml.tmpl
var templatesFS embed.FS

type Generator struct {
	logger *zap.Logger
}

func NewGenerator(logger *zap.Logger) *Generator {
	return &Generator{logger: logger}
}

// Generate proposes an OTel config snippet to fix the issue.
// Uses embedded templates and text/template for dynamic variable substitution (PRD §8.5, Improvement #12).
func (g *Generator) Generate(ctx context.Context, issueType string) (string, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("remediation generator context cancelled or timed out: %w", err)
	}

	data := map[string]interface{}{
		"AttributeKey":       "user_id",
		"SamplingPercentage": 100,
		"OTLPEndpoint":       "0.0.0.0:4317",
		"ServiceName":        "missing_service",
	}

	// Dynamic parsing logic to support variable substitution from issueType strings
	lowerIssue := strings.ToLower(issueType)
	if strings.Contains(lowerIssue, "cardinality") {
		// Check if we have format "High Cardinality (user_id on checkout_service)"
		if start := strings.Index(issueType, "("); start != -1 {
			end := strings.Index(issueType, " on")
			if end == -1 {
				end = strings.Index(issueType, ")")
			}
			if end != -1 && end > start {
				data["AttributeKey"] = strings.TrimSpace(issueType[start+1 : end])
			}
		}
	} else if strings.Contains(lowerIssue, "coverage") || strings.Contains(lowerIssue, "silent") {
		// Check if we have format "silent 14m" or service name
		// If service name is present, substitute it
		if strings.Contains(issueType, "on ") {
			parts := strings.Split(issueType, "on ")
			if len(parts) > 1 {
				data["ServiceName"] = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(issueType, "service") {
			data["ServiceName"] = "missing_service"
		}
	}

	var templateName string
	switch {
	case strings.Contains(lowerIssue, "cardinality"):
		templateName = "templates/cardinality_redaction.yaml.tmpl"
	case strings.Contains(lowerIssue, "sampling"):
		templateName = "templates/sampling_adjustment.yaml.tmpl"
	case strings.Contains(lowerIssue, "broken_trace") || strings.Contains(lowerIssue, "orphan"):
		// Use inline template for broken trace chain, or define a tmpl file.
		// Since trace chain job doesn't have a template file listed, we can use an inline tmpl or define a new tmpl file.
		// Let's fallback to sampling/filtering template.
		return `processors:
  tail_sampling/repair:
    policies:
      [ { name: repair-chain, type: always_sample } ]`, nil
	case strings.Contains(lowerIssue, "coverage") || strings.Contains(lowerIssue, "silent") || strings.Contains(lowerIssue, "gap"):
		templateName = "templates/coverage_enable.yaml.tmpl"
	default:
		return "", fmt.Errorf("unknown issue type: %s", issueType)
	}

	tmplBytes, err := templatesFS.ReadFile(templateName)
	if err != nil {
		return "", fmt.Errorf("read embedded template: %w", err)
	}

	tmpl, err := template.New(templateName).Parse(string(tmplBytes))
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}
