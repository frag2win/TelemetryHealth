package processor

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/frag2win/TelemetryHealth/processor/failopen"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

const healthExportQueueSize = 1000

// healthExportDroppedTotal counts how many health signal events were dropped
// because the bounded EXP2 queue was full (PRD §10 Reliability).
var healthExportDroppedTotal atomic.Int64

// baseConsumer is the shared foundation for all three signal consumers.
// It holds the circuit breaker and a bounded health-signal export channel.
// The healthExportCh is a drop-on-overflow channel — it MUST NOT block
// the primary pipeline (EXP1 path). PRD §10 Reliability.
type baseConsumer struct {
	cb             *failopen.CircuitBreaker
	healthExportCh chan HealthSignal
	logger         *zap.Logger
}

// HealthSignal carries structural health telemetry to be shipped to the control plane.
// These are extracted BEFORE any sampling decisions are applied (PRD §8.2).
type HealthSignal struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	ServiceName  string
	TenantID     string
}

func newBaseConsumer(cfg component.Config, logger *zap.Logger) (baseConsumer, error) {
	if cfg == nil {
		return baseConsumer{}, fmt.Errorf("config cannot be nil")
	}
	procCfg, ok := cfg.(*Config)
	if !ok {
		return baseConsumer{}, fmt.Errorf("invalid config type: expected *Config, got %T", cfg)
	}
	return baseConsumer{
		cb:             failopen.NewCircuitBreaker(procCfg.CircuitBreakerLimit, procCfg.CircuitBreakerTimeout, logger),
		healthExportCh: make(chan HealthSignal, healthExportQueueSize),
		logger:         logger,
	}, nil
}

// EmitHealthSignal enqueues a health signal for export to EXP2.
// If the queue is full, the signal is DROPPED immediately (never blocks the primary pipeline).
// This satisfies PRD §10: bounded EXP2 drop-on-overflow policy.
func (b *baseConsumer) EmitHealthSignal(ctx context.Context, sig HealthSignal) {
	select {
	case b.healthExportCh <- sig:
		// Signal enqueued successfully
	default:
		// Queue full — drop and count (never block)
		dropped := healthExportDroppedTotal.Add(1)
		if dropped%100 == 1 {
			b.logger.Warn("Health export queue full, dropping signal",
				zap.Int64("total_dropped", dropped),
			)
		}
	}
}

// DrainHealthSignals returns all buffered health signals for batch export.
func (b *baseConsumer) DrainHealthSignals() []HealthSignal {
	var signals []HealthSignal
	for {
		select {
		case sig := <-b.healthExportCh:
			signals = append(signals, sig)
		default:
			return signals
		}
	}
}
