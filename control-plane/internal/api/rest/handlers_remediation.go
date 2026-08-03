package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/simulator"
)

// ApplyRemediation logs a remediation apply event with full SOC 2 audit trail.
func (s *Server) ApplyRemediation(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB payload limit
	type ApplyRequest struct {
		TenantID    string `json:"tenantId"`
		IssueType   string `json:"issueType"`
		Yaml        string `json:"yaml"`
		ServiceName string `json:"serviceName"`
	}

	var req ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "INVALID_REQUEST_BODY", "Could not parse request body", http.StatusBadRequest)
		return
	}

	// Validate tenant UUID
	if !validateTenantID(w, req.TenantID) {
		return
	}

	// Size check
	const maxYAMLSize = 64 * 1024 // 64 KB limit
	if len(req.Yaml) > maxYAMLSize {
		writeError(w, "PAYLOAD_TOO_LARGE", "YAML content exceeds maximum allowed size of 64KB", http.StatusRequestEntityTooLarge)
		return
	}

	// Run validator before writing
	if s.validator != nil && req.Yaml != "" {
		valid, err := s.validator.Validate(r.Context(), req.Yaml)
		if !valid || err != nil {
			msg := "Invalid YAML or forbidden OTel components in remediation configuration"
			if err != nil {
				msg = fmt.Sprintf("Validation failed: %v", err)
			}
			writeError(w, "INVALID_YAML_CONFIG", msg, http.StatusBadRequest)
			return
		}
	}

	actorID, _ := r.Context().Value(contextKeyActorID).(string)
	actorRole, _ := r.Context().Value(contextKeyActorRole).(string)
	if actorID == "" {
		actorID = "unknown-actor"
	}
	if actorRole == "" {
		actorRole = "Read-Only"
	}

	sourceIP := r.RemoteAddr
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		sourceIP = ip
	}

	if s.healthRepo != nil && req.TenantID != "" {
		err := s.healthRepo.LogRemediationEvent(
			r.Context(),
			req.TenantID,
			req.IssueType,
			req.Yaml,
			true,  // validated
			true,  // applied
			actorID,
			actorRole,
			sourceIP,
			"apply",
			req.ServiceName,
		)
		if err != nil {
			s.logger.Error("Failed to write SOC 2 audit log to ClickHouse", zap.Error(err))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	s.encodeResponse(w, map[string]string{"status": "success"})
}

// SimulateFailure injects a simulated anomaly into the pipeline for testing.
func (s *Server) SimulateFailure(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if !validateTenantID(w, tenantID) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB payload limit
	var req struct {
		Scenario string `json:"scenario"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "INVALID_REQUEST", "invalid json body", http.StatusBadRequest)
		return
	}

	endpoint := os.Getenv("INGEST_GATEWAY_ENDPOINT")
	if endpoint == "" {
		endpoint = "127.0.0.1:4317"
	}

	sim := simulator.NewSimulator(s.logger, endpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	switch req.Scenario {
	case "high_cardinality":
		err = sim.InjectHighCardinality(ctx, tenantID)
	case "dropped_spans":
		err = sim.InjectDroppedSpans(ctx, tenantID)
	case "agentic_workflow":
		err = sim.InjectAgenticWorkflow(ctx, tenantID)
	case "agentic_failure":
		err = sim.InjectAgenticFailure(ctx, tenantID)
	default:
		writeError(w, "UNKNOWN_SCENARIO", "unknown scenario: "+req.Scenario, http.StatusBadRequest)
		return
	}

	if err != nil {
		s.logger.Error("failed to inject simulation payload", zap.Error(err))
		writeError(w, "SIMULATION_FAILED", "Failed to inject simulation payload: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	s.encodeResponse(w, map[string]string{
		"status":   "accepted",
		"message":  "Simulation scenario triggered successfully",
		"scenario": req.Scenario,
	})
}
