package processor

import (
	"fmt"

	"github.com/frag2win/TelemetryHealth/processor/failopen"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type baseConsumer struct {
	cb *failopen.CircuitBreaker
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
		cb: failopen.NewCircuitBreaker(procCfg.CircuitBreakerLimit, procCfg.CircuitBreakerTimeout, logger),
	}, nil
}

