package engine

import (
	"context"
	"fmt"
)

type GraphNodeData struct {
	Label  string `json:"label"`
	Type   string `json:"type"`   // "service", "kafka", "database", "issue"
	Status string `json:"status"` // "healthy", "warning", "critical", "info"
	Detail string `json:"detail,omitempty"`
}

type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type GraphNode struct {
	ID       string        `json:"id"`
	Position NodePosition  `json:"position"`
	Data     GraphNodeData `json:"data"`
	Type     string        `json:"type,omitempty"` // For custom react flow nodes
}

type GraphEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Animated bool   `json:"animated,omitempty"`
	Label    string `json:"label,omitempty"`
}

type Graph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// Engine handles graph generation using the ReplayRepository abstraction.
type Engine struct {
	repo       ReplayRepository
	builder    BehaviorBuilder
	decBuilder DecisionBuilder
	rcBuilder  RootCauseBuilder
}

func NewEngine(repo ReplayRepository) *Engine {
	return &Engine{
		repo:       repo,
		builder:    NewBehaviorBuilder(),
		decBuilder: NewDecisionBuilder(),
		rcBuilder:  NewRootCauseBuilder(),
	}
}

// GenerateBehaviorGraph returns a live pipeline behavior graph
// It builds a Behavior Graph based on recent replay events.
func (e *Engine) GenerateBehaviorGraph(tenantID string) Graph {
	events, err := e.repo.GetRecentReplays(context.Background(), tenantID, 100)
	if err != nil || len(events) == 0 {
		return defaultTopology()
	}

	bg, err := e.builder.Build(events)
	if err != nil || len(bg.Nodes) == 0 {
		return defaultTopology()
	}

	return toReactFlowGraph(bg)
}

// GenerateRootCause returns a causal decision graph explaining an issue.
func (e *Engine) GenerateRootCause(tenantID, traceID string) Graph {
	var events []ReplayEvent
	var err error

	// Inject benchmark deterministic scenarios for the demo
	if len(traceID) > 10 && traceID[:10] == "benchmark-" {
		scenarioID := traceID[10:]
		events = GetBenchmarkScenario(scenarioID, tenantID)
	} else {
		events, err = e.repo.GetReplay(context.Background(), tenantID, traceID)
	}

	if err != nil || len(events) == 0 {
		return defaultRootCause(traceID)
	}

	bg, err := e.builder.Build(events)
	if err != nil || len(bg.Nodes) == 0 {
		return defaultRootCause(traceID)
	}

	dg := e.decBuilder.Build(bg)
	rcg := e.rcBuilder.Build(dg)

	return toReactFlowRootCauseGraph(rcg)
}

func toReactFlowGraph(bg *BehaviorGraph) Graph {
	nodes := []GraphNode{}
	edges := []GraphEdge{}

	xPos := 50.0
	for _, n := range bg.Nodes {
		nodes = append(nodes, GraphNode{
			ID: n.ID,
			Position: NodePosition{X: xPos, Y: 100},
			Data: GraphNodeData{
				Label:  n.Label,
				Type:   n.Type,
				Status: n.Status,
				Detail: n.Detail,
			},
		})
		xPos += 250
	}

	for _, e := range bg.Edges {
		edges = append(edges, GraphEdge{
			ID:       fmt.Sprintf("e-%s-%s", e.Source, e.Target),
			Source:   e.Source,
			Target:   e.Target,
			Animated: true,
		})
	}

	return Graph{
		Nodes: nodes,
		Edges: edges,
	}
}

func toReactFlowRootCauseGraph(rcg *RootCauseGraph) Graph {
	nodes := []GraphNode{}
	edges := []GraphEdge{}

	xPos := 50.0
	for _, n := range rcg.Nodes {
		nodes = append(nodes, GraphNode{
			ID: n.ID,
			Position: NodePosition{X: xPos, Y: 100},
			Data: GraphNodeData{
				Label:  n.Label,
				Type:   "issue",
				Status: n.Severity,
				Detail: fmt.Sprintf("Category: %s | Confidence: %.2f", n.Category, n.Confidence),
			},
		})
		xPos += 250
	}

	for _, e := range rcg.Edges {
		edges = append(edges, GraphEdge{
			ID:       fmt.Sprintf("e-%s-%s", e.Source, e.Target),
			Source:   e.Source,
			Target:   e.Target,
			Animated: true,
			Label:    e.Type,
		})
	}

	return Graph{
		Nodes: nodes,
		Edges: edges,
	}
}



func defaultTopology() Graph {
	return Graph{
		Nodes: []GraphNode{
			{ID: "app", Position: NodePosition{X: 50, Y: 100}, Data: GraphNodeData{Label: "Waiting for data...", Type: "service", Status: "info"}},
		},
		Edges: []GraphEdge{},
	}
}

func defaultRootCause(issueID string) Graph {
	return Graph{
		Nodes: []GraphNode{
			{ID: "1", Position: NodePosition{X: 50, Y: 50}, Data: GraphNodeData{Label: "Trace Not Found", Type: "event", Status: "info", Detail: "Waiting for replay event data"}},
		},
		Edges: []GraphEdge{},
	}
}
