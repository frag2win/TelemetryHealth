package models

import "time"

// Agent represents the core monitoring profile wrapper.
type Agent struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ServiceName string    `json:"service_name"`
	CreatedAt   time.Time `json:"created_at"`
}

// SpanData represents telemetry trace span properties used as evidence.
type SpanData struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id"`
	ServiceName  string            `json:"service_name"`
	Name         string            `json:"name"`
	DurationNano int64             `json:"duration_nano"`
	Timestamp    time.Time         `json:"timestamp"`
	Attributes   map[string]string `json:"attributes"`
	StatusCode   string            `json:"status_code"`
	StatusMsg    string            `json:"status_msg"`
}

// BehaviorNodeType defines the types of behavior nodes.
type BehaviorNodeType = string

const (
	NodeTypeLLMCall  BehaviorNodeType = "LLM_CALL"
	NodeTypeToolCall BehaviorNodeType = "TOOL_CALL"
	NodeTypeDBQuery  BehaviorNodeType = "DB_QUERY"
	NodeTypeRouting  BehaviorNodeType = "ROUTING"
)

// BehaviorNodeStatus defines execution statuses.
type BehaviorNodeStatus = string

const (
	StatusSuccess BehaviorNodeStatus = "SUCCESS"
	StatusFailed  BehaviorNodeStatus = "FAILED"
	StatusTimeout BehaviorNodeStatus = "TIMEOUT"
)

// BehaviorNode maps tracing history into execution steps.
type BehaviorNode struct {
	NodeID       string             `json:"node_id,omitempty"`
	BehaviorID   string             `json:"behavior_id,omitempty"`
	Type         BehaviorNodeType   `json:"type"`          // e.g. "LLM_CALL", "Tool Retry", "Prompt Expansion", etc.
	Status       BehaviorNodeStatus `json:"status,omitempty"`
	Actor        string             `json:"actor,omitempty"`         // e.g. "Planner", "Retriever", "Memory", "LLM", "Tool", etc.
	DurationMs   float64            `json:"duration_ms"`
	Confidence   float64            `json:"confidence,omitempty"`
	ReplayEvents []string           `json:"replay_events,omitempty"` // IDs of spans / sub-events
	Metadata     map[string]string  `json:"metadata,omitempty"`
	Timestamp    time.Time          `json:"timestamp"`
}

// BehaviorEdge represents relationships between behaviors.
type BehaviorEdge struct {
	Source      string  `json:"source"`
	Destination string  `json:"destination"`
	Type        string  `json:"relationship"` // e.g. "Triggered", "Depends On", "Retries", etc.
	Confidence  float64 `json:"confidence"`
}

// BehaviorGraph represents a collection of execution steps in a trace.
type BehaviorGraph struct {
	TraceID   string         `json:"trace_id"`
	AgentID   string         `json:"agent_id,omitempty"`
	Nodes     []BehaviorNode `json:"nodes"`
	Edges     []BehaviorEdge `json:"edges,omitempty"`
	Timestamp time.Time      `json:"timestamp,omitempty"`
}

// DecisionNode represents logical choice evaluations by an agent.
type DecisionNode struct {
	DecisionID          string            `json:"decision_id"`
	BehaviorNodeID      string            `json:"behavior_node_id,omitempty"`
	DecisionType        string            `json:"decision_type,omitempty"`
	Timestamp           time.Time         `json:"timestamp,omitempty"`
	Actor               string            `json:"actor,omitempty"`
	Confidence          float64           `json:"confidence,omitempty"`
	SupportingBehaviors []string          `json:"supporting_behaviors,omitempty"`
	EvidenceCount       int               `json:"evidence_count,omitempty"`
	Status              string            `json:"status,omitempty"` // "Completed", "Partial", "Unknown"
	Inputs              map[string]string `json:"inputs,omitempty"`
	InputPrompt         string            `json:"input_prompt,omitempty"`
	ChosenOption        string            `json:"chosen_option"`
	Alternatives        []string          `json:"alternatives,omitempty"`
}

// DecisionEdge represents causal relations between decisions.
type DecisionEdge struct {
	Source        string  `json:"source"`
	Destination   string  `json:"destination"`
	Relationship  string  `json:"relationship"` // e.g. "Triggered", "Depends On", "Retries", etc.
	Confidence    float64 `json:"confidence"`
	TemporalOrder int     `json:"temporal_order"`
}

// DecisionGraph is a collection of decision nodes within a specific context.
type DecisionGraph struct {
	TraceID   string         `json:"trace_id"`
	AgentID   string         `json:"agent_id,omitempty"`
	Decisions []DecisionNode `json:"decisions,omitempty"`
	Nodes     []DecisionNode `json:"nodes,omitempty"`
	Edges     []DecisionEdge `json:"edges,omitempty"`
	Timestamp time.Time      `json:"timestamp,omitempty"`
}

// FailureType defines categories of failures in the tracing pipeline.
type FailureType = string

const (
	FailureCardinalityExplosion FailureType = "CARDINALITY_EXPLOSION"
	FailureOrphanSpan           FailureType = "ORPHAN_SPAN"
	FailureSamplingGap          FailureType = "SAMPLING_GAP"
	FailureCoverageHole         FailureType = "COVERAGE_HOLE"
)

// Severity defines the criticality level of an issue.
type Severity = string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityWarning  Severity = "WARNING"
)

// RootCause is the analytical engine's error verdict payload.
type RootCause struct {
	TraceID         string      `json:"trace_id"`
	AgentID         string      `json:"agent_id"`
	FailureType     FailureType `json:"failure_type"` // e.g., "CARDINALITY_EXPLOSION", "tool_timeout", etc.
	EvidenceSpanIDs []string    `json:"evidence_span_ids"`
	Severity        Severity    `json:"severity"` // e.g., "CRITICAL", "High", etc.
	Description     string      `json:"description"`
	Confidence      float64     `json:"confidence,omitempty"`
	Timestamp       time.Time   `json:"timestamp,omitempty"`
	Status          string      `json:"status,omitempty"` // "Detected", "Unknown"
}
