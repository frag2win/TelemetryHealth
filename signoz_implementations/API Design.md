# Section 11
# API Design

Version: 1.0

---

# Overview

TelemetryHealth exposes a versioned REST API that allows clients to ingest telemetry intelligence, retrieve replay sessions, inspect behavioral artifacts, query incident intelligence, and integrate with external observability platforms.

The API is designed around the platform's core domain model rather than its internal implementation.

All APIs are stateless, versioned, deterministic, and vendor-neutral.

---

# API Design Principles

TelemetryHealth APIs follow six principles.

## Resource-Oriented

Every endpoint represents a domain object.

Examples

- Replay Session
- Behavior Graph
- Decision Graph
- Incident
- Evidence
- Replay Timeline

---

## Read-Optimized

Telemetry ingestion occurs through OpenTelemetry.

TelemetryHealth APIs primarily expose reconstructed intelligence.

---

## Explainable

Every API returning analytical results must expose:

- Confidence
- Supporting Evidence
- Metadata

---

## Immutable

Replay Sessions never change after completion.

Historical replay is reproducible.

---

## Versioned

```
/api/v1/
```

Future versions

```
/api/v2/
```

```
/api/v3/
```

---

## Vendor Neutral

The API does not expose implementation details of:

- SigNoz
- ClickHouse
- Kafka
- Jaeger

The API represents TelemetryHealth concepts.

---

# API Overview

```
/api/v1

    replay/

    behaviors/

    decisions/

    incidents/

    evidence/

    narratives/

    benchmark/

    health/

    search/

    export/
```

---

# Replay APIs

## List Replay Sessions

GET

```
/api/v1/replays
```

Query Parameters

```
status

severity

agent

service

start

end

limit

offset
```

Response

```json
{
  "total": 245,
  "items": [
    {
      "replay_id": "replay_001",
      "status": "FAILED",
      "duration_ms": 8421,
      "incident": true
    }
  ]
}
```

---

## Get Replay Session

GET

```
/api/v1/replays/{id}
```

Returns

- Timeline
- Behaviors
- Decisions
- Root Cause
- Evidence

---

## Replay Timeline

GET

```
/api/v1/replays/{id}/timeline
```

Returns

Chronological Replay Events.

---

## Replay Graph

GET

```
/api/v1/replays/{id}/graph
```

Returns

Behavior Graph.

---

# Behavior APIs

## Get Behaviors

GET

```
/api/v1/behaviors
```

Supports filtering by

- Agent
- Tool
- Planner
- Type
- Confidence

---

## Get Behavior

GET

```
/api/v1/behaviors/{id}
```

Returns

- Replay Events
- Metadata
- Signature
- Related Decisions

---

# Decision APIs

## List Decisions

GET

```
/api/v1/decisions
```

Returns

Decision summaries.

---

## Get Decision

GET

```
/api/v1/decisions/{id}
```

Response

```json
{
  "decision_id": "dec_101",
  "type": "Retry",
  "confidence": 0.94,
  "evidence_count": 12
}
```

---

## Explain Decision

GET

```
/api/v1/decisions/{id}/explain
```

Returns

- Supporting Rules
- Supporting Behaviors
- Evidence
- Confidence
- Alternative Decisions

This endpoint powers the Explain button in Flight Recorder.

---

# Incident APIs

## List Incidents

GET

```
/api/v1/incidents
```

Filters

- Severity
- Status
- Category
- Service
- Replay Session

---

## Get Incident

GET

```
/api/v1/incidents/{id}
```

Returns

- Root Cause Graph
- Recovery Chain
- Timeline
- Evidence

---

# Root Cause API

GET

```
/api/v1/incidents/{id}/root-cause
```

Returns

Complete Root Cause Graph.

---

# Evidence APIs

## Get Evidence

GET

```
/api/v1/evidence/{id}
```

Returns

Supporting

- Spans
- Logs
- Metrics
- Replay Events

---

## Search Evidence

GET

```
/api/v1/evidence/search
```

Filters

- Trace
- Span
- Replay
- Incident
- Decision

---

# Narrative APIs

## Incident Summary

GET

```
/api/v1/incidents/{id}/summary
```

Returns

Human-readable explanation.

---

## Replay Summary

GET

```
/api/v1/replays/{id}/summary
```

Returns

Replay narrative.

---

# Benchmark APIs

## Run Benchmark

POST

```
/api/v1/benchmark/run
```

Body

```json
{
  "suite": "telemetryhealth-v1"
}
```

Returns

Benchmark Job ID.

---

## Benchmark Results

GET

```
/api/v1/benchmark/results/{id}
```

Returns

Accuracy

Precision

Recall

Detection Rate

Performance

---

# Search APIs

GET

```
/api/v1/search
```

Supports

- Replay
- Behavior
- Decision
- Incident
- Evidence

Search is semantic across the entire platform.

---

# Export APIs

## Export Replay

GET

```
/api/v1/replays/{id}/export
```

Formats

- JSON
- Markdown
- HTML
- PDF

---

## Export Incident

GET

```
/api/v1/incidents/{id}/export
```

---

# Health APIs

GET

```
/api/v1/health
```

Returns

Platform health.

---

GET

```
/api/v1/version
```

Returns

Version metadata.

---

# Authentication

Supported

- API Key
- JWT
- OAuth2
- Service Token

Future

- mTLS

---

# Authorization

Role-Based Access Control

Roles

- Viewer
- Engineer
- Operator
- Administrator

Replay export permissions are configurable.

---

# Rate Limiting

Replay APIs

100 requests/minute

Search

50 requests/minute

Benchmark

10 requests/hour

Export

20 requests/hour

---

# Error Format

```json
{
  "error": {
    "code": "INCIDENT_NOT_FOUND",
    "message": "Replay Session does not exist.",
    "request_id": "req_12345"
  }
}
```

---

# API Versioning

Major versions

```
/v1
```

Breaking changes require

```
/v2
```

Replay Sessions remain backward compatible.

---

# OpenAPI

TelemetryHealth publishes an OpenAPI 3.1 specification.

Clients can generate SDKs for:

- Go
- Python
- TypeScript
- Java

---

# Non-functional Requirements

Latency

<100 ms

Availability

99.9%

Stateless

Yes

Deterministic

Yes

Thread Safe

Yes

Idempotent

GET endpoints only

---

# Exit Criteria

The API layer is complete when:

- Every domain object is addressable.
- Every analytical result is explainable.
- Replay Sessions are exportable.
- APIs remain vendor-neutral.
- OpenAPI specification can be generated without modification.

The API becomes the primary integration surface for TelemetryHealth and enables future SDKs, CLI tooling, MCP integrations, and ecosystem extensions.