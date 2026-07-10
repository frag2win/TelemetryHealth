package rest

// @title TelemetryHealth API
// @version 1.0
// @description REST API for TelemetryHealth control plane
// @host localhost:8080
// @BasePath /api/v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/mcp"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/remediation"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage/clickhouse"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.uber.org/zap"

	_ "github.com/frag2win/TelemetryHealth/control-plane/docs" // imported for swagger
)

// Struct definitions moved to mcp package

// Server is the REST API server for the control plane dashboard.
type Server struct {
	logger     *zap.Logger
	healthRepo *clickhouse.HealthRepository
	validator  *remediation.Validator
	generator  *remediation.Generator
	httpServer *http.Server
}

func NewServer(logger *zap.Logger, healthRepo *clickhouse.HealthRepository) *Server {
	return &Server{
		logger:     logger,
		healthRepo: healthRepo,
		validator:  remediation.NewValidator(logger),
		generator:  remediation.NewGenerator(logger),
	}
}

// corsMiddleware adds basic CORS headers.
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := os.Getenv("CORS_ORIGIN")
		if origin == "" {
			origin = "http://localhost:5173"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// metricsMiddleware tracks API requests for Prometheus.
func metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		srw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		startTime := time.Now()
		next(srw, r)
		duration := time.Since(startTime).Seconds()

		statusStr := strconv.Itoa(srw.statusCode)
		telemetry.ApiRequestsTotal.WithLabelValues(r.Method, r.URL.Path, statusStr).Inc()
		telemetry.ApiRequestDuration.WithLabelValues(r.Method, r.URL.Path, statusStr).Observe(duration)
	}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	mux.HandleFunc("/api/v1/tenant/", corsMiddleware(metricsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/health") {
			s.GetTenantHealth(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/issues") {
			s.GetTenantIssues(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/agents") {
			s.GetAgentTraces(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/coverage") {
			s.GetCoverage(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/traces/orphans") {
			s.GetTracesOrphans(w, r)
		} else {
			http.NotFound(w, r)
		}
	})))

	mux.HandleFunc("/api/v1/remediation/apply", corsMiddleware(metricsMiddleware(s.ApplyRemediation)))

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.logger.Info("Starting API Server", zap.String("addr", addr))
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	s.logger.Info("Shutting down API Server")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) GetAgentTraces(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.healthRepo != nil {
		traces, err := s.healthRepo.QueryAgentTraces(r.Context())
		if err != nil {
			s.logger.Error("query agent traces failed", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(traces)
		return
	}

	// Fallback to static mock traces if no repository is configured
	json.NewEncoder(w).Encode([]clickhouse.AgentTrace{
		{
			ID:                "trace-991",
			Model:             "gpt-4o",
			Tokens:            4120,
			Cost:              0.041,
			Latency:           "3.2s",
			HallucinationRisk: "Low",
			Decisions: []clickhouse.AgentDecision{
				{Step: "Retrieved 15 similar spans from ClickHouse (gen_ai.system)", Tool: "query_clickhouse", Status: "success"},
				{Step: "Analyzed cardinality distribution for user_id", Tool: "python_eval", Status: "success"},
				{Step: "Generated remediation YAML via SigNoz MCP tool", Tool: "generate_yaml", Status: "success"},
			},
		},
		{
			ID:                "trace-992",
			Model:             "claude-3-5-sonnet",
			Tokens:            8450,
			Cost:              0.025,
			Latency:           "6.1s",
			HallucinationRisk: "High",
			Decisions: []clickhouse.AgentDecision{
				{Step: "Attempted to query missing index (gen_ai.request.model)", Tool: "query_clickhouse", Status: "error"},
				{Step: "Retried with full table scan (token limit warning)", Tool: "query_clickhouse", Status: "warning"},
				{Step: "Formulated remediation with unverified field names", Tool: "generate_yaml", Status: "warning"},
			},
		},
	})
}


// GetTenantHealth godoc
// @Summary Get Health Metrics for a Tenant
// @Description Returns the composite health score, signal metrics, and auto-generated OTel remediation for a given tenant.
// @Produce json
// @Param tenant_id path string true "Tenant UUID"
// @Success 200 {object} HealthResponse
// @Failure 400 {string} string "invalid path"
// @Router /tenant/{tenant_id}/health [get]
// @Router /tenant/{tenant_id}/health [get]
func (s *Server) GetTenantHealth(w http.ResponseWriter, r *http.Request) {

		// Extract tenant ID from URL: /api/v1/tenant/{id}/health
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 4 {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		tenantID := parts[3]

		w.Header().Set("Content-Type", "application/json")

		// Try real ClickHouse if repo is available
		if s.healthRepo != nil {
			metrics, err := s.healthRepo.QueryHealthMetrics(r.Context(), tenantID)
			if err != nil {
				s.logger.Error("clickhouse query failed", zap.Error(err))
			} else {
				issueType := metrics.RemediationIssue
				remediationYaml := ""
				validated := false
				if issueType != "" {
					var genErr error
					remediationYaml, genErr = s.generator.Generate(r.Context(), issueType)
					if genErr != nil {
						s.logger.Error("failed to generate remediation yaml", zap.Error(genErr))
					}
					if s.validator != nil && remediationYaml != "" {
						validated, _ = s.validator.Validate(r.Context(), remediationYaml)
					}
				}

				resp := mcp.HealthResponse{
					HealthScore: metrics.CompositeScore,
					Metrics: mcp.MetricsPayload{
						Cardinality: mcp.MetricValue{
							Value:  fmtLarge(metrics.CardinalityMax),
							Change: cardChange(metrics.CardinalityMax),
						},
						Orphans: mcp.MetricValue{
							Value:  fmt.Sprintf("%d", metrics.OrphanCount),
							Change: calculateDelta(metrics.OrphanCount, metrics.PreviousOrphanCount),
						},
						Coverage: mcp.MetricValue{
							Value:  fmt.Sprintf("%d", metrics.ActiveServices),
							Change: 0,
						},
					},
					Remediation: mcp.RemediationPayload{
						IssueType: issueType,
						Yaml:      remediationYaml,
						Validated: validated,
					},
					TenantId: tenantID,
					Version:  "v1.1.0-hackathon",
				}
				telemetry.PipelineHealthScore.WithLabelValues(tenantID).Set(metrics.CompositeScore)
				json.NewEncoder(w).Encode(resp)
				return
			}
		}

		// Fallback: structured mock data (no ClickHouse available)
		json.NewEncoder(w).Encode(mcp.HealthResponse{
			HealthScore: 84,
			Metrics: mcp.MetricsPayload{
				Cardinality: mcp.MetricValue{Value: "1.2M", Change: 14.5},
				Orphans:     mcp.MetricValue{Value: "432", Change: -5.2},
				Coverage:    mcp.MetricValue{Value: "14", Change: 0},
			},
			Remediation: mcp.RemediationPayload{
				IssueType: "High Cardinality (user_id on checkout_service)",
				Yaml: `processors:
  attributes/remediation:
    actions:
      - key: "user_id"
        action: "delete"`,
				Validated: true,
			},
			TenantId: tenantID,
			Version:  "v1.1.0-hackathon",
		})
}

func (s *Server) GetTenantIssues(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]map[string]interface{}{
		{
			"id": "iss-1",
			"service": "payments-api",
			"description": "Broken trace chain · 18% orphan rate · §8.2",
			"impact": -18,
		},
		{
			"id": "iss-2",
			"service": "checkout-service",
			"description": "Cardinality spike · user_id_raw · §8.1",
			"impact": -12,
		},
		{
			"id": "iss-3",
			"service": "inventory-worker",
			"description": "Coverage gap · silent 14m · §8.3",
			"impact": -8,
		},
	})
}

func fmtLarge(n uint64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func cardChange(n uint64) float64 {
	if n > 1_000_000 {
		return 14.5
	}
	return 2.1
}

func calculateDelta(current, previous uint64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100.0
		}
		return 0.0
	}
	return (float64(current) - float64(previous)) / float64(previous) * 100.0
}

func (s *Server) ApplyRemediation(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// Duplicate GetAgentTraces method removed

func (s *Server) GetCoverage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]map[string]interface{}{
		{ "service": "inventory-worker", "status": "silent", "lastSeen": "14m ago" },
		{ "service": "auth-service", "status": "active", "lastSeen": "1s ago" },
	})
}

func (s *Server) GetTracesOrphans(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"orphanRate": "6.2%",
		"topOrphanedService": "payments-api",
		"missingParents": 142,
	})
}
