# TelemetryHealth Connector

This connector handles cross-signal telemetry pipeline health correlation between metrics, logs, and traces (PRD §5.1, §7).

## Component Responsibilities

1. **Cross-Signal Correlation**: Correlate metrics and logs with traces to detect silent coverage gaps or sampling paradox anomalies.
2. **State Merging**: Approximates and aggregates signal metrics before fanning out to the control plane.
3. **Fail-Open Integration**: Leverages the shared fail-open circuit breaker core (`processor/failopen/`) to ensure no data is dropped on downstream connection failure.
