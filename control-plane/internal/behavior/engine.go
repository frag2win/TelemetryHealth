package behavior

import (
	"fmt"
	"strings"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/pkg/models"
)

// Engine is the Behavior Reconstruction Engine.
type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

// Reconstruct builds a BehaviorGraph from a slice of raw spans.
func (e *Engine) Reconstruct(traceID string, spans []models.SpanData) (*models.BehaviorGraph, error) {
	if len(spans) == 0 {
		return nil, fmt.Errorf("no spans provided for behavior reconstruction")
	}

	graph := &models.BehaviorGraph{
		TraceID:   traceID,
		AgentID:   "ai-agent", // Default if not found in attributes
		Nodes:     make([]models.BehaviorNode, 0),
		Edges:     make([]models.BehaviorEdge, 0),
		Timestamp: time.Now(),
	}

	// Try to find AgentID from attributes
	for _, span := range spans {
		if agentID, ok := span.Attributes["agent.id"]; ok {
			graph.AgentID = agentID
			break
		} else if agentID, ok := span.Attributes["agent_id"]; ok {
			graph.AgentID = agentID
			break
		}
	}

	// Map of span ID to behavior node ID for edge generation
	spanToBehavior := make(map[string]string)
	// Map of tool name to previously observed failed tool behavior node to detect retries
	failedTools := make(map[string]models.BehaviorNode)

	// Step 1: Create Behavior Nodes from spans
	for _, span := range spans {
		nodeID := "beh_" + span.SpanID
		spanToBehavior[span.SpanID] = nodeID

		actor := "Planner" // default
		behType := "Execution"

		nameLower := strings.ToLower(span.Name)
		if strings.Contains(nameLower, "llm") || strings.Contains(nameLower, "summarize") || span.Attributes["llm.model"] != "" {
			actor = "LLM"
			behType = "LLM Call"
		} else if strings.Contains(nameLower, "research") || strings.Contains(nameLower, "tool") || span.Attributes["llm.tool_name"] != "" {
			actor = "Tool"
			behType = "Tool Execution"
		} else if strings.Contains(nameLower, "retrieve") || strings.Contains(nameLower, "retriever") {
			actor = "Retriever"
			behType = "Retrieval"
		} else if strings.Contains(nameLower, "memory") {
			actor = "Memory"
			behType = "Memory Access"
		} else if strings.Contains(nameLower, "workflow") {
			actor = "Supervisor"
			behType = "Workflow"
		}

		// Detect failure patterns or metadata
		metadata := make(map[string]string)
		for k, v := range span.Attributes {
			metadata[k] = v
		}
		metadata["span_name"] = span.Name
		metadata["service_name"] = span.ServiceName

		// Determine if error or warning
		isError := span.StatusCode == "ERROR" || span.Attributes["llm.tool_call.error"] != "" || span.Attributes["error"] != ""
		if isError {
			behType = actor + " Failure"
			if span.Attributes["llm.tool_call.error"] != "" {
				metadata["failure_reason"] = span.Attributes["llm.tool_call.error"]
			} else {
				metadata["failure_reason"] = span.StatusMsg
			}
		}

		// Specific behavior checks
		// Prompt expansion
		if actor == "LLM" {
			promptExpanded := false
			for k := range span.Attributes {
				if strings.HasPrefix(k, "llm.prompt.raw_") {
					promptExpanded = true
					break
				}
			}
			if promptExpanded {
				behType = "Prompt Expansion"
			}
		}

		// Memory Degradation
		if actor == "Memory" && span.DurationNano > 500*1e6 { // > 500ms
			behType = "Memory Degradation"
			metadata["warning"] = "High latency detected during memory lookup"
		}

		// Retrieval Failure
		if actor == "Retriever" && (isError || span.Attributes["retriever.documents_count"] == "0") {
			behType = "Retrieval Failure"
		}

		confidence := 1.0
		if isError {
			confidence = 0.9 // slightly lower confidence on clean reconstruction if failed
		}

		node := models.BehaviorNode{
			BehaviorID:   nodeID,
			Type:         behType,
			Actor:        actor,
			DurationMs:   float64(span.DurationNano) / 1e6,
			Confidence:   confidence,
			ReplayEvents: []string{span.SpanID},
			Metadata:     metadata,
			Timestamp:    span.Timestamp,
		}

		// Tool Retry detection helper
		if actor == "Tool" {
			toolName := span.Attributes["llm.tool_name"]
			if toolName == "" {
				toolName = span.Name
			}
			if isError {
				failedTools[toolName] = node
			} else {
				// If we succeeded but previously failed this tool, it's a Tool Retry
				if prevFailed, ok := failedTools[toolName]; ok {
					node.Type = "Tool Retry"
					// Connect them
					graph.Edges = append(graph.Edges, models.BehaviorEdge{
						Source:      prevFailed.BehaviorID,
						Destination: node.BehaviorID,
						Type:        "Retries",
						Confidence:  1.0,
					})
					delete(failedTools, toolName) // resolved
				}
			}
		}

		graph.Nodes = append(graph.Nodes, node)
	}

	// Step 2: Establish relationships (edges) based on parent-child spans
	for _, span := range spans {
		if span.ParentSpanID == "" {
			continue
		}
		parentBehID, parentExists := spanToBehavior[span.ParentSpanID]
		childBehID, childExists := spanToBehavior[span.SpanID]

		if parentExists && childExists && parentBehID != childBehID {
			// Ensure edge doesn't already exist from retry
			edgeExists := false
			for _, edge := range graph.Edges {
				if edge.Source == parentBehID && edge.Destination == childBehID {
					edgeExists = true
					break
				}
			}
			if !edgeExists {
				graph.Edges = append(graph.Edges, models.BehaviorEdge{
					Source:      parentBehID,
					Destination: childBehID,
					Type:        "Triggered",
					Confidence:  1.0,
				})
			}
		}
	}

	return graph, nil
}
