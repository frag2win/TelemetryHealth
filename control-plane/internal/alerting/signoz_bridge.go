package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AlertSeverity maps health scores to PagerDuty/Slack severity levels (PRD §8.6).
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityWarning  AlertSeverity = "warning"
	SeverityInfo     AlertSeverity = "info"
)

// AlertPayload contains all required fields per PRD §8.6:
// "Alerts must include: current score, contributing signals, affected service/attribute,
//  remediation snippet, and a link to the drilldown dashboard."
type AlertPayload struct {
	AlertID              string
	TenantID             string
	Score                float64
	Severity             AlertSeverity
	AffectedService      string
	AffectedAttribute    string
	ContributingSignals  map[string]float64 // signal_name → value
	RemediationSnippet   string
	DashboardLink        string
	FiredAt              time.Time
}

// SeverityFromScore returns the alert severity based on the health score.
func SeverityFromScore(score float64) AlertSeverity {
	switch {
	case score < 60:
		return SeverityCritical
	case score < 80:
		return SeverityWarning
	default:
		return SeverityInfo
	}
}

// AlertBridge is the interface implemented by all alerting integrations.
type AlertBridge interface {
	FireAlert(ctx context.Context, payload AlertPayload) error
}

type alertmanagerAlert struct {
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	GeneratorURL string            `json:"generatorURL,omitempty"`
}

// SigNozBridge fires alerts to SigNoz Alertmanager with deduplication and cooldown (PRD §8.6).
type SigNozBridge struct {
	mu          sync.Mutex
	logger      *zap.Logger
	cooldown    time.Duration
	lastFired   map[string]time.Time
	httpClient  *http.Client
	endpointURL string
}

func NewSigNozBridge(logger *zap.Logger) *SigNozBridge {
	url := os.Getenv("SIGNOZ_ALERTMANAGER_URL")
	if url == "" {
		url = "http://localhost:9093/api/v2/alerts"
	}
	return &SigNozBridge{
		logger:      logger,
		cooldown:    15 * time.Minute,
		lastFired:   make(map[string]time.Time),
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		endpointURL: url,
	}
}

// FireAlert sends an alert to SigNoz Alertmanager via HTTP POST (PRD §8.6).
// Implements deduplication: identical alert_id will not re-fire within the cooldown window (PRD §8.6).
func (b *SigNozBridge) FireAlert(ctx context.Context, payload AlertPayload) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if last, exists := b.lastFired[payload.AlertID]; exists && now.Sub(last) < b.cooldown {
		b.logger.Debug("Alert suppressed due to cooldown",
			zap.String("alert_id", payload.AlertID),
			zap.Duration("remaining", b.cooldown-now.Sub(last)),
		)
		return nil
	}

	// Construct Prometheus Alertmanager standard alerts array payload
	alerts := []alertmanagerAlert{
		{
			Labels: map[string]string{
				"alertname":        "TelemetryPipelineHealthDrop",
				"severity":         string(payload.Severity),
				"tenant_id":        payload.TenantID,
				"service_name":     payload.AffectedService,
				"attribute_key":    payload.AffectedAttribute,
				"alert_id":         payload.AlertID,
			},
			Annotations: map[string]string{
				"summary":        fmt.Sprintf("Telemetry health score dropped to %.1f for service %s", payload.Score, payload.AffectedService),
				"description":    fmt.Sprintf("The composite health score of %s is %.1f. Affected attribute: %s.", payload.AffectedService, payload.Score, payload.AffectedAttribute),
				"remediation":    payload.RemediationSnippet,
				"dashboard_link": payload.DashboardLink,
			},
			GeneratorURL: payload.DashboardLink,
		},
	}

	body, err := json.Marshal(alerts)
	if err != nil {
		return fmt.Errorf("signoz alertmanager: marshal alerts: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpointURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("signoz alertmanager: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("signoz alertmanager: send alert request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("signoz alertmanager: unexpected response status %d", resp.StatusCode)
	}

	b.lastFired[payload.AlertID] = now
	b.logger.Info("Firing alert to SigNoz Alertmanager",
		zap.String("alert_id", payload.AlertID),
		zap.String("tenant_id", payload.TenantID),
		zap.Float64("score", payload.Score),
		zap.String("severity", string(payload.Severity)),
		zap.String("affected_service", payload.AffectedService),
		zap.String("dashboard_link", payload.DashboardLink),
	)
	return nil
}
