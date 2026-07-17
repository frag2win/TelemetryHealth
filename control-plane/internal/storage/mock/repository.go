package mock

import (
	"context"
	"strings"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/engine"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
	"github.com/frag2win/TelemetryHealth/control-plane/pkg/models"
)

type MockRepository struct{}

func NewRepository() *MockRepository {
	return &MockRepository{}
}

// Ensure interface compliance
var _ storage.HealthRepository = (*MockRepository)(nil)
var _ engine.ReplayRepository = (*MockRepository)(nil)

// HealthRepository Implementation
func (m *MockRepository) QueryHealthMetrics(ctx context.Context, tenantID string) (*storage.HealthMetrics, error) {
	return &storage.HealthMetrics{
		TenantID: tenantID,
		CompositeScore: 85.0,
		CardinalityMax: 50,
		OrphanCount: 0,
		PreviousOrphanCount: 0,
		ActiveServices: 3,
		Window: time.Now(),
	}, nil
}

func (m *MockRepository) QueryAgentTraces(ctx context.Context) ([]storage.AgentTrace, error) {
	return []storage.AgentTrace{}, nil
}

func (m *MockRepository) GetTenantWeights(ctx context.Context, tenantID string) (telemetry.TenantWeights, error) {
	return telemetry.TenantWeights{
		CardinalityWeight: 0.20,
		OrphanWeight: 0.30,
		CoverageWeight: 0.50,
	}, nil
}

func (m *MockRepository) SaveTenantConfig(ctx context.Context, tenantID string, weights telemetry.TenantWeights) error {
	return nil
}

func (m *MockRepository) LogRemediationEvent(ctx context.Context, tenantID string, issueType string, yamlConfig string, validated, applied bool, actorID, actorRole, sourceIP, action, resourceID string) error {
	return nil
}

func (m *MockRepository) QuerySpansByTraceID(ctx context.Context, traceID string) ([]models.SpanData, error) {
	return generateMockSpans(traceID), nil
}

// ReplayRepository Implementation
func (m *MockRepository) GetReplay(ctx context.Context, tenantID, traceID string) ([]engine.ReplayEvent, error) {
	spans := generateMockSpans(traceID)
	var events []engine.ReplayEvent
	for _, span := range spans {
		attrs := make(map[string]interface{})
		for k, v := range span.Attributes {
			attrs[k] = v
		}
		events = append(events, engine.ReplayEvent{
			TraceID:       span.TraceID,
			SpanID:        span.SpanID,
			ParentSpanID:  span.ParentSpanID,
			ServiceName:   span.ServiceName,
			OperationName: span.Name,
			StartTime:     span.Timestamp,
			EndTime:       span.Timestamp.Add(time.Duration(span.DurationNano)),
			Status:        span.StatusCode,
			Attributes:    attrs,
		})
	}
	return events, nil
}

func (m *MockRepository) GetRecentReplays(ctx context.Context, tenantID string, limit int) ([]engine.ReplayEvent, error) {
	// For mock, just return one mocked trace
	return m.GetReplay(ctx, tenantID, "mock-recent-trace")
}

