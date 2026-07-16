package behavior_test

import (
	"testing"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/behavior"
	"github.com/frag2win/TelemetryHealth/control-plane/pkg/models"
)

func TestBehaviorEngine_NormalFlow(t *testing.T) {
	engine := behavior.NewEngine()

	// Normal workflow spans
	spans := []models.SpanData{
		{
			TraceID:      "trace-1",
			SpanID:       "span-root",
			ParentSpanID: "",
			Name:         "agent.workflow",
			ServiceName:  "ai-agent",
			DurationNano: int64(2000 * time.Millisecond),
			Timestamp:    time.Now(),
			Attributes:   map[string]string{"workflow.topic": "testing"},
		},
		{
			TraceID:      "trace-1",
			SpanID:       "span-tool",
			ParentSpanID: "span-root",
			Name:         "agent.research",
			ServiceName:  "ai-agent",
			DurationNano: int64(800 * time.Millisecond),
			Timestamp:    time.Now().Add(100 * time.Millisecond),
			Attributes:   map[string]string{"llm.tool_name": "web_search"},
		},
		{
			TraceID:      "trace-1",
			SpanID:       "span-llm",
			ParentSpanID: "span-root",
			Name:         "agent.summarize",
			ServiceName:  "ai-agent",
			DurationNano: int64(1000 * time.Millisecond),
			Timestamp:    time.Now().Add(1000 * time.Millisecond),
			Attributes:   map[string]string{"llm.model": "gpt-4o"},
		},
	}

	graph, err := engine.Reconstruct("trace-1", spans)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if graph.TraceID != "trace-1" {
		t.Errorf("expected TraceID trace-1, got %s", graph.TraceID)
	}

	if len(graph.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(graph.Nodes))
	}

	// Verify types
	nodeMap := make(map[string]models.BehaviorNode)
	for _, n := range graph.Nodes {
		nodeMap[n.BehaviorID] = n
	}

	rootNode, ok := nodeMap["beh_span-root"]
	if !ok || rootNode.Type != "Workflow" {
		t.Errorf("root node type is incorrect: %+v", rootNode)
	}

	toolNode, ok := nodeMap["beh_span-tool"]
	if !ok || toolNode.Type != "Tool Execution" {
		t.Errorf("tool node type is incorrect: %+v", toolNode)
	}

	llmNode, ok := nodeMap["beh_span-llm"]
	if !ok || llmNode.Type != "LLM Call" {
		t.Errorf("llm node type is incorrect: %+v", llmNode)
	}

	// Verify edges
	if len(graph.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(graph.Edges))
	}

	for _, edge := range graph.Edges {
		if edge.Source != "beh_span-root" {
			t.Errorf("expected source beh_span-root, got %s", edge.Source)
		}
		if edge.Type != "Triggered" {
			t.Errorf("expected relationship Triggered, got %s", edge.Type)
		}
	}
}

func TestBehaviorEngine_FailureAndRetry(t *testing.T) {
	engine := behavior.NewEngine()

	// Failure and retry spans
	spans := []models.SpanData{
		{
			TraceID:      "trace-2",
			SpanID:       "span-root",
			ParentSpanID: "",
			Name:         "agent.workflow",
			ServiceName:  "ai-agent",
			DurationNano: int64(3000 * time.Millisecond),
			Timestamp:    time.Now(),
		},
		// Failed Tool call
		{
			TraceID:      "trace-2",
			SpanID:       "span-tool-fail",
			ParentSpanID: "span-root",
			Name:         "agent.research",
			ServiceName:  "ai-agent",
			DurationNano: int64(100 * time.Millisecond),
			Timestamp:    time.Now().Add(50 * time.Millisecond),
			Attributes:   map[string]string{"llm.tool_name": "web_search", "llm.tool_call.error": "TimeoutError"},
			StatusCode:   "ERROR",
		},
		// Successful Retry Tool call
		{
			TraceID:      "trace-2",
			SpanID:       "span-tool-success",
			ParentSpanID: "span-root",
			Name:         "agent.research",
			ServiceName:  "ai-agent",
			DurationNano: int64(800 * time.Millisecond),
			Timestamp:    time.Now().Add(1000 * time.Millisecond),
			Attributes:   map[string]string{"llm.tool_name": "web_search"},
		},
	}

	graph, err := engine.Reconstruct("trace-2", spans)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nodeMap := make(map[string]models.BehaviorNode)
	for _, n := range graph.Nodes {
		nodeMap[n.BehaviorID] = n
	}

	failNode, ok := nodeMap["beh_span-tool-fail"]
	if !ok || failNode.Type != "Tool Failure" {
		t.Errorf("fail node type incorrect: %+v", failNode)
	}

	successNode, ok := nodeMap["beh_span-tool-success"]
	if !ok || successNode.Type != "Tool Retry" {
		t.Errorf("success node type incorrect: %+v", successNode)
	}

	// Verify retry edge exists
	hasRetryEdge := false
	for _, edge := range graph.Edges {
		if edge.Source == "beh_span-tool-fail" && edge.Destination == "beh_span-tool-success" && edge.Type == "Retries" {
			hasRetryEdge = true
			break
		}
	}
	if !hasRetryEdge {
		t.Errorf("expected 'Retries' edge from fail node to retry node, edges: %+v", graph.Edges)
	}
}
