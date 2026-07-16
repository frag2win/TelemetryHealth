# System Architecture

This document details the end-to-end telemetry pipeline mechanics, data flows, and structural architecture for TelemetryHealth.

## Deployment Architecture Map

```text
┌────────────────────────────────────────────────────────┐
│                   AI Agent Cluster                     │
│    Emits trace payloads instrumented with llm.* tags   │
└───────────────────────────┬────────────────────────────┘
                            │ OTLP / gRPC (mTLS)
                            ▼
┌────────────────────────────────────────────────────────┐
│            TelemetryHealth Collector Fleet             │
│   ┌────────────────────────────────────────────────┐   │
│   │ Custom OTel Processor (Fan-Out Async Branch)   │   │
│   │ Enrichments: Tag telemetry.health.ai_agent=true│   │
│   │ Circuit Breaker (Fail-Open Guard Mechanism)    │   │
│   └───────────────────────┬────────────────────────┘   │
└───────────────────────────┼────────────────────────────┘
                            │ gRPC Stream Export
                            ▼
┌────────────────────────────────────────────────────────┐
│                  Control Plane (Go)                    │
│   ┌────────────────────────────────────────────────┐   │
│   │ Ingest Gateway: Receives telemetry streams     │   │
│   │ Stream Worker Daemon: Merges HLL & rollups     │   │
│   │ go-chi REST API Engine (Uses strict UUID path) │   │
│   └───────────────────────┬────────────────────────┘   │
└───────────────────────────┼────────────────────────────┘
                            │ Native mTLS Pipeline
                            ▼
┌────────────────────────────────────────────────────────┐
│             SigNoz / ClickHouse Store                  │
│   ┌────────────────────────────────────────────────┐   │
│   │  ClickHouse: OLAP Analytical Aggregator        │   │
│   │  SigNoz UI: Custom Gauges & Alertmanager Web   │   │
│   └────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────┘
```

## End-to-End Telemetry Pipeline Mechanics
1. **AI Agent Cluster (Emission):** Microservices utilizing AI agents generate OpenTelemetry traces. These traces are enriched natively with standard OpenLLM semantic conventions (`llm.request.tokens`, `llm.response.model`, etc.). Transport utilizes mTLS over gRPC.
2. **Collector Fleet (Processing):** 
   - A custom OpenTelemetry processor intercepts incoming spans.
   - **Enrichment:** Spans missing AI context but belonging to an AI agent trace are tagged with `telemetry.health.ai_agent=true`.
   - **Fan-Out Async Branch:** The pipeline duplicates streams securely. One branch writes to primary storage; the other feeds the Control Plane for real-time anomaly detection.
   - **Fail-Open Circuit Breaker:** In the event of processor lag or downstream failures, structural drops are absorbed, bypassing the async analytical branch to preserve core metric delivery.
3. **Control Plane (Analysis):** 
   - Receives gRPC streams, applying HyperLogLog (HLL) cardinality approximations and trace rollups.
   - Executes structural integrity checks to detect `ORPHAN_SPAN` or `SAMPLING_GAP` errors.
   - Exposes findings via a `go-chi` REST API utilizing Vite-compliant proxy routing (`/api/v1/...`).
4. **Data Sink (Storage):** Final telemetry rests in ClickHouse for high-throughput OLAP querying, powered by SigNoz visual dashboards and Alertmanager.
