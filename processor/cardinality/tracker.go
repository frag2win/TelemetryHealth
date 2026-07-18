package cardinality

import (
	"os"
	"strconv"
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
	precision      int
	sketchMem      int64

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

	precisionVal := 14
	if pStr := os.Getenv("HLL_PRECISION"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p >= 10 && p <= 16 {
			precisionVal = p
		}
	}
	memVal := int64(1 << precisionVal)
	if mStr := os.Getenv("HLL_SKETCH_MEM"); mStr != "" {
		if m, err := strconv.ParseInt(mStr, 10, 64); err == nil && m > 0 {
			memVal = m
		}
	}

	return &Tracker{
		maxMemoryBytes:   maxMemoryBytes,
		maxKeys:          maxKeys,
		precision:        precisionVal,
		sketchMem:        memVal,
		sketches:         make(map[string]map[string]*hyperloglog.Sketch),
		previousSketches: make(map[string]map[string]*hyperloglog.Sketch),
		logger:           logger,
	}
}

// Observe records a value for a specific service and attribute key.
// Returns true if the key is allowed/tracked, and false if dropped due to limit/memory.
func (t *Tracker) Observe(service, attrKey, attrValue string) bool {
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
			return false
		}

		if t.maxMemoryBytes > 0 && t.currentMemory+t.sketchMem > t.maxMemoryBytes {
			t.logger.Warn("Memory limit reached for cardinality tracker", zap.Int64("limit", t.maxMemoryBytes))
			return false
		}

		var err error
		sketch, err = hyperloglog.NewSketch(uint8(t.precision), true)
		if err != nil {
			sketch = hyperloglog.New14()
		}

		svcMap[attrKey] = sketch
		t.currentMemory += t.sketchMem
	}

	sketch.Insert([]byte(attrValue))
	return true
}

// Flush returns a deep copy of the current tracking state and resets it (for rolling window export).
func (t *Tracker) Flush() map[string]map[string]*hyperloglog.Sketch {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := make(map[string]map[string]*hyperloglog.Sketch, len(t.sketches))
	for svc, attrs := range t.sketches {
		svcCopy := make(map[string]*hyperloglog.Sketch, len(attrs))
		for k, s := range attrs {
			svcCopy[k] = s.Clone()
		}
		result[svc] = svcCopy
	}

	t.previousSketches = t.sketches
	t.sketches = make(map[string]map[string]*hyperloglog.Sketch)
	t.currentMemory = 0

	return result
}

// PreviousSketches returns the snapshot of sketches from the previous window.
func (t *Tracker) PreviousSketches() map[string]map[string]*hyperloglog.Sketch {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.previousSketches
}
