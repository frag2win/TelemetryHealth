package cardinality

import (
	"sync"

	"github.com/axiomhq/hyperloglog"
	"go.uber.org/zap"
)

const (
	defaultMaxKeysPerService = 100
)

// Tracker tracks attribute cardinality per service and attribute key using HyperLogLog sketches.
type Tracker struct {
	mu             sync.Mutex
	maxMemoryBytes int64
	currentMemory  int64
	maxKeys        int

	// map of service -> attribute_key -> HLL sketch
	sketches         map[string]map[string]*hyperloglog.Sketch
	previousSketches map[string]map[string]*hyperloglog.Sketch
	
	logger *zap.Logger
}

// NewTracker creates a new Cardinality Tracker.
func NewTracker(maxMemoryBytes int64, maxKeys int, logger *zap.Logger) *Tracker {
	if maxKeys <= 0 {
		maxKeys = defaultMaxKeysPerService
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Tracker{
		maxMemoryBytes:   maxMemoryBytes,
		maxKeys:          maxKeys,
		sketches:         make(map[string]map[string]*hyperloglog.Sketch),
		previousSketches: make(map[string]map[string]*hyperloglog.Sketch),
		logger:           logger,
	}
}

// Observe records a value for a specific service and attribute key.
func (t *Tracker) Observe(service, attrKey, attrValue string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	svcMap, ok := t.sketches[service]
	if !ok {
		svcMap = make(map[string]*hyperloglog.Sketch)
		t.sketches[service] = svcMap
	}

	sketch, ok := svcMap[attrKey]
	if !ok {
		if len(svcMap) >= t.maxKeys {
			// Key-space explosion protection
			t.logger.Warn("Key-space explosion detected, dropping new key",
				zap.String("service", service),
				zap.String("key", attrKey),
				zap.Int("max_keys", t.maxKeys))
			return
		}

		// A precision 14 HLL sketch takes around 12-16KB. Let's assume 16KB.
		const sketchMem = 16384
		if t.maxMemoryBytes > 0 && t.currentMemory+sketchMem > t.maxMemoryBytes {
			t.logger.Warn("Memory limit reached for cardinality tracker", zap.Int64("limit", t.maxMemoryBytes))
			return
		}

		sketch = hyperloglog.New14()
		svcMap[attrKey] = sketch
		t.currentMemory += sketchMem
	}

	sketch.Insert([]byte(attrValue))
}

// Flush returns the current tracking state and resets it (for rolling window export).
// Note: The returned map and its underlying sketches should be treated as read-only.
// Inserting new values into the returned sketches will bypass memory accounting.
func (t *Tracker) Flush() map[string]map[string]*hyperloglog.Sketch {
	t.mu.Lock()
	defer t.mu.Unlock()

	current := t.sketches
	t.previousSketches = current
	t.sketches = make(map[string]map[string]*hyperloglog.Sketch)
	t.currentMemory = 0

	return current
}
