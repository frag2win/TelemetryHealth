package rest

import (
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
	r.Get("/livez", s.livezHandler)
	r.Get("/healthz", s.livezHandler)
	r.Get("/readyz", s.readyzHandler)

	// Tenant-scoped API endpoints
	r.Route("/api/v1/tenant/{tenant_id}", func(r chi.Router) {
		r.Use(oidcAuthMiddleware)
		r.Get("/health", s.GetTenantHealth)
		r.Post("/simulate", s.SimulateFailure)
		r.Get("/agents", s.GetAgentTraces)
		r.Get("/coverage", s.GetCoverage)
		r.Get("/traces/orphans", s.GetTracesOrphans)
		r.Get("/config", s.HandleTenantConfigGet)
		r.Put("/config", s.HandleTenantConfigPut)
		r.Post("/config", s.HandleTenantConfigPut)
		r.Get("/behavior", s.handleBehaviorGraph)
		r.Get("/root-cause", s.GetTenantRootCause)
	})

	// Remediation apply endpoint
	r.With(oidcAuthMiddleware).Post("/api/v1/remediation/apply", s.ApplyRemediation)

	// Agent trace intelligence endpoints
	r.Route("/api/v1/trace/{trace_id}", func(r chi.Router) {
		r.Use(oidcAuthMiddleware)
		r.Get("/behavior", s.GetBehaviorGraph)
		r.Get("/decisions", s.GetDecisionGraph)
		r.Get("/root-cause", s.GetRootCause)
	})

	return r
}
