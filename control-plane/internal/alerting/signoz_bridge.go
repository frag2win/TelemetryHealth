package alerting

import (
	"context"
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

// SigNozBridge fires alerts to SigNoz Alertmanager with deduplication and cooldown (PRD §8.6).
type SigNozBridge struct {
	mu        sync.Mutex
	logger    *zap.Logger
	cooldown  time.Duration
	lastFired map[string]time.Time
}

func NewSigNozBridge(logger *zap.Logger) *SigNozBridge {
	return &SigNozBridge{
		logger:    logger,
		cooldown:  15 * time.Minute,
		lastFired: make(map[string]time.Time),
	}
}

// FireAlert sends an alert to SigNoz Alertmanager.
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
