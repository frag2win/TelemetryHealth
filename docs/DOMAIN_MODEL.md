# Domain Model

This document establishes the single source of truth for the TelemetryHealth data model.

## Agent
The core monitoring profile wrapper for an AI agent.
* **id** (UUID string): Unique identifier for the agent.
* **name** (string): Human-readable name of the agent.
* **service_name** (string): The backing service associated with the agent.
* **created_at** (Timestamp): When the agent profile was created.

## BehaviorGraph / BehaviorNode
Maps tracing history into execution steps.
* **node_id** (string): Unique identifier for the node (often corresponds to a span ID).
* **type** (enum): The type of behavior node. Values: `LLM_CALL`, `TOOL_CALL`, `DB_QUERY`, `ROUTING`.
* **status** (enum): Execution status. Values: `SUCCESS`, `FAILED`, `TIMEOUT`.
* **duration_ms** (int64): Execution duration in milliseconds.
* **timestamp** (Timestamp): When the behavior was recorded.

## DecisionGraph / DecisionNode
Represents logical choice evaluations by an agent.
* **decision_id** (string): Unique identifier for the decision point.
* **behavior_node_id** (string): Foreign key mapping back to the associated BehaviorNode.
* **chosen_option** (string): The path or option selected by the agent.
* **alternatives** (array of strings): Other paths or options evaluated but rejected.
* **input_prompt** (string): The prompt or contextual input leading to this decision.

## RootCause
The analytical engine's error verdict payload for tracing breakdowns.
* **trace_id** (UUID string): The trace where the failure occurred.
* **agent_id** (UUID string): The agent responsible for the trace.
* **failure_type** (enum): Category of failure. Values: `CARDINALITY_EXPLOSION`, `ORPHAN_SPAN`, `SAMPLING_GAP`, `COVERAGE_HOLE`.
* **evidence_span_ids** (array of strings): List of span IDs contributing to the error verdict.
* **severity** (enum): Severity of the failure. Values: `CRITICAL`, `WARNING`.
* **description** (string): Detailed explanation of the failure.
