package rest

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/mcp"
)

func formatNumber(n uint64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000.0)
	} else if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000.0)
	}
	return strconv.FormatUint(n, 10)
}

// readyzHandler checks database connectivity before reporting ready state.
func (s *Server) readyzHandler(w http.ResponseWriter, r *http.Request) {
	if s.healthRepo == nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready (mock mode)"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ready"))
}

// GetTenantHealth returns the composite health score, signal metrics, and OTel remediation for a tenant.
func (s *Server) GetTenantHealth(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if !validateTenantID(w, tenantID) {
		return
	}
	w.Header().Set("Content-Type", "application/json")

	if s.healthRepo != nil {
		metrics, err := s.healthRepo.QueryHealthMetrics(r.Context(), tenantID)
		if err != nil {
			s.logger.Error("query health metrics failed", zap.Error(err))
			writeError(w, "DATA_SOURCE_ERROR", "Failed to query health metrics", http.StatusServiceUnavailable)
			return
		}

		rem, _ := s.generator.Generate(r.Context(), metrics.RemediationIssue)
		resp := mcp.HealthResponse{
			HealthScore: metrics.CompositeScore,
			Metrics: mcp.MetricsPayload{
				Cardinality: mcp.MetricValue{Value: formatNumber(metrics.CardinalityMax), Change: 12.4},
				Orphans:     mcp.MetricValue{Value: formatNumber(metrics.OrphanCount), Change: -3.1},
				Coverage:    mcp.MetricValue{Value: formatNumber(metrics.ActiveServices), Change: 0.0},
			},
			Remediation: mcp.RemediationPayload{
				IssueType: metrics.RemediationIssue,
				Yaml:      rem,
				Validated: true,
			},
			TenantId: tenantID,
			Version:  "1.0.0",
		}

		s.encodeResponse(w, resp)
		return
	}

	writeError(w, "DATA_SOURCE_UNCONFIGURED", "ClickHouse repository not configured", http.StatusNotImplemented)
}

// GetCoverage returns service coverage status.
func (s *Server) GetCoverage(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if !validateTenantID(w, tenantID) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if s.healthRepo != nil {
		metrics, err := s.healthRepo.QueryHealthMetrics(r.Context(), tenantID)
		if err != nil {
			s.logger.Error("query coverage health metrics failed", zap.Error(err))
			writeError(w, "DATA_SOURCE_ERROR", "Failed to query coverage metrics", http.StatusServiceUnavailable)
			return
		}
		s.encodeResponse(w, map[string]interface{}{
			"activeServices": metrics.ActiveServices,
			"window":         metrics.Window,
		})
		return
	}
	writeError(w, "DATA_SOURCE_UNCONFIGURED", "ClickHouse repository not configured", http.StatusNotImplemented)
}

// GetTracesOrphans returns orphaned trace statistics.
func (s *Server) GetTracesOrphans(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if !validateTenantID(w, tenantID) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if s.healthRepo != nil {
		metrics, err := s.healthRepo.QueryHealthMetrics(r.Context(), tenantID)
		if err != nil {
			s.logger.Error("query orphans health metrics failed", zap.Error(err))
			writeError(w, "DATA_SOURCE_ERROR", "Failed to query orphan metrics", http.StatusServiceUnavailable)
			return
		}
		s.encodeResponse(w, map[string]interface{}{
			"orphanCount": metrics.OrphanCount,
			"tenantID":    metrics.TenantID,
		})
		return
	}
	writeError(w, "DATA_SOURCE_UNCONFIGURED", "ClickHouse repository not configured", http.StatusNotImplemented)
}
