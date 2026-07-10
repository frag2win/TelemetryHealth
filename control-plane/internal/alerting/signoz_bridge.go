package alerting

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

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

func (b *SigNozBridge) FireAlert(ctx context.Context, alertID string, score float64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if last, exists := b.lastFired[alertID]; exists && now.Sub(last) < b.cooldown {
		b.logger.Debug("Alert suppressed due to cooldown", zap.String("alertID", alertID))
		return nil
	}

	b.lastFired[alertID] = now
	b.logger.Info("Firing alert to SigNoz Alertmanager", zap.String("alertID", alertID), zap.Float64("score", score))
	return nil
}
