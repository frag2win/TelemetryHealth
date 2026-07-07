package processor

import (
	"github.com/frag2win/TelemetryHealth/processor/failopen"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type baseConsumer struct {
	cb *failopen.CircuitBreaker
}

func newBaseConsumer(cfg component.Config, logger *zap.Logger) baseConsumer {
	procCfg := cfg.(*Config)
	return baseConsumer{
		cb: failopen.NewCircuitBreaker(procCfg.CircuitBreakerLimit, procCfg.CircuitBreakerTimeout, logger),
	}
}
