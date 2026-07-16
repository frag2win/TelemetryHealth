package engine

import "fmt"

// BehaviorNode represents an interpreted behavioral step in an execution pipeline.
type BehaviorNode struct {
	ID     string
	Type   string // e.g. "planner", "retriever", "memory", "tool", "llm", "service"
	Label  string
	Status string // "healthy", "warning", "critical"
	Detail string
}

// BehaviorGraph is a semantic graph of interpreted behaviors.
type BehaviorGraph struct {
	Nodes []BehaviorNode
	Edges []BehaviorEdge
}

type BehaviorEdge struct {
	Source string
	Target string
}

// BehaviorBuilder abstracts the logic of converting raw ReplayEvents into a semantic BehaviorGraph.
type BehaviorBuilder interface {
	Build(events []ReplayEvent) (*BehaviorGraph, error)
}

// DefaultBehaviorBuilder implements BehaviorBuilder using the BehaviorClassifier.
type DefaultBehaviorBuilder struct{}

func NewBehaviorBuilder() *DefaultBehaviorBuilder {
	return &DefaultBehaviorBuilder{}
}

func (b *DefaultBehaviorBuilder) Build(events []ReplayEvent) (*BehaviorGraph, error) {
	nodes := []BehaviorNode{}
	edges := []BehaviorEdge{}
	nodeMap := make(map[string]bool)

	// spanMap maps spanID to its generated behavior ID
	spanToBehaviorID := make(map[string]string)
	// spanToParent maps spanID to parentSpanID for edge resolution
	spanToParent := make(map[string]string)

	for _, e := range events {
		node := ClassifyBehavior(e)
		
		// If classifier returned a valid behavior, register it
		if node.ID != "" {
			spanToBehaviorID[e.SpanID] = node.ID
			spanToParent[e.SpanID] = e.ParentSpanID

			if !nodeMap[node.ID] {
				nodes = append(nodes, node)
				nodeMap[node.ID] = true
			}
		}
	}

	// Resolve edges
	edgeMap := make(map[string]bool)
	for spanID, behaviorID := range spanToBehaviorID {
		parentSpanID := spanToParent[spanID]
		if parentSpanID != "" && parentSpanID != "0000000000000000" {
			if parentBehaviorID, ok := spanToBehaviorID[parentSpanID]; ok {
				if parentBehaviorID != behaviorID {
					edgeID := fmt.Sprintf("%s->%s", parentBehaviorID, behaviorID)
					if !edgeMap[edgeID] {
						edges = append(edges, BehaviorEdge{
							Source: parentBehaviorID,
							Target: behaviorID,
						})
						edgeMap[edgeID] = true
					}
				}
			}
		}
	}

	return &BehaviorGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// ClassifyBehavior determines the semantic behavior of a raw replay event based on its attributes.
func ClassifyBehavior(span ReplayEvent) BehaviorNode {
	// 1. LLM Operations (Planner, Summarizer, etc.)
	if role, ok := span.Attributes["llm.role"].(string); ok {
		return BehaviorNode{
			ID:     fmt.Sprintf("llm-%s", role),
			Type:   "planner", // could be dynamic based on role
			Label:  fmt.Sprintf("LLM %s", role),
			Status: span.Status,
			Detail: span.ServiceName,
		}
	}

	// 2. Tool Executions
	if tool, ok := span.Attributes["tool.name"].(string); ok {
		return BehaviorNode{
			ID:     fmt.Sprintf("tool-%s", tool),
			Type:   "tool",
			Label:  fmt.Sprintf("Tool: %s", tool),
			Status: span.Status,
			Detail: span.ServiceName,
		}
	}

	// 3. Retriever / Vector Search
	if _, ok := span.Attributes["vector.search"]; ok {
		return BehaviorNode{
			ID:     "retriever",
			Type:   "retriever",
			Label:  "Vector Retriever",
			Status: span.Status,
			Detail: span.ServiceName,
		}
	}

	// 4. Memory Operations
	if op, ok := span.Attributes["memory.operation"].(string); ok {
		return BehaviorNode{
			ID:     fmt.Sprintf("memory-%s", op),
			Type:   "memory",
			Label:  fmt.Sprintf("Memory %s", op),
			Status: span.Status,
			Detail: span.ServiceName,
		}
	}

	// Fallback to Service/Operation Name if no semantic attributes match
	id := span.OperationName
	if id == "" {
		id = span.ServiceName
	}
	
	if id == "" {
		return BehaviorNode{}
	}

	return BehaviorNode{
		ID:     id,
		Type:   "service",
		Label:  id,
		Status: span.Status,
		Detail: span.ServiceName,
	}
}
