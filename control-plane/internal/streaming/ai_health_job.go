package streaming

import (
	"sync"

	"go.uber.org/zap"
)

// AIAgentHealth struct to track AI specific metrics
type AIAgentHealth struct {
	TotalTokens     int64
	ToolCalls       int64
	ToolFailures    int64
}

// AIAgentHealthJob tracks metrics specific to AI agents such as token usage and tool failures.
type AIAgentHealthJob struct {
	logger *zap.Logger
	mu     sync.RWMutex
	
	// Map from TenantID to their AI agent health metrics
	tenantMetrics map[string]*AIAgentHealth
}

func NewAIAgentHealthJob(logger *zap.Logger) *AIAgentHealthJob {
	return &AIAgentHealthJob{
		logger:        logger,
		tenantMetrics: make(map[string]*AIAgentHealth),
	}
}

// ProcessSpan extracts AI-specific attributes to calculate health metrics
func (j *AIAgentHealthJob) ProcessSpan(tenantID string, attributes map[string]interface{}) {
	j.mu.Lock()
	defer j.mu.Unlock()

	health, exists := j.tenantMetrics[tenantID]
	if !exists {
		health = &AIAgentHealth{}
		j.tenantMetrics[tenantID] = health
	}

	// Check for token usage
	if tokenUsage, ok := attributes["llm.token_usage"]; ok {
		// Attempt to parse token usage as int64, depending on the type
		if val, isFloat := tokenUsage.(float64); isFloat {
			health.TotalTokens += int64(val)
		} else if val, isInt := tokenUsage.(int64); isInt {
			health.TotalTokens += val
		}
	}

	// Check for tool calls
	if _, ok := attributes["llm.tool_name"]; ok {
		health.ToolCalls++
		
		// Check if the tool failed
		if _, errOk := attributes["llm.tool_call.error"]; errOk {
			health.ToolFailures++
		}
	}
}

// GetMetrics returns the calculated AI agent health metrics for a tenant
func (j *AIAgentHealthJob) GetMetrics(tenantID string) *AIAgentHealth {
	j.mu.RLock()
	defer j.mu.RUnlock()

	if health, exists := j.tenantMetrics[tenantID]; exists {
		// Return a copy to avoid race conditions
		return &AIAgentHealth{
			TotalTokens:  health.TotalTokens,
			ToolCalls:    health.ToolCalls,
			ToolFailures: health.ToolFailures,
		}
	}
	
	return &AIAgentHealth{}
}