func generateMockSpans(traceID string) []models.SpanData {
	now := time.Now()

	if strings.Contains(traceID, "992") || strings.Contains(traceID, "fail") || strings.Contains(traceID, "timeout") {
		return []models.SpanData{
			{
				TraceID:      traceID,
				SpanID:       "span-root",
				ParentSpanID: "",
				ServiceName:  "ai-agent",
				Name:         "agent.workflow",
				DurationNano: int64(3000 * time.Millisecond),
				Timestamp:    now,
				Attributes: map[string]string{
					"workflow.topic": "Observability best practices",
				},
				StatusCode: "ERROR",
			},
			{
				TraceID:      traceID,
				SpanID:       "span-tool-fail",
				ParentSpanID: "span-root",
				ServiceName:  "ai-agent",
				Name:         "agent.research",
				DurationNano: int64(1000 * time.Millisecond),
				Timestamp:    now.Add(100 * time.Millisecond),
				Attributes: map[string]string{
					"llm.tool_name":       "web_search",
					"llm.tool_call.error": "TimeoutError: connection refused",
				},
				StatusCode: "ERROR",
			},
			{
				TraceID:      traceID,
				SpanID:       "span-tool-retry",
				ParentSpanID: "span-root",
				ServiceName:  "ai-agent",
				Name:         "agent.research",
				DurationNano: int64(800 * time.Millisecond),
				Timestamp:    now.Add(1200 * time.Millisecond),
				Attributes: map[string]string{
					"llm.tool_name": "web_search",
				},
				StatusCode: "OK",
			},
			{
				TraceID:      traceID,
				SpanID:       "span-llm-summarize",
				ParentSpanID: "span-root",
				ServiceName:  "ai-agent",
				Name:         "agent.summarize",
				DurationNano: int64(1200 * time.Millisecond),
				Timestamp:    now.Add(2100 * time.Millisecond),
				Attributes: map[string]string{
					"llm.model":           "gpt-4o",
					"llm.token_usage":     "1250",
					"llm.prompt.raw_abcd": "Information about Observability best practices",
				},
				StatusCode: "OK",
			},
		}
	}

	if strings.Contains(traceID, "token") || strings.Contains(traceID, "limit") {
		return []models.SpanData{
			{
				TraceID:      traceID,
				SpanID:       "span-root",
				ParentSpanID: "",
				ServiceName:  "ai-agent",
				Name:         "agent.workflow",
				DurationNano: int64(2200 * time.Millisecond),
				Timestamp:    now,
				Attributes: map[string]string{
					"workflow.topic": "Vector databases",
				},
				StatusCode: "OK",
			},
			{
				TraceID:      traceID,
				SpanID:       "span-tool",
				ParentSpanID: "span-root",
				ServiceName:  "ai-agent",
				Name:         "agent.research",
				DurationNano: int64(900 * time.Millisecond),
				Timestamp:    now.Add(100 * time.Millisecond),
				Attributes: map[string]string{
					"llm.tool_name": "web_search",
				},
				StatusCode: "OK",
			},
			{
				TraceID:      traceID,
				SpanID:       "span-llm-heavy",
				ParentSpanID: "span-root",
				ServiceName:  "ai-agent",
				Name:         "agent.summarize",
				DurationNano: int64(1200 * time.Millisecond),
				Timestamp:    now.Add(1000 * time.Millisecond),
				Attributes: map[string]string{
					"llm.model":       "gpt-4o",
					"llm.token_usage": "5200",
				},
				StatusCode: "OK",
			},
		}
	}

	if strings.Contains(traceID, "retrieve") || strings.Contains(traceID, "collapse") {
		return []models.SpanData{
			{
				TraceID:      traceID,
				SpanID:       "span-root",
				ParentSpanID: "",
				ServiceName:  "ai-agent",
				Name:         "agent.workflow",
				DurationNano: int64(1500 * time.Millisecond),
				Timestamp:    now,
				Attributes:   map[string]string{"workflow.topic": "nonexistent term"},
				StatusCode:   "OK",
			},
			{
				TraceID:      traceID,
				SpanID:       "span-retriever",
				ParentSpanID: "span-root",
				ServiceName:  "ai-agent",
				Name:         "agent.retrieve",
				DurationNano: int64(300 * time.Millisecond),
				Timestamp:    now.Add(100 * time.Millisecond),
				Attributes: map[string]string{
					"retriever.documents_count": "0",
				},
				StatusCode: "OK",
			},
			{
				TraceID:      traceID,
				SpanID:       "span-llm",
				ParentSpanID: "span-root",
				ServiceName:  "ai-agent",
				Name:         "agent.summarize",
				DurationNano: int64(1000 * time.Millisecond),
				Timestamp:    now.Add(400 * time.Millisecond),
				Attributes: map[string]string{
					"llm.model": "gpt-4o",
				},
				StatusCode: "OK",
			},
		}
	}

	return []models.SpanData{
		{
			TraceID:      traceID,
			SpanID:       "span-root",
			ParentSpanID: "",
			ServiceName:  "ai-agent",
			Name:         "agent.workflow",
			DurationNano: int64(2500 * time.Millisecond),
			Timestamp:    now,
			Attributes: map[string]string{
				"workflow.topic": "LLM cost optimization",
			},
			StatusCode: "OK",
		},
		{
			TraceID:      traceID,
			SpanID:       "span-tool",
			ParentSpanID: "span-root",
			ServiceName:  "ai-agent",
			Name:         "agent.research",
			DurationNano: int64(1100 * time.Millisecond),
			Timestamp:    now.Add(100 * time.Millisecond),
			Attributes: map[string]string{
				"llm.tool_name": "web_search",
			},
			StatusCode: "OK",
		},
		{
			TraceID:      traceID,
			SpanID:       "span-llm",
			ParentSpanID: "span-root",
			ServiceName:  "ai-agent",
			Name:         "agent.summarize",
			DurationNano: int64(1200 * time.Millisecond),
			Timestamp:    now.Add(1250 * time.Millisecond),
			Attributes: map[string]string{
				"llm.model":       "gpt-4o",
				"llm.token_usage": "980",
			},
			StatusCode: "OK",
		},
	}
}
