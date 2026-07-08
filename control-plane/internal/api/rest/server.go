package rest

// @title TelemetryHealth API
// @version 1.0
// @description REST API for TelemetryHealth control plane
// @host localhost:8080
// @BasePath /api/v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/remediation"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage/clickhouse"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.uber.org/zap"

	_ "github.com/frag2win/TelemetryHealth/control-plane/docs" // imported for swagger
)

type HealthResponse struct {
	HealthScore float64            `json:"healthScore"`
	Metrics     MetricsPayload     `json:"metrics"`
	Remediation RemediationPayload `json:"remediation"`
	TenantId    string             `json:"tenantId"`
	Version     string             `json:"version"`
}

type MetricValue struct {
	Value  string  `json:"value"`
	Change float64 `json:"change"`
}

type MetricsPayload struct {
	Cardinality MetricValue `json:"cardinality"`
	Orphans     MetricValue `json:"orphans"`
	Coverage    MetricValue `json:"coverage"`
}

type RemediationPayload struct {
	IssueType string `json:"issueType"`
	Yaml      string `json:"yaml"`
	Validated bool   `json:"validated"`
}

// Server is the REST API server for the control plane dashboard.
type Server struct {
	logger     *zap.Logger
	healthRepo *clickhouse.HealthRepository
	validator  *remediation.Validator
}

func NewServer(logger *zap.Logger, healthRepo *clickhouse.HealthRepository) *Server {
	return &Server{logger: logger, healthRepo: healthRepo, validator: remediation.NewValidator(logger)}
}

// corsMiddleware adds basic CORS headers.
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// metricsMiddleware tracks API requests for Prometheus.
func metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timer := prometheus.NewTimer(telemetry.ApiRequestDuration.WithLabelValues(r.Method, r.URL.Path, "200"))
		defer timer.ObserveDuration()

		telemetry.ApiRequestsTotal.WithLabelValues(r.Method, r.URL.Path, "200").Inc()
		next(w, r)
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
		} else {
			http.NotFound(w, r)
		}
	})))

	s.logger.Info("Starting API Server", zap.String("addr", addr))
	return http.ListenAndServe(addr, mux)
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
					remediationYaml = `processors:
  attributes/remediation:
    actions:
      - key: "user_id"
        action: "delete"`
					if s.validator != nil {
						validated, _ = s.validator.Validate(r.Context(), remediationYaml)
					}
				}

				resp := HealthResponse{
					HealthScore: metrics.CompositeScore,
					Metrics: MetricsPayload{
						Cardinality: MetricValue{
							Value:  fmtLarge(metrics.CardinalityMax),
							Change: cardChange(metrics.CardinalityMax),
						},
						Orphans: MetricValue{
							Value:  fmt.Sprintf("%d", metrics.OrphanCount),
							Change: calculateDelta(metrics.OrphanCount, metrics.PreviousOrphanCount),
						},
						Coverage: MetricValue{
							Value:  fmt.Sprintf("%d", metrics.ActiveServices),
							Change: 0,
						},
					},
					Remediation: RemediationPayload{
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
		json.NewEncoder(w).Encode(HealthResponse{
			HealthScore: 84,
			Metrics: MetricsPayload{
				Cardinality: MetricValue{Value: "1.2M", Change: 14.5},
				Orphans:     MetricValue{Value: "432", Change: -5.2},
				Coverage:    MetricValue{Value: "14", Change: 0},
			},
			Remediation: RemediationPayload{
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
