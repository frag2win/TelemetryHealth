package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// pagerdutyEvent is the PagerDuty Events API v2 payload.
type pagerdutyEvent struct {
	RoutingKey  string            `json:"routing_key"`
	EventAction string            `json:"event_action"`
	DedupKey    string            `json:"dedup_key"`
	Payload     pagerdutyPayload  `json:"payload"`
	Links       []pagerdutyLink   `json:"links,omitempty"`
}

type pagerdutyPayload struct {
	Summary   string            `json:"summary"`
	Severity  string            `json:"severity"`
	Source    string            `json:"source"`
	Timestamp string            `json:"timestamp"`
	CustomDetails map[string]interface{} `json:"custom_details,omitempty"`
}

type pagerdutyLink struct {
	Href string `json:"href"`
	Text string `json:"text"`
}

// PagerDutyBridge fires alerts to PagerDuty Events API v2 (PRD §8.6, §7, Improvement #9).
type PagerDutyBridge struct {
	mu          sync.Mutex
	logger      *zap.Logger
	routingKey  string
	cooldown    time.Duration
	lastFired   map[string]time.Time
	httpClient  *http.Client
	apiEndpoint string
}

func NewPagerDutyBridge(logger *zap.Logger, routingKey string) *PagerDutyBridge {
	return &PagerDutyBridge{
		logger:      logger,
		routingKey:  routingKey,
		cooldown:    15 * time.Minute,
		lastFired:   make(map[string]time.Time),
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		apiEndpoint: "https://events.pagerduty.com/v2/enqueue",
	}
}

// FireAlert sends a PagerDuty alert with all required fields (PRD §8.6).
// Implements 15-minute deduplication cooldown to prevent alert fatigue.
func (b *PagerDutyBridge) FireAlert(ctx context.Context, payload AlertPayload) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if last, exists := b.lastFired[payload.AlertID]; exists && now.Sub(last) < b.cooldown {
		b.logger.Debug("PagerDuty alert suppressed due to cooldown",
			zap.String("alert_id", payload.AlertID),
		)
		return nil
	}

	// Map severity
	pdSeverity := "info"
	switch payload.Severity {
	case SeverityCritical:
		pdSeverity = "critical"
	case SeverityWarning:
		pdSeverity = "warning"
	}

	event := pagerdutyEvent{
		RoutingKey:  b.routingKey,
		EventAction: "trigger",
		DedupKey:    payload.AlertID,
		Payload: pagerdutyPayload{
			Summary:   fmt.Sprintf("[TelemetryHealth] %s: Health score %.0f for %s", string(payload.Severity), payload.Score, payload.AffectedService),
			Severity:  pdSeverity,
			Source:    "telemetryhealth-control-plane",
			Timestamp: now.UTC().Format(time.RFC3339),
			CustomDetails: map[string]interface{}{
				"tenant_id":           payload.TenantID,
				"health_score":        payload.Score,
				"affected_service":    payload.AffectedService,
				"affected_attribute":  payload.AffectedAttribute,
				"contributing_signals": payload.ContributingSignals,
				"remediation_snippet": payload.RemediationSnippet,
			},
		},
		Links: []pagerdutyLink{
			{Href: payload.DashboardLink, Text: "TelemetryHealth Dashboard"},
		},
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("pagerduty: marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.apiEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("pagerduty: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("pagerduty: send alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("pagerduty: unexpected status %d", resp.StatusCode)
	}

	b.lastFired[payload.AlertID] = now
	b.logger.Info("PagerDuty alert fired",
		zap.String("alert_id", payload.AlertID),
		zap.String("affected_service", payload.AffectedService),
		zap.Float64("score", payload.Score),
	)
	return nil
}
