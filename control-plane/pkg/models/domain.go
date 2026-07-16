package models

import "time"

// Agent represents the core monitoring profile wrapper.
type Agent struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ServiceName string    `json:"service_name"`
	CreatedAt   time.Time `json:"created_at"`
}

// BehaviorNodeType defines the types of behavior nodes.
type BehaviorNodeType string

const (
	NodeTypeLLMCall  BehaviorNodeType = "LLM_CALL"
	NodeTypeToolCall BehaviorNodeType = "TOOL_CALL"
	NodeTypeDBQuery  BehaviorNodeType = "DB_QUERY"
	NodeTypeRouting  BehaviorNodeType = "ROUTING"
)

// BehaviorNodeStatus defines execution statuses.
type BehaviorNodeStatus string

const (
	StatusSuccess BehaviorNodeStatus = "SUCCESS"
	StatusFailed  BehaviorNodeStatus = "FAILED"
	StatusTimeout BehaviorNodeStatus = "TIMEOUT"
)

// BehaviorNode maps tracing history into execution steps.
type BehaviorNode struct {
	NodeID     string             `json:"node_id"`
	Type       BehaviorNodeType   `json:"type"`
	Status     BehaviorNodeStatus `json:"status"`
	DurationMs int64              `json:"duration_ms"`
	Timestamp  time.Time          `json:"timestamp"`
}

// BehaviorGraph represents a collection of execution steps in a trace.
type BehaviorGraph struct {
	TraceID string         `json:"trace_id"`
	Nodes   []BehaviorNode `json:"nodes"`
}

// DecisionNode represents logical choice evaluations by an agent.
type DecisionNode struct {
	DecisionID     string   `json:"decision_id"`
	BehaviorNodeID string   `json:"behavior_node_id"`
	ChosenOption   string   `json:"chosen_option"`
	Alternatives   []string `json:"alternatives"`
	InputPrompt    string   `json:"input_prompt"`
}

// DecisionGraph is a collection of decision nodes within a specific context.
type DecisionGraph struct {
	TraceID  string         `json:"trace_id"`
	Decisions []DecisionNode `json:"decisions"`
}

// FailureType defines categories of failures in the tracing pipeline.
type FailureType string

const (
	FailureCardinalityExplosion FailureType = "CARDINALITY_EXPLOSION"
	FailureOrphanSpan           FailureType = "ORPHAN_SPAN"
	FailureSamplingGap          FailureType = "SAMPLING_GAP"
	FailureCoverageHole         FailureType = "COVERAGE_HOLE"
)

// Severity defines the criticality level of an issue.
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityWarning  Severity = "WARNING"
)

// RootCause is the analytical engine's error verdict payload.
type RootCause struct {
	TraceID         string      `json:"trace_id"`
	AgentID         string      `json:"agent_id"`
	FailureType     FailureType `json:"failure_type"`
	EvidenceSpanIDs []string    `json:"evidence_span_ids"`
	Severity        Severity    `json:"severity"`
	Description     string      `json:"description"`
}
