package decision_test

import (
	"testing"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/decision"
	"github.com/frag2win/TelemetryHealth/control-plane/pkg/models"
)

func TestDecisionEngine_ReconstructWorkflow(t *testing.T) {
	engine := decision.NewEngine()

	// A behavior graph representing a workflow with a failed tool call, a prompt expansion, and a retry
	bg := &models.BehaviorGraph{
		TraceID:   "trace-dec-1",
		AgentID:   "ai-agent-v1",
		Timestamp: time.Now(),
		Nodes: []models.BehaviorNode{
			{
				BehaviorID: "beh-workflow",
				Type:       "Workflow",
				Actor:      "Supervisor",
				Confidence: 1.0,
				Timestamp:  time.Now(),
			},
			{
				BehaviorID: "beh-tool-fail",
				Type:       "Tool Failure",
				Actor:      "Tool",
				Confidence: 0.9,
				Metadata:   map[string]string{"llm.tool_name": "database_query"},
				Timestamp:  time.Now().Add(100 * time.Millisecond),
			},
			{
				BehaviorID: "beh-prompt",
				Type:       "Prompt Expansion",
				Actor:      "LLM",
				Confidence: 1.0,
				Timestamp:  time.Now().Add(200 * time.Millisecond),
			},
			{
				BehaviorID: "beh-tool-retry",
				Type:       "Tool Retry",
				Actor:      "Tool",
				Confidence: 1.0,
				Metadata:   map[string]string{"llm.tool_name": "database_query"},
				Timestamp:  time.Now().Add(300 * time.Millisecond),
			},
		},
		Edges: []models.BehaviorEdge{
			{Source: "beh-workflow", Destination: "beh-tool-fail", Type: "Triggered", Confidence: 1.0},
			{Source: "beh-tool-fail", Destination: "beh-prompt", Type: "Triggered", Confidence: 1.0},
			{Source: "beh-prompt", Destination: "beh-tool-retry", Type: "Triggered", Confidence: 1.0},
			{Source: "beh-tool-fail", Destination: "beh-tool-retry", Type: "Retries", Confidence: 1.0},
		},
	}

	dg, err := engine.Reconstruct(bg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dg.TraceID != "trace-dec-1" {
		t.Errorf("expected TraceID trace-dec-1, got %s", dg.TraceID)
	}

	if len(dg.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(dg.Nodes))
	}

	nodeMap := make(map[string]models.DecisionNode)
	for _, n := range dg.Nodes {
		nodeMap[n.DecisionID] = n
	}

	// Verify we have a recovery strategy decision for Tool Failure
	failDec, ok := nodeMap["dec_beh-tool-fail"]
	if !ok {
		t.Fatal("fail decision not found")
	}
	if failDec.DecisionType != "Recovery Strategy" {
		t.Errorf("expected Recovery Strategy decision, got %s", failDec.DecisionType)
	}
	if failDec.Status != "Partial" {
		t.Errorf("expected status Partial for failed tool call, got %s", failDec.Status)
	}

	// Verify prompt expansion decision
	promptDec, ok := nodeMap["dec_beh-prompt"]
	if !ok {
		t.Fatal("prompt decision not found")
	}
	if promptDec.DecisionType != "Prompt Decision" {
		t.Errorf("expected Prompt Decision, got %s", promptDec.DecisionType)
	}

	// Verify tool retry decision
	retryDec, ok := nodeMap["dec_beh-tool-retry"]
	if !ok {
		t.Fatal("retry decision not found")
	}
	if retryDec.DecisionType != "Tool Retry" {
		t.Errorf("expected Tool Retry decision, got %s", retryDec.DecisionType)
	}
}
