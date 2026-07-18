package signoz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/engine"
	"go.uber.org/zap"
)

type QueryClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewQueryClient(logger *zap.Logger) *QueryClient {
	baseURL := os.Getenv("SIGNOZ_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3301" // Default frontend dev port
	}
	apiKey := os.Getenv("SIGNOZ_API_KEY")

	return &QueryClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

// QueryRange calls SigNoz v3 Query API to query metrics/traces/logs.
func (c *QueryClient) QueryRange(ctx context.Context, payload interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v3/query_range", c.baseURL)

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("signoz client marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("signoz client new request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("signoz-access-token", c.apiKey)
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("signoz client execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("signoz client returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("signoz client decode response: %w", err)
	}

	return result, nil
}

// Ping checks if the SigNoz query-service endpoint is reachable.
func (c *QueryClient) Ping(ctx context.Context) error {
	// Ping the health or version endpoint if available, otherwise hit GET /api/v3/query_range with empty request or config endpoint
	url := fmt.Sprintf("%s/api/v1/version", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("signoz-access-token", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("unhealthy status: %d", resp.StatusCode)
	}
	return nil
}

// GetReplay fetches a single trace from the official SigNoz API for agent trace reconstruction
func (c *QueryClient) GetReplay(ctx context.Context, tenantID, traceID string) ([]engine.ReplayEvent, error) {
	url := fmt.Sprintf("%s/api/v1/traces/%s", c.baseURL, traceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if c.apiKey != "" {
		req.Header.Set("signoz-access-token", c.apiKey)
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trace from signoz: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("signoz trace api returned status: %d", resp.StatusCode)
	}

	var payload struct {
		Status string `json:"status"`
		Data   []struct {
			TraceID           string                 `json:"traceID"`
			SpanID            string                 `json:"spanID"`
			ParentSpanID      string                 `json:"parentSpanID"`
			ServiceName       string                 `json:"serviceName"`
			Name              string                 `json:"name"`
			StartTimeUnixNano int64                  `json:"startTimeUnixNano"`
			EndTimeUnixNano   int64                  `json:"endTimeUnixNano"`
			StatusCode        int64                  `json:"statusCode"`
			StringTagMap      map[string]interface{} `json:"stringTagMap"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode trace response: %w", err)
	}

	var events []engine.ReplayEvent
	for _, span := range payload.Data {
		status := "ok"
		if span.StatusCode == 2 {
			status = "error"
		}
		
		events = append(events, engine.ReplayEvent{
			TraceID:       span.TraceID,
			SpanID:        span.SpanID,
			ParentSpanID:  span.ParentSpanID,
			ServiceName:   span.ServiceName,
			OperationName: span.Name,
			StartTime:     time.Unix(0, span.StartTimeUnixNano),
			EndTime:       time.Unix(0, span.EndTimeUnixNano),
			Status:        status,
			Attributes:    span.StringTagMap,
		})
	}
	
	if len(events) == 0 {
		return nil, fmt.Errorf("trace not found in signoz")
	}

	return events, nil
}

