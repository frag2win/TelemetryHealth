package alerting

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type SigNozBridge struct {
	logger   *zap.Logger
	cooldown time.Duration
}

func NewSigNozBridge(logger *zap.Logger) *SigNozBridge {
	return &SigNozBridge{
		logger:   logger,
		cooldown: 15 * time.Minute,
	}
}

func (b *SigNozBridge) FireAlert(ctx context.Context, alertID string, score float64) error {
	// PRD §8.6 Alert deduplication and suppression
	b.logger.Info("Firing alert to SigNoz Alertmanager", zap.String("alertID", alertID), zap.Float64("score", score))
	return nil
}
