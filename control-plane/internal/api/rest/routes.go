package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// routes sets up the routing logic and middleware stack for the server.
func (s *Server) routes() *chi.Mux {
	r := chi.NewRouter()

	// Core middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(rateLimitMiddleware)
	r.Use(corsMiddleware)
	r.Use(metricsMiddleware)
	r.Use(tracingMiddleware)

	// Infrastructure endpoints (no auth required)
	r.Handle("/metrics", promhttp.Handler())
	r.Handle("/swagger/*", httpSwagger.WrapHandler)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	r.Get("/readyz", s.readyzHandler)

	// Tenant-scoped API endpoints (auth required)
	r.Route("/api/v1/tenant/{tenant_id}", func(r chi.Router) {
		r.Use(oidcAuthMiddleware)
		r.Get("/health", s.GetTenantHealth)
		r.Post("/simulate", s.SimulateFailure)
		r.Get("/issues", s.GetTenantIssues)
		r.Get("/agents", s.GetAgentTraces)
		r.Get("/coverage", s.GetCoverage)
		r.Get("/traces/orphans", s.GetTracesOrphans)
		r.Get("/config", s.HandleTenantConfigGet)
		r.Put("/config", s.HandleTenantConfigPut)
		r.Post("/config", s.HandleTenantConfigPut)
		r.Get("/behavior", s.handleBehaviorGraph)
		r.Get("/root-cause", s.GetTenantRootCause)
		r.Get("/replay", s.GetTenantReplay)
	})

	// SigNoz connectivity and config endpoints
	r.Route("/api/v1/signoz", func(r chi.Router) {
		r.Use(oidcAuthMiddleware)
		r.Get("/health", s.handleSignozHealth)
		r.Get("/config", s.handleSignozConfig)
	})

	// Remediation apply endpoint
	r.With(oidcAuthMiddleware).Post("/api/v1/remediation/apply", s.ApplyRemediation)

	// Agent trace intelligence endpoints (milestone Person A)
	r.Route("/api/agents/{agent_id}/traces/{trace_id}", func(r chi.Router) {
		r.Use(oidcAuthMiddleware)
		r.Get("/behavior", s.GetBehaviorGraph)
		r.Get("/decisions", s.GetDecisionGraph)
		r.Get("/root-cause", s.GetRootCause)
	})

	return r
}
