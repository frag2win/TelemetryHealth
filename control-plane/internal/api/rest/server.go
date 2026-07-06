package rest

import (
	"encoding/json"
	"net/http"
	"go.uber.org/zap"
)

// Server is the REST API server for the control plane dashboard.
type Server struct {
	logger *zap.Logger
}

func NewServer(logger *zap.Logger) *Server {
	return &Server{logger: logger}
}

func (s *Server) Start(addr string) error {
	http.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"score":  98.5,
		})
	})
	
	s.logger.Info("Starting API Server", zap.String("addr", addr))
	return http.ListenAndServe(addr, nil)
}
