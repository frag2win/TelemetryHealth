package rest

import (
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// Server is the REST API server for the control plane dashboard.
type Server struct {
	logger *zap.Logger
}

func NewServer(logger *zap.Logger) *Server {
	return &Server{logger: logger}
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

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/tenant/", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Ensure path matches /api/v1/tenant/{id}/health
		if !strings.HasSuffix(r.URL.Path, "/health") {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		
		// In a real implementation, we would query ClickHouse here.
		// For now, return dynamic mock data structured for the React UI.
		json.NewEncoder(w).Encode(map[string]interface{}{
			"healthScore": 84,
			"metrics": map[string]interface{}{
				"cardinality": map[string]interface{}{"value": "1.2M", "change": 14.5},
				"orphans":     map[string]interface{}{"value": "432", "change": -5.2},
				"coverage":    map[string]interface{}{"value": "14", "change": 0},
			},
			"remediation": map[string]interface{}{
				"issueType": "High Cardinality (user_id on checkout_service)",
				"yaml":      "processors:\n  attributes/remediation:\n    actions:\n      - key: \"user_id\"\n        action: \"delete\"",
			},
		})
	}))

	s.logger.Info("Starting API Server", zap.String("addr", addr))
	return http.ListenAndServe(addr, mux)
}
