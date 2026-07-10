package telemetry

func Clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

// CalculateHealthScoreFromViolations computes the score from pre-computed violations.
func CalculateHealthScoreFromViolations(cardinalityViolation, orphanRate, coverageDrop float64) float64 {
	score := 100.0 - (0.20*cardinalityViolation + 0.30*orphanRate + 0.50*coverageDrop)
	if score < 0 {
		return 0
	}
	return score
}

// CalculateHealthScore computes the composite Telemetry Health Score from raw metrics.
// Weights: cardinality 20%, orphan 30%, coverage 50%
// Normalization: >1M cardinality = 100% violation, >1000 orphans = 100% violation, coverage active services < 10 drops health.
func CalculateHealthScore(cardinalityMax, orphanCount, activeServices uint64) float64 {
	cardViolation := Clamp(float64(cardinalityMax)/1_000_000.0) * 100
	orphanViolation := Clamp(float64(orphanCount)/1000.0) * 100
	coverageDrop := 0.0
	if activeServices < 10 {
		coverageDrop = (1.0 - float64(activeServices)/10.0) * 100
	}

	return CalculateHealthScoreFromViolations(cardViolation, orphanViolation, coverageDrop)
}
