package engine

import "fmt"

type DecisionNode struct {
	ID         string
	Type       string
	Label      string
	Confidence float64
	Status     string
	Detail     string
	Evidence   []string
}

type DecisionEdge struct {
	Source string
	Target string
	Type   string
}

type DecisionGraph struct {
	Nodes []DecisionNode
	Edges []DecisionEdge
}

type DecisionBuilder interface {
	Build(bg *BehaviorGraph) *DecisionGraph
}

type DefaultDecisionBuilder struct{}

func NewDecisionBuilder() *DefaultDecisionBuilder {
	return &DefaultDecisionBuilder{}
}

func (b *DefaultDecisionBuilder) Build(bg *BehaviorGraph) *DecisionGraph {
	nodes := []DecisionNode{}
	edges := []DecisionEdge{}
	nodeMap := make(map[string]bool)

	// Map to keep track of decisions linked to behaviors to build edges
	behaviorToDecisionID := make(map[string]string)

	for _, n := range bg.Nodes {
		var dNode DecisionNode
		confidence := 1.0

		// Infer decisions based on behavior type and status
		if n.Type == "tool" {
			if n.Status == "error" || n.Status == "critical" {
				dNode = DecisionNode{
					ID:         fmt.Sprintf("dec-failure-%s", n.ID),
					Type:       "Tool Failure",
					Label:      fmt.Sprintf("%s Failed", n.Label),
					Confidence: confidence,
					Status:     "error",
					Detail:     "Observable error in tool execution",
					Evidence:   []string{n.ID},
				}
			} else {
				dNode = DecisionNode{
					ID:         fmt.Sprintf("dec-sel-%s", n.ID),
					Type:       "Tool Selection",
					Label:      fmt.Sprintf("Select %s", n.Label),
					Confidence: confidence,
					Status:     "ok",
					Detail:     "Planner invoked tool",
					Evidence:   []string{n.ID},
				}
			}
		} else if n.Type == "planner" {
			dNode = DecisionNode{
				ID:         fmt.Sprintf("dec-plan-%s", n.ID),
				Type:       "Agent Routing",
				Label:      fmt.Sprintf("Route to %s", n.Label),
				Confidence: confidence,
				Status:     "ok",
				Detail:     "LLM planner invoked",
				Evidence:   []string{n.ID},
			}
		} else if n.Type == "retriever" {
			dNode = DecisionNode{
				ID:         fmt.Sprintf("dec-ret-%s", n.ID),
				Type:       "Document Selection",
				Label:      "Query Knowledge Base",
				Confidence: confidence,
				Status:     "ok",
				Detail:     "Vector search executed",
				Evidence:   []string{n.ID},
			}
		}

		if dNode.ID != "" {
			if !nodeMap[dNode.ID] {
				nodes = append(nodes, dNode)
				nodeMap[dNode.ID] = true
			}
			behaviorToDecisionID[n.ID] = dNode.ID
		}
	}

	// Resolve edges based on behavior edges
	edgeMap := make(map[string]bool)
	for _, e := range bg.Edges {
		srcDecID, srcOk := behaviorToDecisionID[e.Source]
		tgtDecID, tgtOk := behaviorToDecisionID[e.Target]

		if srcOk && tgtOk && srcDecID != tgtDecID {
			edgeID := fmt.Sprintf("%s->%s", srcDecID, tgtDecID)
			if !edgeMap[edgeID] {
				edges = append(edges, DecisionEdge{
					Source: srcDecID,
					Target: tgtDecID,
					Type:   "Triggered",
				})
				edgeMap[edgeID] = true
			}
		}
	}

	return &DecisionGraph{
		Nodes: nodes,
		Edges: edges,
	}
}
