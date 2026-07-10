# Runbook — Processor Circuit Breaker Tripped

## Symptom
The OTel Collector log displays:
`traces consumer execution failed, failing open: circuit breaker open`
Metrics show `telemetryhealth_health_export_dropped_total` is increasing, and health status indicators are offline/stale.

## Troubleshooting Steps

1. **Check Collector Memory and Latency**
   Verify if the collector instance is under heavy resource pressure (Memory limit near 256MB, CPU near 5%).
   
2. **Verify Control Plane Connectivity**
   Ensure the ingest gateway endpoint is reachable from the collector instance (port 4317 or gateway REST port).

## Mitigation

1. **Restart Collector**
   Breaker is self-healing, but if a stuck state occurs or memory limits were exceeded, restart the daemon.
   ```bash
   kubectl rollout restart daemonset otel-collector
   ```

2. **Adjust Circuit Breaker Limits**
   Increase limits in the configuration if necessary:
   ```yaml
   processors:
     telemetryhealth:
       circuit_breaker_limit: 50
       circuit_breaker_timeout: 60s
   ```
