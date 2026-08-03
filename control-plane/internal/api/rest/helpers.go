package rest

import (
	"encoding/json"
	"net/http"
	"os"
	"regexp"

	"go.uber.org/zap"
)

type contextKey string

const (
	contextKeyActorID   contextKey = "actor_id"
	contextKeyActorRole contextKey = "actor_role"
)

// uuidRegex validates that a tenant_id conforms to UUID v4 format.
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// devSlugRegex validates that a tenant_id conforms to a clean slug format in development.
var devSlugRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// APIError is the standard structured error response body.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeError writes a structured JSON error response — never leaks raw Go error strings.
func writeError(w http.ResponseWriter, code string, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(APIError{Code: code, Message: message}); err != nil {
		zap.L().Warn("failed to encode json error response", zap.Error(err))
	}
}

// encodeResponse writes a JSON response and logs encoding errors.
func (s *Server) encodeResponse(w http.ResponseWriter, payload interface{}) {
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		if s != nil && s.logger != nil {
			s.logger.Warn("failed to encode json response", zap.Error(err))
		} else {
			zap.L().Warn("failed to encode json response", zap.Error(err))
		}
	}
}

// validateTenantID checks that tenant_id is a valid UUID or slug in dev mode.
func validateTenantID(w http.ResponseWriter, tenantID string) bool {
	isProduction := os.Getenv("ENV") == "production"
	if isProduction {
		if !uuidRegex.MatchString(tenantID) {
			writeError(w, "INVALID_TENANT_ID", "tenant_id must be a valid UUID", http.StatusBadRequest)
			return false
		}
	} else {
		if !uuidRegex.MatchString(tenantID) && !devSlugRegex.MatchString(tenantID) {
			writeError(w, "INVALID_TENANT_ID", "tenant_id must be a valid UUID or a valid alphanumeric slug", http.StatusBadRequest)
			return false
		}
	}
	return true
}
