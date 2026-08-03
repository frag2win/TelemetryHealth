package rest

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/frag2win/TelemetryHealth/control-plane/internal/behavior"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/decision"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/engine"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/rootcause"
	"github.com/frag2win/TelemetryHealth/control-plane/internal/telemetry"
	"github.com/frag2win/TelemetryHealth/control-plane/pkg/models"
)

// GetAgentTraces returns LLM agent trace data.
func (s *Server) GetAgentTraces(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if !validateTenantID(w, tenantID) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if s.healthRepo != nil {
		traces, err := s.healthRepo.QueryAgentTraces(r.Context())
		if err != nil {
			s.logger.Error("query agent traces failed", zap.Error(err))
			writeError(w, "DATA_SOURCE_ERROR", "Failed to query agent traces", http.StatusServiceUnavailable)
			return
		}

		for _, trace := range traces {
			serviceName := "ai-agent-service"
			agentID := trace.Model
			
			score := 100.0
			errorCount := 0.0
			for _, dec := range trace.Decisions {
				if dec.Status == "error" || dec.Status == "warning" {
					score -= 20.0
					if dec.Status == "error" {
						errorCount += 1.0
					}
				}
			}
			if trace.HallucinationRisk == "High" {
				score -= 30.0
			}
			if score < 0 {
				score = 0
			}

			telemetry.AgentHealthScore.WithLabelValues(serviceName, agentID).Set(score)
			telemetry.AgentTokenBurnRate.WithLabelValues(serviceName, agentID).Add(float64(trace.Tokens))
			if errorCount > 0 {
				telemetry.AgentTraceErrorCount.WithLabelValues(serviceName, agentID).Add(errorCount)
			}

			riskVal := 0.0
			if trace.HallucinationRisk == "High" {
				riskVal = 1.0
			} else if trace.HallucinationRisk == "Medium" {
				riskVal = 0.5
			}
			telemetry.AgentHallucinationRisk.WithLabelValues(serviceName, agentID).Set(riskVal)

			if len(trace.Decisions) > 0 {
				efficiency := float64(trace.Tokens) / float64(len(trace.Decisions))
				telemetry.AgentTokenEfficiency.WithLabelValues(serviceName, agentID).Set(efficiency)
			}
		}

		s.encodeResponse(w, traces)
		return
	}
	writeError(w, "DATA_SOURCE_UNCONFIGURED", "ClickHouse repository not configured", http.StatusNotImplemented)
}

func (s *Server) handleBehaviorGraph(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if !validateTenantID(w, tenantID) {
		return
	}
	graph := s.graphEngine.GenerateBehaviorGraph(tenantID)
	w.Header().Set("Content-Type", "application/json")
	s.encodeResponse(w, graph)
}

// GetTenantRootCause returns the causal graph explaining an issue.
func (s *Server) GetTenantRootCause(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if !validateTenantID(w, tenantID) {
		return
	}
	issueID := r.URL.Query().Get("issue_id")

	w.Header().Set("Content-Type", "application/json")
	graph := s.graphEngine.GenerateRootCause(tenantID, issueID)
	s.encodeResponse(w, graph)
}

