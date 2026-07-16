package decision

import (
	"fmt"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/pkg/models"
)

// Engine is the Decision Reconstruction Engine.
type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

// Reconstruct builds a DecisionGraph from a BehaviorGraph.
func (e *Engine) Reconstruct(behaviorGraph *models.BehaviorGraph) (*models.DecisionGraph, error) {
	if behaviorGraph == nil {
		return nil, fmt.Errorf("nil behavior graph provided for decision reconstruction")
	}

	graph := &models.DecisionGraph{
		TraceID:   behaviorGraph.TraceID,
		AgentID:   behaviorGraph.AgentID,
		Nodes:     make([]models.DecisionNode, 0),
		Edges:     make([]models.DecisionEdge, 0),
		Timestamp: time.Now(),
	}

	// Map of behavior ID to decision node ID
	behToDecision := make(map[string]string)
	// Map of tool name to failed decision node ID to link retry decisions
	failedToolDecisions := make(map[string]string)

	orderCounter := 0

	for _, beh := range behaviorGraph.Nodes {
		decID := "dec_" + beh.BehaviorID
		behToDecision[beh.BehaviorID] = decID

		decType := "Inferred Choice"
		status := "Completed"
		inputs := make(map[string]string)
		for k, v := range beh.Metadata {
			inputs[k] = v
		}

		chosenOption := beh.Type
		var alternatives []string

		switch beh.Type {
		case "Tool Execution":
			decType = "Tool Selection"
			chosenOption = beh.Metadata["llm.tool_name"]
			if chosenOption == "" {
				chosenOption = beh.Metadata["span_name"]
			}
			alternatives = []string{"direct_llm_response", "memory_lookup"}
		case "Tool Failure":
			decType = "Recovery Strategy"
			chosenOption = "Trigger Retry / Fallback"
			status = "Partial"
			alternatives = []string{"escalate_to_human", "abort"}
		case "Tool Retry":
			decType = "Tool Retry"
			chosenOption = beh.Metadata["llm.tool_name"]
			if chosenOption == "" {
				chosenOption = beh.Metadata["span_name"]
			}
			alternatives = []string{"abort", "use_fallback_tool"}
		case "Prompt Expansion":
			decType = "Prompt Decision"
			chosenOption = "Expand Context"
			alternatives = []string{"compress_prompt", "truncate_history"}
		case "Retrieval Failure":
			decType = "Retrieval Decision"
			chosenOption = "Proceed Without Context / Retrieval Fallback"
			status = "Partial"
			alternatives = []string{"retry_query", "abort"}
		case "Memory Degradation":
			decType = "Memory Decision"
			chosenOption = "Ignore Missing Context / Bypass Memory"
			status = "Partial"
			alternatives = []string{"retry_lookup", "re-index"}
		case "LLM Call":
			decType = "Model Selection"
			chosenOption = beh.Metadata["llm.model"]
			if chosenOption == "" {
				chosenOption = "default-model"
			}
			alternatives = []string{"claude-3-5-sonnet", "gpt-4o-mini"}
		}

		node := models.DecisionNode{
			DecisionID:          decID,
			DecisionType:        decType,
			Timestamp:           beh.Timestamp,
			Actor:               beh.Actor,
			Confidence:          beh.Confidence,
			SupportingBehaviors: []string{beh.BehaviorID},
			EvidenceCount:       len(beh.ReplayEvents),
			Status:              status,
			Inputs:              inputs,
			ChosenOption:        chosenOption,
			Alternatives:        alternatives,
		}

		// Tool Retry linking
		if decType == "Tool Selection" || decType == "Tool Retry" {
			toolName := beh.Metadata["llm.tool_name"]
			if toolName == "" {
				toolName = beh.Metadata["span_name"]
			}
			if beh.Type == "Tool Failure" || beh.Type == "Tool Failure/Timeout" {
				failedToolDecisions[toolName] = decID
			} else if beh.Type == "Tool Retry" {
				if prevFailedDecID, ok := failedToolDecisions[toolName]; ok {
					// Add causal edge: Retry depends on / retries the failed decision
					graph.Edges = append(graph.Edges, models.DecisionEdge{
						Source:        prevFailedDecID,
						Destination:   decID,
						Relationship:  "Retries",
						Confidence:    1.0,
						TemporalOrder: orderCounter,
					})
					orderCounter++
					delete(failedToolDecisions, toolName)
				}
			}
		}

		graph.Nodes = append(graph.Nodes, node)
	}

	// Create edges mirroring behavior graph edges
	for _, edge := range behaviorGraph.Edges {
		srcDec, srcExists := behToDecision[edge.Source]
		dstDec, dstExists := behToDecision[edge.Destination]

		if srcExists && dstExists && srcDec != dstDec {
			// Check if edge already exists
			edgeExists := false
			for _, e := range graph.Edges {
				if e.Source == srcDec && e.Destination == dstDec {
					edgeExists = true
					break
				}
			}
			if !edgeExists {
				relType := edge.Type
				if relType == "Triggered" {
					relType = "Triggered"
				}
				graph.Edges = append(graph.Edges, models.DecisionEdge{
					Source:        srcDec,
					Destination:   dstDec,
					Relationship:  relType,
					Confidence:    edge.Confidence,
					TemporalOrder: orderCounter,
				})
				orderCounter++
			}
		}
	}

	return graph, nil
}
