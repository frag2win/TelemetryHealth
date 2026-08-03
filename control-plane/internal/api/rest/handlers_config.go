package rest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
)

// HandleTenantConfigGet serves GET /api/v1/tenant/{tenant_id}/config.
func (s *Server) HandleTenantConfigGet(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if !validateTenantID(w, tenantID) {
		return
	}
	if s.healthRepo == nil {
		writeError(w, "DATA_SOURCE_UNCONFIGURED", "ClickHouse repository not configured", http.StatusNotImplemented)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	weights, err := s.healthRepo.GetTenantWeights(r.Context(), tenantID)
	if err != nil {
		s.logger.Error("failed to get tenant config", zap.String("tenant_id", tenantID), zap.Error(err))
		writeError(w, "DATA_SOURCE_ERROR", "Failed to retrieve tenant config", http.StatusServiceUnavailable)
		return
	}
	s.encodeResponse(w, weights)
}

// HandleTenantConfigPut serves PUT/POST /api/v1/tenant/{tenant_id}/config.
func (s *Server) HandleTenantConfigPut(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if !validateTenantID(w, tenantID) {
		return
	}
	if s.healthRepo == nil {
		writeError(w, "DATA_SOURCE_UNCONFIGURED", "ClickHouse repository not configured", http.StatusNotImplemented)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB payload limit
	var weights telemetry.TenantWeights
	if err := json.NewDecoder(r.Body).Decode(&weights); err != nil {
		writeError(w, "INVALID_REQUEST_BODY", "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.healthRepo.SaveTenantConfig(r.Context(), tenantID, weights); err != nil {
		s.logger.Error("failed to save tenant config", zap.String("tenant_id", tenantID), zap.Error(err))
		writeError(w, "DATA_SOURCE_ERROR", "Failed to save tenant config", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	s.encodeResponse(w, map[string]string{"status": "success"})
}
