package engine

import "fmt"

type RootCauseNode struct {
	ID            string
	Category      string
	Label         string
	Severity      string
	Confidence    float64
	EvidenceCount int
	Status        string
}

type RootCauseEdge struct {
	Source string
	Target string
	Type   string
}

type RootCauseGraph struct {
	Nodes []RootCauseNode
	Edges []RootCauseEdge
}

type RootCauseBuilder interface {
	Build(dg *DecisionGraph) *RootCauseGraph
}

type DefaultRootCauseBuilder struct{}

func NewRootCauseBuilder() *DefaultRootCauseBuilder {
	return &DefaultRootCauseBuilder{}
}

func (b *DefaultRootCauseBuilder) Build(dg *DecisionGraph) *RootCauseGraph {
	nodes := []RootCauseNode{}
	edges := []RootCauseEdge{}
	nodeMap := make(map[string]bool)
	
	// Map to link decisions to root causes
	decisionToCauseID := make(map[string]string)

	for _, n := range dg.Nodes {
		var rcNode RootCauseNode
		
		// Map Decisions to Failure/Root Cause nodes
		if n.Type == "Tool Failure" {
			rcNode = RootCauseNode{
				ID:            fmt.Sprintf("rc-fail-%s", n.ID),
				Category:      "External Dependency",
				Label:         fmt.Sprintf("Tool Execution Failed: %s", n.Label),
				Severity:      "critical",
				Confidence:    n.Confidence,
				EvidenceCount: len(n.Evidence),
				Status:        "active",
			}
		} else if n.Type == "Retry Strategy" {
			rcNode = RootCauseNode{
				ID:            fmt.Sprintf("rc-retry-%s", n.ID),
				Category:      "AI Agent",
				Label:         "Retry Storm",
				Severity:      "warning",
				Confidence:    n.Confidence,
				EvidenceCount: len(n.Evidence),
				Status:        "active",
			}
		} else if n.Type == "Document Selection" && n.Status == "error" {
			rcNode = RootCauseNode{
				ID:            fmt.Sprintf("rc-mem-%s", n.ID),
				Category:      "Infrastructure",
				Label:         "Vector DB Timeout",
				Severity:      "critical",
				Confidence:    n.Confidence,
				EvidenceCount: len(n.Evidence),
				Status:        "active",
			}
		}
		// If it's just a normal operational decision, it might not be a root cause failure.
		// For the sake of the graph, we might want to include the normal path leading up to the failure.
		
		if rcNode.ID == "" && n.Status == "error" {
			rcNode = RootCauseNode{
				ID:            fmt.Sprintf("rc-err-%s", n.ID),
				Category:      "Unknown",
				Label:         fmt.Sprintf("Unexpected Error: %s", n.Label),
				Severity:      "warning",
				Confidence:    n.Confidence,
				EvidenceCount: len(n.Evidence),
				Status:        "active",
			}
		}

		if rcNode.ID != "" {
			if !nodeMap[rcNode.ID] {
				nodes = append(nodes, rcNode)
				nodeMap[rcNode.ID] = true
			}
			decisionToCauseID[n.ID] = rcNode.ID
		}
	}

	// For the Hackathon Demo, if we detect no explicit errors but want to show a propagation:
	// We'll generate the edges based on the decision graph edges if both nodes are tracked.
	edgeMap := make(map[string]bool)
	for _, e := range dg.Edges {
		srcRcID, srcOk := decisionToCauseID[e.Source]
		tgtRcID, tgtOk := decisionToCauseID[e.Target]

		if srcOk && tgtOk && srcRcID != tgtRcID {
			edgeID := fmt.Sprintf("%s->%s", srcRcID, tgtRcID)
			if !edgeMap[edgeID] {
				edges = append(edges, RootCauseEdge{
					Source: srcRcID,
					Target: tgtRcID,
					Type:   "Caused",
				})
				edgeMap[edgeID] = true
			}
		}
	}

	return &RootCauseGraph{
		Nodes: nodes,
		Edges: edges,
	}
}
