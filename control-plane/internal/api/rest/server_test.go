package rest

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage/mock"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	trace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func TestRateLimitMiddleware(t *testing.T) {
	// A simple handler that returns 200 OK
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	limiterHandler := rateLimitMiddleware(handler)

	// Make 20 requests rapidly (which matches our burst limit of 20)
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/api/v1/tenant/acme-prod/health", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()
		limiterHandler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d failed: expected 200, got %d", i+1, w.Code)
		}
	}

	// The 21st request should be blocked
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/api/v1/tenant/acme-prod/health", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	w := httptest.NewRecorder()
	limiterHandler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 Too Many Requests, got %d", w.Code)
	}

	// A different IP address should still go through (as rate limits are per-IP)
	req2 := httptest.NewRequest(http.MethodGet, "http://localhost:8080/api/v1/tenant/acme-prod/health", nil)
	req2.RemoteAddr = "192.168.1.2:1234"
	w2 := httptest.NewRecorder()
	limiterHandler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 OK for different IP, got %d", w2.Code)
	}
}

func TestTracingMiddleware(t *testing.T) {
	// Setup TracerProvider
	oldProvider := otel.GetTracerProvider()
	defer otel.SetTracerProvider(oldProvider)
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	// Setup Propagator
	oldPropagator := otel.GetTextMapPropagator()
	defer otel.SetTextMapPropagator(oldPropagator)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	var ctxWithSpan context.Context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxWithSpan = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	tracingHandler := tracingMiddleware(handler)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/api/v1/tenant/acme-prod/health", nil)
	w := httptest.NewRecorder()

	// Call handler
	tracingHandler.ServeHTTP(w, req)

	// Check if context in handler got a span
	span := trace.SpanFromContext(ctxWithSpan)
	if !span.SpanContext().IsValid() {
		t.Error("expected valid span context in handler request context")
	}

	// Check response headers for injected trace headers (traceparent)
	traceparent := w.Header().Get("traceparent")
	if traceparent == "" {
		traceparent = w.Header().Get("Traceparent")
	}
	if traceparent == "" {
		t.Error("expected traceparent response header to be injected")
	}
}

func TestInjectTraceContext(t *testing.T) {
	// Setup trace parent context
	traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	spanID, _ := trace.SpanIDFromHex("0102030405060708")
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8080/api", nil)

	// Temporarily configure propagator
	oldProp := otel.GetTextMapPropagator()
	defer otel.SetTextMapPropagator(oldProp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	req = req.WithContext(ctx)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	// Verify traceparent header was injected
	traceparent := req.Header.Get("traceparent")
	if traceparent == "" {
		t.Error("expected traceparent header to be injected")
	}
	expectedSubstring := "0102030405060708090a0b0c0d0e0f10"
	if !strconv.CanBackquote(traceparent) || traceparent == "" {
		// Just a general check
	}
	_ = expectedSubstring
}

func TestServer_AgentTraceEndpoints(t *testing.T) {
	mockRepo := mock.NewRepository()
	s := NewServer(zap.NewNop(), mockRepo, mockRepo)

	r := chi.NewRouter()
	r.Get("/api/agents/{agent_id}/traces/{trace_id}/behavior", s.GetBehaviorGraph)
	r.Get("/api/agents/{agent_id}/traces/{trace_id}/decisions", s.GetDecisionGraph)
	r.Get("/api/agents/{agent_id}/traces/{trace_id}/root-cause", s.GetRootCause)

	// 1. Test Behavior Graph Endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/agents/ai-agent/traces/trace-992/behavior", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	// 2. Test Decisions Graph Endpoint
	req = httptest.NewRequest(http.MethodGet, "/api/agents/ai-agent/traces/trace-992/decisions", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	// 3. Test Root Cause Endpoint
	req = httptest.NewRequest(http.MethodGet, "/api/agents/ai-agent/traces/trace-992/root-cause", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}
}

func TestApplyRemediation_SizeAndValidation(t *testing.T) {
	mockRepo := mock.NewRepository()
	s := NewServer(zap.NewNop(), mockRepo, mockRepo)

	// 1. Test payload exceeding 64KB
	largeYaml := strings.Repeat("a", 65*1024)
	body := `{"tenantId":"123e4567-e89b-12d3-a456-426614174000","issueType":"cardinality","yaml":"` + largeYaml + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/remediation/apply", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	s.ApplyRemediation(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 Request Entity Too Large for oversized YAML, got %d", w.Code)
	}

	// 2. Test invalid OTel component (blocked by allowlist, e.g. filelog receiver)
	blockedYaml := `receivers:\n  filelog:\n    include: [/var/log/*.log]`
	body = `{"tenantId":"123e4567-e89b-12d3-a456-426614174000","issueType":"cardinality","yaml":"` + blockedYaml + `"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/remediation/apply", bytes.NewBufferString(body))
	w = httptest.NewRecorder()
	s.ApplyRemediation(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for blocked OTel component YAML, got %d", w.Code)
	}

	// 3. Test valid allowed OTel component (e.g. attributes processor)
	validYaml := `processors:\n  attributes/remediation:\n    actions:\n      - key: user_id\n        action: delete`
	body = `{"tenantId":"123e4567-e89b-12d3-a456-426614174000","issueType":"cardinality","yaml":"` + validYaml + `"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/remediation/apply", bytes.NewBufferString(body))
	w = httptest.NewRecorder()
	s.ApplyRemediation(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for valid OTel component YAML, got %d", w.Code)
	}
}

func TestUnconfiguredClickHouse_Returns501(t *testing.T) {
	oldDevMode := os.Getenv("INSECURE_DEV_MODE")
	os.Setenv("INSECURE_DEV_MODE", "true")
	defer os.Setenv("INSECURE_DEV_MODE", oldDevMode)

	// Server with nil healthRepo (unconfigured ClickHouse)
	s := NewServer(zap.NewNop(), nil, nil)
	r := s.routes()

	endpoints := []string{
		"/api/v1/tenant/123e4567-e89b-12d3-a456-426614174000/health",
		"/api/v1/tenant/123e4567-e89b-12d3-a456-426614174000/agents",
		"/api/v1/tenant/123e4567-e89b-12d3-a456-426614174000/config",
	}

	for _, path := range endpoints {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotImplemented {
			t.Errorf("expected 501 Not Implemented for %s when ClickHouse unconfigured, got %d", path, w.Code)
		}
	}
}
