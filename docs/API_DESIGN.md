# API Design

This document details the REST API endpoints and JSON response payloads to act as the strict development contract for the Control Plane API and frontend consumers.

## Base Path
All endpoints are relative to: `/api/v1`

## Standard Error Format (APIError)
When an internal breakdown occurs, the API will return this standard error structure.

```json
{
  "error_code": "INVALID_UUID",
  "message": "The provided tenant ID is structurally malformed"
}
```

## Endpoints

### 1. Retrieve Behavior Graph
Returns the execution steps for a specific trace.

**Endpoint:** `GET /api/v1/tenant/{tenant_id}/agent/{id}/traces/{trace_id}/behavior`

**Successful Response (200 OK):**
```json
{
  "trace_id": "a1b2c3d4-e5f6-7890-1234-56789abcdef0",
  "nodes": [
    {
      "node_id": "span-901283",
      "type": "LLM_CALL",
      "status": "SUCCESS",
      "duration_ms": 1250,
      "timestamp": "2026-07-16T20:57:33Z"
    },
    {
      "node_id": "span-901284",
      "type": "TOOL_CALL",
      "status": "FAILED",
      "duration_ms": 45,
      "timestamp": "2026-07-16T20:57:34Z"
    }
  ]
}
```

### 2. Retrieve Decision Graph
Returns the logical choice evaluations made by an agent during a trace.

**Endpoint:** `GET /api/v1/tenant/{tenant_id}/agent/{id}/traces/{trace_id}/decisions`

**Successful Response (200 OK):**
```json
{
  "trace_id": "a1b2c3d4-e5f6-7890-1234-56789abcdef0",
  "decisions": [
    {
      "decision_id": "dec-xyz-123",
      "behavior_node_id": "span-901283",
      "chosen_option": "execute_database_query",
      "alternatives": [
        "ask_user_for_clarification",
        "fallback_to_cache"
      ],
      "input_prompt": "Determine the next action based on user request: 'Find active users'"
    }
  ]
}
```

### 3. Retrieve Root Cause Analysis
Returns the analytical engine's verdict regarding a trace's structural integrity or failure.

**Endpoint:** `GET /api/v1/tenant/{tenant_id}/agent/{id}/traces/{trace_id}/root-cause`

**Successful Response (200 OK):**
```json
{
  "trace_id": "a1b2c3d4-e5f6-7890-1234-56789abcdef0",
  "agent_id": "f8a9b2c1-d3e4-5678-9012-345678abcdef",
  "failure_type": "ORPHAN_SPAN",
  "evidence_span_ids": [
    "span-901284",
    "span-901285"
  ],
  "severity": "CRITICAL",
  "description": "Trace structural integrity violation. Parent span IDs referenced do not exist in the ingested stream, indicating a sampling gap or missing telemetry payload."
}
```

**Validation Error Response (400 Bad Request):**
```json
{
  "error_code": "MISSING_SPANS",
  "message": "Trace data cannot be analyzed due to 45% missing span structures, breaching analysis threshold"
}
```