// GetBehaviorGraph returns the reconstructed BehaviorGraph for a given traceID formatted for ReactFlow.
func (s *Server) GetBehaviorGraph(w http.ResponseWriter, r *http.Request) {
	traceID := chi.URLParam(r, "trace_id")
	if traceID == "" {
		writeError(w, "INVALID_TRACE_ID", "trace_id is required", http.StatusBadRequest)
		return
	}

	var spans []models.SpanData
	var err error
	if s.healthRepo != nil {
		spans, err = s.healthRepo.QuerySpansByTraceID(r.Context(), traceID)
	}
	if s.healthRepo == nil || err != nil || len(spans) == 0 {
		w.Header().Set("Content-Type", "application/json")
		s.encodeResponse(w, fallbackBehaviorGraph(traceID))
		return
	}

	behEngine := behavior.NewEngine()
	graph, err := behEngine.Reconstruct(traceID, spans)
	if err != nil {
		s.logger.Error("Failed to reconstruct behavior graph", zap.String("trace_id", traceID), zap.Error(err))
		writeError(w, "RECONSTRUCTION_FAILED", "Failed to reconstruct behavior graph: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var rfNodes []engine.GraphNode
	var rfEdges []engine.GraphEdge
	xPos := 50.0
	for _, n := range graph.Nodes {
		nodeType := "service"
		if n.Actor != "" {
			nodeType = strings.ToLower(n.Actor)
		}
		status := "healthy"
		if n.Confidence < 0.95 {
			status = "warning"
		}
		detail := ""
		if reason, ok := n.Metadata["failure_reason"]; ok {
			detail = reason
			status = "critical"
		} else if warningMsg, ok := n.Metadata["warning"]; ok {
			detail = warningMsg
			status = "warning"
		} else {
			detail = fmt.Sprintf("Duration: %.2fms | Confidence: %.0f%%", n.DurationMs, n.Confidence*100)
		}
		rfNodes = append(rfNodes, engine.GraphNode{
			ID:       n.BehaviorID,
			Position: engine.NodePosition{X: xPos, Y: 120},
			Type:     nodeType,
			Data: engine.GraphNodeData{
				Label:  fmt.Sprintf("%s (%s)", n.Type, n.Actor),
				Type:   nodeType,
				Status: status,
				Detail: detail,
			},
		})
		xPos += 270
	}
	for _, e := range graph.Edges {
		rfEdges = append(rfEdges, engine.GraphEdge{
			ID:       fmt.Sprintf("e-%s-%s", e.Source, e.Destination),
			Source:   e.Source,
			Target:   e.Destination,
			Animated: true,
			Label:    e.Type,
		})
	}
	rfGraph := engine.Graph{Nodes: rfNodes, Edges: rfEdges}

	w.Header().Set("Content-Type", "application/json")
	s.encodeResponse(w, rfGraph)
}

// GetDecisionGraph returns the reconstructed DecisionGraph for a given traceID.
func (s *Server) GetDecisionGraph(w http.ResponseWriter, r *http.Request) {
	traceID := chi.URLParam(r, "trace_id")
	if traceID == "" {
		writeError(w, "INVALID_TRACE_ID", "trace_id is required", http.StatusBadRequest)
		return
	}

	var spans []models.SpanData
	var err error
	if s.healthRepo != nil {
		spans, err = s.healthRepo.QuerySpansByTraceID(r.Context(), traceID)
	}
	if s.healthRepo == nil || err != nil || len(spans) == 0 {
		w.Header().Set("Content-Type", "application/json")
		s.encodeResponse(w, fallbackDecisionGraph(traceID))
		return
	}

	behEngine := behavior.NewEngine()
	behGraph, err := behEngine.Reconstruct(traceID, spans)
	if err != nil {
		s.logger.Error("Failed to reconstruct behavior graph", zap.String("trace_id", traceID), zap.Error(err))
		writeError(w, "RECONSTRUCTION_FAILED", "Failed to reconstruct behavior graph: "+err.Error(), http.StatusInternalServerError)
		return
	}

	decEngine := decision.NewEngine()
	graph, err := decEngine.Reconstruct(behGraph)
	if err != nil {
		s.logger.Error("Failed to reconstruct decision graph", zap.String("trace_id", traceID), zap.Error(err))
		writeError(w, "RECONSTRUCTION_FAILED", "Failed to reconstruct decision graph: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	s.encodeResponse(w, graph)
}

// GetRootCause returns the RootCause diagnosis for a given traceID.
func (s *Server) GetRootCause(w http.ResponseWriter, r *http.Request) {
	traceID := chi.URLParam(r, "trace_id")
	if traceID == "" {
		writeError(w, "INVALID_TRACE_ID", "trace_id is required", http.StatusBadRequest)
		return
	}

	var spans []models.SpanData
	var err error
	if s.healthRepo != nil {
		spans, err = s.healthRepo.QuerySpansByTraceID(r.Context(), traceID)
	}
	if s.healthRepo == nil || err != nil || len(spans) == 0 {
		w.Header().Set("Content-Type", "application/json")
		s.encodeResponse(w, fallbackRootCause(traceID))
		return
	}

	behEngine := behavior.NewEngine()
	behGraph, err := behEngine.Reconstruct(traceID, spans)
	if err != nil {
		s.logger.Error("Failed to reconstruct behavior graph", zap.String("trace_id", traceID), zap.Error(err))
		writeError(w, "RECONSTRUCTION_FAILED", "Failed to reconstruct behavior graph: "+err.Error(), http.StatusInternalServerError)
		return
	}

	decEngine := decision.NewEngine()
	decGraph, err := decEngine.Reconstruct(behGraph)
	if err != nil {
		s.logger.Error("Failed to reconstruct decision graph", zap.String("trace_id", traceID), zap.Error(err))
		writeError(w, "RECONSTRUCTION_FAILED", "Failed to reconstruct decision graph: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rcEngine := rootcause.NewEngine()
	cause, err := rcEngine.Analyze(behGraph, decGraph)
	if err != nil {
		s.logger.Error("Failed to analyze root cause", zap.String("trace_id", traceID), zap.Error(err))
		writeError(w, "ANALYSIS_FAILED", "Failed to analyze root cause: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	s.encodeResponse(w, cause)
}

// fallbackBehaviorGraph returns graph structure for trace rendering fallback.
func fallbackBehaviorGraph(traceID string) engine.Graph {
	return engine.Graph{
		Nodes: []engine.GraphNode{},
		Edges: []engine.GraphEdge{},
	}
}

// fallbackDecisionGraph returns decision graph for trace rendering fallback.
func fallbackDecisionGraph(traceID string) *models.DecisionGraph {
	return &models.DecisionGraph{
		TraceID:   traceID,
		AgentID:   "ai-agent",
		Decisions: []models.DecisionNode{},
		Timestamp: time.Now(),
	}
}

// fallbackRootCause returns root cause diagnosis for trace rendering fallback.
func fallbackRootCause(traceID string) *models.RootCause {
	return &models.RootCause{
		TraceID:     traceID,
		AgentID:     "ai-agent",
		FailureType: "None",
		Severity:    "Info",
		Description: "No root cause data found for trace.",
		Confidence:  0.0,
		Status:      "Unknown",
		Timestamp:   time.Now(),
	}
}
