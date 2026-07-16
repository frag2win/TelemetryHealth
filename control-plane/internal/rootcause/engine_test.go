package rootcause_test

import (
	"testing"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/rootcause"
	"github.com/frag2win/TelemetryHealth/control-plane/pkg/models"
)

func TestRootCauseEngine_ToolTimeout(t *testing.T) {
	engine := rootcause.NewEngine()

	// A behavior graph indicating a tool failure/timeout
	bg := &models.BehaviorGraph{
		TraceID: "trace-rc-1",
		AgentID: "ai-agent-v1",
		Nodes: []models.BehaviorNode{
			{
				BehaviorID:   "beh-1",
				Type:         "Tool Failure",
				Actor:        "Tool",
				Confidence:   0.95,
				ReplayEvents: []string{"span-tool-id"},
				Metadata: map[string]string{
					"llm.tool_name":  "web_search",
					"failure_reason": "TimeoutError: connection timed out",
				},
				Timestamp: time.Now(),
			},
		},
	}

	dg := &models.DecisionGraph{
		TraceID: "trace-rc-1",
		AgentID: "ai-agent-v1",
		Nodes: []models.DecisionNode{
			{
				DecisionID:   "dec-1",
				DecisionType: "Recovery Strategy",
				Actor:        "Tool",
				Confidence:   0.95,
				Status:       "Partial",
			},
		},
	}

	rc, err := engine.Analyze(bg, dg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rc.FailureType != "tool_timeout" {
		t.Errorf("expected FailureType tool_timeout, got %s", rc.FailureType)
	}

	if rc.Severity != "Critical" {
		t.Errorf("expected Severity Critical, got %s", rc.Severity)
	}

	if len(rc.EvidenceSpanIDs) != 1 || rc.EvidenceSpanIDs[0] != "span-tool-id" {
		t.Errorf("expected EvidenceSpanIDs [span-tool-id], got %v", rc.EvidenceSpanIDs)
	}
}

func TestRootCauseEngine_TokenLimit(t *testing.T) {
	engine := rootcause.NewEngine()

	// A behavior graph indicating high token usage
	bg := &models.BehaviorGraph{
		TraceID: "trace-rc-2",
		AgentID: "ai-agent-v1",
		Nodes: []models.BehaviorNode{
			{
				BehaviorID:   "beh-1",
				Type:         "LLM Call",
				Actor:        "LLM",
				Confidence:   1.0,
				ReplayEvents: []string{"span-llm-id"},
				Metadata: map[string]string{
					"llm.model":       "gpt-4o",
					"llm.token_usage": "5200",
				},
				Timestamp: time.Now(),
			},
		},
	}

	dg := &models.DecisionGraph{
		TraceID: "trace-rc-2",
		AgentID: "ai-agent-v1",
		Nodes: []models.DecisionNode{
			{
				DecisionID:   "dec-1",
				DecisionType: "Model Selection",
				Actor:        "LLM",
				Confidence:   1.0,
				Status:       "Completed",
			},
		},
	}

	rc, err := engine.Analyze(bg, dg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rc.FailureType != "token_limit" {
		t.Errorf("expected FailureType token_limit, got %s", rc.FailureType)
	}

	if rc.Severity != "High" {
		t.Errorf("expected Severity High, got %s", rc.Severity)
	}

	if len(rc.EvidenceSpanIDs) != 1 || rc.EvidenceSpanIDs[0] != "span-llm-id" {
		t.Errorf("expected EvidenceSpanIDs [span-llm-id], got %v", rc.EvidenceSpanIDs)
	}
}

func TestRootCauseEngine_LatencyAnomaly(t *testing.T) {
	engine := rootcause.NewEngine()

	// A behavior graph indicating high latency
	bg := &models.BehaviorGraph{
		TraceID: "trace-rc-3",
		AgentID: "ai-agent-v1",
		Nodes: []models.BehaviorNode{
			{
				BehaviorID:   "beh-1",
				Type:         "Memory Access",
				Actor:        "Memory",
				Confidence:   0.90,
				DurationMs:   2500, // 2.5s duration
				ReplayEvents: []string{"span-mem-id"},
				Timestamp:    time.Now(),
			},
		},
	}

	dg := &models.DecisionGraph{
		TraceID: "trace-rc-3",
		AgentID: "ai-agent-v1",
		Nodes: []models.DecisionNode{
			{
				DecisionID:   "dec-1",
				DecisionType: "Memory Decision",
				Actor:        "Memory",
				Confidence:   0.90,
				Status:       "Completed",
			},
		},
	}

	rc, err := engine.Analyze(bg, dg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rc.FailureType != "latency" {
		t.Errorf("expected FailureType latency, got %s", rc.FailureType)
	}

	if rc.Severity != "Medium" {
		t.Errorf("expected Severity Medium, got %s", rc.Severity)
	}
}
