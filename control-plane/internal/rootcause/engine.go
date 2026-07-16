package rootcause

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/pkg/models"
)

// Engine is the Root Cause Intelligence Engine.
type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

// Analyze evaluates the behavior and decision graphs to identify the probable root cause.
func (e *Engine) Analyze(behGraph *models.BehaviorGraph, decGraph *models.DecisionGraph) (*models.RootCause, error) {
	if behGraph == nil || decGraph == nil {
		return nil, fmt.Errorf("both behavior and decision graphs are required for root cause analysis")
	}

	rc := &models.RootCause{
		TraceID:         behGraph.TraceID,
		AgentID:         behGraph.AgentID,
		FailureType:     "none",
		Description:     "No anomalies or failures detected in the execution path.",
		EvidenceSpanIDs: make([]string, 0),
		Severity:        "Low",
		Confidence:      1.0,
		Timestamp:       time.Now(),
		Status:          "Unknown",
	}

	// 1. Scan behavior nodes for failures/anomalies
	var toolFailureNode *models.BehaviorNode
	var tokenExceededNode *models.BehaviorNode
	var retrievalFailureNode *models.BehaviorNode
	var latencyAnomalyNode *models.BehaviorNode
	var generalErrorNode *models.BehaviorNode

	for _, node := range behGraph.Nodes {
		nodeTypeLower := strings.ToLower(node.Type)

		// Check for tool failures / timeouts
		if strings.Contains(nodeTypeLower, "tool failure") || strings.Contains(nodeTypeLower, "tool timeout") {
			toolFailureNode = &node
		}

		// Check for retrieval failure
		if strings.Contains(nodeTypeLower, "retrieval failure") {
			retrievalFailureNode = &node
		}

		// Check for token usage issue
		if tokenUsageStr, ok := node.Metadata["llm.token_usage"]; ok {
			if val, err := strconv.Atoi(tokenUsageStr); err == nil && val >= 4000 {
				tokenExceededNode = &node
			}
		}

		// Check for latency anomaly (e.g. tool execution taking > 2 seconds or memory degradation)
		if node.DurationMs > 2000 {
			latencyAnomalyNode = &node
		}

		// General error status in span
		if strings.Contains(nodeTypeLower, "failure") || strings.Contains(nodeTypeLower, "error") {
			if generalErrorNode == nil {
				generalErrorNode = &node
			}
		}
	}

	// Determine the primary root cause based on priority
	if toolFailureNode != nil {
		rc.FailureType = "tool_timeout"
		reason := toolFailureNode.Metadata["failure_reason"]
		if reason == "" {
			reason = "connection refused or deadline exceeded"
		}
		toolName := toolFailureNode.Metadata["llm.tool_name"]
		if toolName == "" {
			toolName = "external_tool"
		}
		rc.Description = fmt.Sprintf("Tool call failure detected on tool '%s' due to: %s.", toolName, reason)
		rc.EvidenceSpanIDs = append(rc.EvidenceSpanIDs, toolFailureNode.ReplayEvents...)
		rc.Severity = "Critical"
		rc.Confidence = toolFailureNode.Confidence
		rc.Status = "Detected"
	} else if tokenExceededNode != nil {
		rc.FailureType = "token_limit"
		tokenUsage := tokenExceededNode.Metadata["llm.token_usage"]
		modelName := tokenExceededNode.Metadata["llm.model"]
		rc.Description = fmt.Sprintf("LLM agent execution path hit token burn rate/limit threshold on model '%s'. Used %s tokens.", modelName, tokenUsage)
		rc.EvidenceSpanIDs = append(rc.EvidenceSpanIDs, tokenExceededNode.ReplayEvents...)
		rc.Severity = "High"
		rc.Confidence = 0.95
		rc.Status = "Detected"
	} else if retrievalFailureNode != nil {
		rc.FailureType = "retrieval_collapse"
		rc.Description = "Retrieval collapse: Retriever returned zero documents or failed, leading to lack of query context."
		rc.EvidenceSpanIDs = append(rc.EvidenceSpanIDs, retrievalFailureNode.ReplayEvents...)
		rc.Severity = "Medium"
		rc.Confidence = 0.85
		rc.Status = "Detected"
	} else if latencyAnomalyNode != nil {
		rc.FailureType = "latency"
		rc.Description = fmt.Sprintf("High latency anomaly detected: node '%s' took %.1f ms to execute.", latencyAnomalyNode.Type, latencyAnomalyNode.DurationMs)
		rc.EvidenceSpanIDs = append(rc.EvidenceSpanIDs, latencyAnomalyNode.ReplayEvents...)
		rc.Severity = "Medium"
		rc.Confidence = 0.90
		rc.Status = "Detected"
	} else if generalErrorNode != nil {
		rc.FailureType = "error"
		reason := generalErrorNode.Metadata["failure_reason"]
		if reason == "" {
			reason = "internal server error or execution failure"
		}
		rc.Description = fmt.Sprintf("Execution failed due to: %s.", reason)
		rc.EvidenceSpanIDs = append(rc.EvidenceSpanIDs, generalErrorNode.ReplayEvents...)
		rc.Severity = "High"
		rc.Confidence = generalErrorNode.Confidence
		rc.Status = "Detected"
	}

	return rc, nil
}
