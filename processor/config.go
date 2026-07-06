package processor

import (
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
}

var _ component.Config = (*Config)(nil)

func createDefaultConfig() component.Config {
	return &Config{
		ControlPlaneEndpoint: "",
		TenantID:             "",
		MaxMemoryBytes:       256 * 1024 * 1024,
		LatenessWindow:       "30s",
	}
}
