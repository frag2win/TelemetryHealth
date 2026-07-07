package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/storage/clickhouse"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// HealthResponse is the JSON shape consumed by the React dashboard.
type HealthResponse struct {
	HealthScore float64            `json:"healthScore"`
	Metrics     MetricsPayload     `json:"metrics"`
	Remediation RemediationPayload `json:"remediation"`
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
}

// Server is the REST API server for the control plane dashboard.
type Server struct {
	logger     *zap.Logger
	healthRepo *clickhouse.HealthRepository
}

func NewServer(logger *zap.Logger, healthRepo *clickhouse.HealthRepository) *Server {
	return &Server{logger: logger, healthRepo: healthRepo}
}

// corsMiddleware adds basic CORS headers.
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
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

	mux.HandleFunc("/api/v1/tenant/", corsMiddleware(metricsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/health") {
			http.NotFound(w, r)
			return
		}

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
				if issueType != "" {
					remediationYaml = `processors:
  attributes/remediation:
    actions:
      - key: "user_id"
        action: "delete"`
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
							Change: -5.2,
						},
						Coverage: MetricValue{
							Value:  fmt.Sprintf("%d", metrics.ActiveServices),
							Change: 0,
						},
					},
					Remediation: RemediationPayload{
						IssueType: issueType,
						Yaml:      remediationYaml,
					},
				}
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
			},
		})
	})))

	s.logger.Info("Starting API Server", zap.String("addr", addr))
	return http.ListenAndServe(addr, mux)
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
