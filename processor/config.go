package processor

import (
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
)

// Config represents the configuration for telemetryhealth processor
type Config struct {
	// ControlPlaneEndpoint is the gRPC endpoint for the TelemetryHealth control plane (EXP2 destination)
	ControlPlaneEndpoint string `mapstructure:"control_plane_endpoint"`
	// TenantID is the UUID for the tenant
	TenantID string `mapstructure:"tenant_id"`
	// MaxMemoryBytes defines the total memory bound for the local cardinality LRU tracker. Default 256MB.
	MaxMemoryBytes int64 `mapstructure:"max_memory_bytes"`
	// LatenessWindow defines the out-of-order allowance for trace chain buffering. Default 30s.
	LatenessWindow string `mapstructure:"lateness_window"`
	// CircuitBreakerLimit is the number of failures before opening the circuit breaker.
	CircuitBreakerLimit int64 `mapstructure:"circuit_breaker_limit"`
	// CircuitBreakerTimeout is the time to wait before half-opening the circuit breaker.
	CircuitBreakerTimeout time.Duration `mapstructure:"circuit_breaker_timeout"`
}

var _ component.Config = (*Config)(nil)

func createDefaultConfig() component.Config {
	return &Config{
		ControlPlaneEndpoint:  "",
		TenantID:              "",
		MaxMemoryBytes:        256 * 1024 * 1024,
		LatenessWindow:        "30s",
		CircuitBreakerLimit:   5,
		CircuitBreakerTimeout: 30 * time.Second,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if _, err := time.ParseDuration(c.LatenessWindow); err != nil {
		return fmt.Errorf("invalid lateness_window %q: %w", c.LatenessWindow, err)
	}
	return nil
}
