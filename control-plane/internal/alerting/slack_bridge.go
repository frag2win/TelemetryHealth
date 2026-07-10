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

// slackBlock is a minimal Slack Block Kit structure for an alert message.
type slackMessage struct {
	Text   string       `json:"text"`
	Blocks []slackBlock `json:"blocks"`
}

type slackBlock struct {
	Type string        `json:"type"`
	Text *slackText    `json:"text,omitempty"`
}

type slackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SlackBridge fires alerts to a Slack channel via the Slack Web API (PRD §8.6, §7, Improvement #9).
type SlackBridge struct {
	mu         sync.Mutex
	logger     *zap.Logger
	webhookURL string
	cooldown   time.Duration
	lastFired  map[string]time.Time
	httpClient *http.Client
}

func NewSlackBridge(logger *zap.Logger, webhookURL string) *SlackBridge {
	return &SlackBridge{
		logger:     logger,
		webhookURL: webhookURL,
		cooldown:   15 * time.Minute,
		lastFired:  make(map[string]time.Time),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// FireAlert sends a Slack Block Kit message with all required fields (PRD §8.6).
// Implements 15-minute deduplication cooldown to prevent alert fatigue.
func (b *SlackBridge) FireAlert(ctx context.Context, payload AlertPayload) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if last, exists := b.lastFired[payload.AlertID]; exists && now.Sub(last) < b.cooldown {
		b.logger.Debug("Slack alert suppressed due to cooldown",
			zap.String("alert_id", payload.AlertID),
		)
		return nil
	}

	icon := "🔴"
	switch payload.Severity {
	case SeverityWarning:
		icon = "🟡"
	case SeverityInfo:
		icon = "🟢"
	}

	// Build contributing signals text
	signalText := ""
	for k, v := range payload.ContributingSignals {
		signalText += fmt.Sprintf("• *%s:* %.2f\n", k, v)
	}

	bodyText := fmt.Sprintf(
		"%s *TelemetryHealth Alert* | Score: *%.0f* | Tenant: `%s`\n\n"+
			"*Service:* %s\n"+
			"*Attribute:* %s\n\n"+
			"*Contributing Signals:*\n%s\n"+
			"*Remediation:*\n```%s```\n\n"+
			"<%s|View in Dashboard>",
		icon,
		payload.Score,
		payload.TenantID,
		payload.AffectedService,
		payload.AffectedAttribute,
		signalText,
		payload.RemediationSnippet,
		payload.DashboardLink,
	)

	msg := slackMessage{
		Text: fmt.Sprintf("%s TelemetryHealth Alert for %s (score: %.0f)", icon, payload.AffectedService, payload.Score),
		Blocks: []slackBlock{
			{Type: "section", Text: &slackText{Type: "mrkdwn", Text: bodyText}},
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("slack: marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("slack: send alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack: unexpected status %d", resp.StatusCode)
	}

	b.lastFired[payload.AlertID] = now
	b.logger.Info("Slack alert fired",
		zap.String("alert_id", payload.AlertID),
		zap.String("affected_service", payload.AffectedService),
		zap.Float64("score", payload.Score),
	)
	return nil
}
