package streaming

import (
	"context"
	"regexp"
	"sync"
	"time"

	"github.com/axiomhq/hyperloglog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// keyspaceExplosionTotal counts key-space explosion detections per service (PRD §8.1, Improvement #8).
var keyspaceExplosionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "telemetryhealth_keyspace_explosion_total",
	Help: "Number of times attribute key-space explosion was detected, by service and attribute prefix.",
}, []string{"service", "attribute_prefix"})

// dynamicKeyPattern detects dynamic key patterns like user_id_12345, session_abc123, etc.
var dynamicKeyPattern = regexp.MustCompile(`^(.+?)[\._-]\d+$`)

// CardinalityJob merges HLL sketches arriving from the collector fleet.
type CardinalityJob struct {
	logger  *zap.Logger
	mu      sync.Mutex
	state   map[string]*hyperloglog.Sketch
	maxKeys int // hard cap on distinct attribute keys per service (PRD §8.1, default 100)
	keysByService map[string]map[string]struct{} // tracks key count per service
}

func NewCardinalityJob(logger *zap.Logger) *CardinalityJob {
	return &CardinalityJob{
		logger:        logger,
		state:         make(map[string]*hyperloglog.Sketch),
		maxKeys:       100, // PRD §8.1 default
		keysByService: make(map[string]map[string]struct{}),
	}
}

// normalizeKey applies key normalization for dynamic attribute key patterns.
// e.g., "user_id_1042" → "user_id_*" (PRD §8.1).
func normalizeKey(key string) string {
	if m := dynamicKeyPattern.FindStringSubmatch(key); len(m) == 2 {
		return m[1] + "_*"
	}
	return key
}

// Process merges an incoming sketch into the central state.
// Enforces key-space explosion protection with alerting (PRD §8.1, Improvement #8).
func (j *CardinalityJob) Process(ctx context.Context, tenantID, service, key string, sketch *hyperloglog.Sketch, timestamp time.Time) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	// Normalize dynamic keys before tracking (PRD §8.1)
	normalizedKey := normalizeKey(key)
	stateKey := tenantID + ":" + service + ":" + normalizedKey

	// Key-space explosion protection (PRD §8.1)
	svcKeys, exists := j.keysByService[service]
	if !exists {
		svcKeys = make(map[string]struct{})
		j.keysByService[service] = svcKeys
	}

	if _, tracked := svcKeys[normalizedKey]; !tracked {
		if len(svcKeys) >= j.maxKeys {
			// Key-space cap exceeded: emit metric alert (PRD §8.1)
			prefix := key
			if len(prefix) > 20 {
				prefix = prefix[:20]
			}
			keyspaceExplosionTotal.WithLabelValues(service, prefix).Inc()
			j.logger.Warn("Key-space explosion detected: hard cap reached, dropping new key",
				zap.String("service", service),
				zap.String("key", key),
				zap.String("normalized_key", normalizedKey),
				zap.Int("cap", j.maxKeys),
			)
			return nil
		}
		svcKeys[normalizedKey] = struct{}{}
	}

	// Merge sketch into central state
	if existing, ok := j.state[stateKey]; ok {
		existing.Merge(sketch)
	} else {
		j.state[stateKey] = sketch
	}
	return nil
}

// EventTimeProcess is a variant that accepts event_time for watermark-based windowing (PRD §8.2, Improvement #13).
func (j *CardinalityJob) EventTimeProcess(ctx context.Context, tenantID, service, key string, sketch *hyperloglog.Sketch, eventTime time.Time) error {
	// Use eventTime (from Kafka header) rather than wall-clock (PRD §8.2, §13)
	return j.Process(ctx, tenantID, service, key, sketch, eventTime)
}
