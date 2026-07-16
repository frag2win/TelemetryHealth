# Adding New Diagnostic Detectors

This guide explains how to introduce a new diagnostic engine (detector) into the custom OpenTelemetry pipeline branch.

## 1. The Async Fan-Out Branch

Detectors live off the async fan-out branch within the collector to ensure primary trace telemetry is never blocked by analytical workloads.

## 2. Hooking Core Logic in Processor Structures

1. **Locate the Detector Interface:**
   Navigate to `processor/detectors/detector.go` to review the base interface.
2. **Implement your Detector:**
   Create a new file in `processor/detectors/` (e.g., `processor/detectors/token_leak.go`).
   ```go
   func (t *TokenLeakDetector) Evaluate(trace *pdata.Traces) (models.RootCause, error) {
       // Traverse spans looking for llm.request.tokens anomalies
   }
   ```
3. **Register the Detector:**
   Open `processor/factory.go`. Inject your new detector into the async routing loop:
   ```go
   pipeline.RegisterDetector("TokenLeak", detectors.NewTokenLeakDetector())
   ```

## 3. Safe Fail-Open Constraints

Ensure your detector does not panic or block indefinitely.
- Always wrap evaluations in the built-in circuit breaker timeout context (default: 500ms).
- If your detector exceeds the evaluation budget, return a `nil` verdict to ensure the pipeline fails open and continues processing.

## 4. Mapping Endpoints

If your detector introduces a new `FailureType`, ensure:
1. It is added to `control-plane/pkg/models/domain.go`.
2. The REST API exposes any required configuration overrides (e.g., `POST /api/v1/tenant/{id}/config/token_leak_threshold`).
