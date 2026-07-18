package alerting

import (
	"context"
	"os"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage"
	"go.uber.org/zap"
)

// TelemetryPoller periodically fetches health scores and triggers alerts.
type TelemetryPoller struct {
	logger       *zap.Logger
	healthRepo   storage.HealthRepository
	bridge       *SigNozBridge
	interval     time.Duration
	threshold    float64
	targetTenant string
}

// NewTelemetryPoller creates a new poller.
func NewTelemetryPoller(logger *zap.Logger, healthRepo storage.HealthRepository, bridge *SigNozBridge, interval time.Duration, threshold float64, targetTenant string) *TelemetryPoller {
	return &TelemetryPoller{
		logger:       logger,
		healthRepo:   healthRepo,
		bridge:       bridge,
		interval:     interval,
		threshold:    threshold,
		targetTenant: targetTenant,
	}
}

// Start begins the polling loop in the background.
func (p *TelemetryPoller) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				p.logger.Info("Stopping telemetry poller")
				return
			case <-ticker.C:
				p.poll(ctx)
			}
		}
	}()
}

func (p *TelemetryPoller) poll(ctx context.Context) {
	metrics, err := p.healthRepo.QueryHealthMetrics(ctx, p.targetTenant)
	if err != nil {
		p.logger.Error("Failed to query health metrics", zap.Error(err))
		return
	}

	if metrics.CompositeScore < p.threshold {
		p.logger.Warn("Health score below threshold, triggering alert", zap.Float64("score", metrics.CompositeScore))

		// Identify worst signal based on raw counts roughly
		worstSignal := "cardinality"
		if metrics.OrphanCount > 100 {
			worstSignal = "trace-chains"
		} else if metrics.ActiveServices == 0 {
			worstSignal = "coverage"
		}

		payload := AlertPayload{
			AlertID:           "health-drop-" + p.targetTenant,
			TenantID:          p.targetTenant,
			Score:             metrics.CompositeScore,
			Severity:          SeverityFromScore(metrics.CompositeScore),
			AffectedService:   "Multiple", // Default or aggregate
			AffectedAttribute: "",
			ContributingSignals: map[string]float64{
				"cardinality_max": float64(metrics.CardinalityMax),
				"orphan_count":    float64(metrics.OrphanCount),
				"active_services": float64(metrics.ActiveServices),
			},
			RemediationSnippet: "See dashboard for auto-generated remediation snippet",
			DashboardLink:      func() string {
				if link := os.Getenv("DASHBOARD_URL"); link != "" {
					return link
				}
				return "http://localhost:5173"
			}(),
			FiredAt:            time.Now(),
		}

		// Update affected service if we know which one is worst
		if worstSignal == "coverage" {
			payload.AffectedService = "unknown-dropped-service"
		}

		if err := p.bridge.FireAlert(ctx, payload); err != nil {
			p.logger.Error("Failed to fire alert to SigNoz", zap.Error(err))
		}
	}
}
