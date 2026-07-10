package telemetry

// TenantWeights holds per-tenant configurable health score weights (PRD §8.4).
// Defaults match the formula from PRD §8.4: cardinality 20%, orphan 30%, coverage 50%.
type TenantWeights struct {
	CardinalityWeight float64 // default 0.20
	OrphanWeight      float64 // default 0.30
	CoverageWeight    float64 // default 0.50
}

// DefaultWeights returns the PRD-specified default weights.
func DefaultWeights() TenantWeights {
	return TenantWeights{
		CardinalityWeight: 0.20,
		OrphanWeight:      0.30,
		CoverageWeight:    0.50,
	}
}

// ScoreScope defines at which granularity the health score is computed.
type ScoreScope string

const (
	ScopeService     ScoreScope = "service"
	ScopeEnvironment ScoreScope = "environment"
	ScopeOrg         ScoreScope = "org"
)

// Clamp ensures a value is in [0.0, 1.0].
func Clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

// CalculateHealthScoreFromViolations computes the score from pre-computed violations
// using the provided tenant weights (PRD §8.4 — configurable per tenant).
func CalculateHealthScoreFromViolations(cardinalityViolation, orphanRate, coverageDrop float64, weights TenantWeights) float64 {
	score := 100.0 - (weights.CardinalityWeight*100*cardinalityViolation + weights.OrphanWeight*100*orphanRate + weights.CoverageWeight*100*coverageDrop)
	if score < 0 {
		return 0
	}
	return score
}

// CalculateHealthScore computes the composite Telemetry Health Score from raw metrics
// using the provided tenant weights. Weights: cardinality 20%, orphan 30%, coverage 50% by default.
// Normalization: >1M cardinality = 100% violation, >1000 orphans = 100% violation, coverage active services <10 drops health.
func CalculateHealthScore(cardinalityMax, orphanCount, activeServices uint64, weights TenantWeights) float64 {
	cardViolation := Clamp(float64(cardinalityMax) / 1_000_000.0)
	orphanViolation := Clamp(float64(orphanCount) / 1000.0)
	coverageDrop := 0.0
	if activeServices < 10 {
		coverageDrop = (1.0 - float64(activeServices)/10.0)
	}

	return CalculateHealthScoreFromViolations(cardViolation, orphanViolation, coverageDrop, weights)
}
