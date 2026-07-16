# SigNoz Dashboard & Alertmanager Integration

This guide describes mapping custom OpenTelemetry metrics to SigNoz dashboards and establishing Alertmanager hooks.

## 1. Custom OTel Metric Mapping

The Control Plane stream worker emits rollup metrics. We specifically track `telemetryhealth_agent_health_score`.

1. Ensure the stream daemon correctly aggregates the gauge:
   ```go
   meter.Float64Gauge("telemetryhealth_agent_health_score",
       metric.WithDescription("Composite health score of AI Agent traces"),
   )
   ```
2. In SigNoz, this will be ingested as a standard Prometheus metric.

## 2. Managing Dashboard JSON States

SigNoz dashboards for TelemetryHealth are maintained as version-controlled JSON payloads to prevent configuration drift.

1. Navigate to the `dashboards/` directory in the repository.
2. Edit `agent_health.json`. You can import this directly via the SigNoz UI for rapid prototyping.
3. To deploy changes, commit the JSON state. The CI/CD pipeline executes a declarative sync against the SigNoz ClickHouse backing store via its internal configuration API.

## 3. Setting Up Alertmanager Hooks

1. Alerts are defined in `alerts/rules.yaml`.
2. **Routing:** Define the receiver hooks. Ensure the `telemetryhealth_agent_health_score` threshold breaches route to the designated engineering PagerDuty service.
   ```yaml
   receivers:
     - name: 'pagerduty-ai-ops'
       pagerduty_configs:
         - service_key: '<env_secret>'
   ```
3. **Validation:** Use `amtool check config alerts/rules.yaml` to ensure structural integrity prior to committing.
