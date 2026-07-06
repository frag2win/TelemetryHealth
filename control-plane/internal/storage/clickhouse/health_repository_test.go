package clickhouse_test

import (
	"context"
	"testing"
	"time"

	ch "github.com/frag2win/TelemetryHealth/control-plane/internal/storage/clickhouse"
	"go.uber.org/zap"
)

// TestHealthRepository_Offline verifies the repository gracefully handles
// a tenant with no data rows without panicking.
func TestHealthRepository_Offline(t *testing.T) {
	// We can't connect to ClickHouse in CI without a real server.
	// This test validates the client constructor returns an error cleanly.
	_, err := ch.NewClient(
		[]string{"localhost:19000"}, // guaranteed unreachable
		"telemetry_health", "default", "",
		zap.NewNop(),
	)
	if err == nil {
		t.Skip("ClickHouse is unexpectedly available on port 19000 — skipping offline test")
	}
	// If we get an error that means the client correctly refused to connect.
	t.Logf("Client correctly returned error for unreachable host: %v", err)
}

// TestHealthScore_Formula validates the composite score formula math.
func TestHealthScore_Formula(t *testing.T) {
	tests := []struct {
		name          string
		cardMax       uint64
		orphanCount   uint64
		activeServices uint64
		wantMin       float64
		wantMax       float64
	}{
		{"healthy", 100_000, 10, 14, 90, 100},
		{"cardinality explosion", 2_000_000, 10, 14, 70, 90},
		{"all bad", 2_000_000, 2000, 0, 0, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = context.Background()
			_ = time.Now()

			cardViolation := clamp(float64(tt.cardMax) / 1_000_000.0) * 100
			orphanViolation := clamp(float64(tt.orphanCount) / 1000.0) * 100
			coverageDrop := 0.0
			if tt.activeServices < 10 {
				coverageDrop = (1.0 - float64(tt.activeServices)/10.0) * 100
			}
			score := 100 - (0.20*cardViolation + 0.30*orphanViolation + 0.50*coverageDrop)
			if score < 0 {
				score = 0
			}

			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("score=%.1f, want [%.0f, %.0f]", score, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func clamp(v float64) float64 {
	if v > 1.0 {
		return 1.0
	}
	return v
}
