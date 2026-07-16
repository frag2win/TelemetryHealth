package engine

import (
	"time"
)

// GetBenchmarkScenario returns deterministic replay events for judging.
func GetBenchmarkScenario(scenarioID string, tenantID string) []ReplayEvent {
	baseTime := time.Now().Add(-1 * time.Hour)
	
	switch scenarioID {
	case "prompt-explosion":
		return []ReplayEvent{
			{SpanID: "s1", ParentSpanID: "", ServiceName: "agent-service", OperationName: "planner", Status: "ok", StartTime: baseTime, EndTime: baseTime.Add(1 * time.Second), Attributes: map[string]interface{}{"llm.role": "Planner"}, TenantID: tenantID},
			{SpanID: "s2", ParentSpanID: "s1", ServiceName: "agent-service", OperationName: "tool-execution", Status: "ok", StartTime: baseTime.Add(100 * time.Millisecond), EndTime: baseTime.Add(200 * time.Millisecond), Attributes: map[string]interface{}{"tool.name": "FlightSearch"}, TenantID: tenantID},
			{SpanID: "s3", ParentSpanID: "s1", ServiceName: "agent-service", OperationName: "tool-execution", Status: "error", StartTime: baseTime.Add(300 * time.Millisecond), EndTime: baseTime.Add(400 * time.Millisecond), Attributes: map[string]interface{}{"tool.name": "BookingAPI"}, TenantID: tenantID},
			// Retry loop
			{SpanID: "s4", ParentSpanID: "s1", ServiceName: "agent-service", OperationName: "tool-execution", Status: "error", StartTime: baseTime.Add(500 * time.Millisecond), EndTime: baseTime.Add(600 * time.Millisecond), Attributes: map[string]interface{}{"tool.name": "BookingAPI"}, TenantID: tenantID},
			{SpanID: "s5", ParentSpanID: "s1", ServiceName: "agent-service", OperationName: "tool-execution", Status: "error", StartTime: baseTime.Add(700 * time.Millisecond), EndTime: baseTime.Add(800 * time.Millisecond), Attributes: map[string]interface{}{"tool.name": "BookingAPI"}, TenantID: tenantID},
		}
	
	case "vector-timeout":
		return []ReplayEvent{
			{SpanID: "v1", ParentSpanID: "", ServiceName: "rag-service", OperationName: "planner", Status: "ok", StartTime: baseTime, EndTime: baseTime.Add(2 * time.Second), Attributes: map[string]interface{}{"llm.role": "Summarizer"}, TenantID: tenantID},
			{SpanID: "v2", ParentSpanID: "v1", ServiceName: "rag-service", OperationName: "retrieval", Status: "error", StartTime: baseTime.Add(100 * time.Millisecond), EndTime: baseTime.Add(1900 * time.Millisecond), Attributes: map[string]interface{}{"vector.search": "true"}, TenantID: tenantID},
		}

	case "span-drop":
		// A trace where the parent span is missing but children exist
		return []ReplayEvent{
			{SpanID: "c1", ParentSpanID: "missing-parent", ServiceName: "inventory-service", OperationName: "check-stock", Status: "ok", StartTime: baseTime.Add(10 * time.Millisecond), EndTime: baseTime.Add(20 * time.Millisecond), Attributes: map[string]interface{}{}, TenantID: tenantID},
		}
	
	default:
		return nil
	}
}
