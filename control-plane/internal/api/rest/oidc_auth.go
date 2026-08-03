package rest

// oidc_auth.go — OIDC JWT verification helpers for the REST API middleware.
// PRD §10 Security: "OIDC SSO for dashboard; RBAC roles: Org Admin, Service Owner, Read-Only"
// Improvement #1.2: Real OIDC JWT validation using go-oidc/v3 with JWKS-based signature verification.

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
)

// oidcVerifierCache holds a lazily-initialised OIDC verifier per issuer URL.
// The verifier automatically fetches and caches the JWKS from the IdP.
var (
	oidcVerifierMu    sync.RWMutex
	oidcVerifierCache map[string]*gooidc.IDTokenVerifier = make(map[string]*gooidc.IDTokenVerifier)
)

// getOrCreateVerifier lazily creates an OIDC verifier for the given issuer.
// The verifier is cached after the first successful creation to avoid redundant JWKS fetches.
func getOrCreateVerifier(ctx context.Context, issuer string) (*gooidc.IDTokenVerifier, error) {
	oidcVerifierMu.RLock()
	if v, ok := oidcVerifierCache[issuer]; ok {
		oidcVerifierMu.RUnlock()
		return v, nil
	}
	oidcVerifierMu.RUnlock()

	// Create verifier (discovers JWKS endpoint automatically via OIDC discovery).
	provider, err := gooidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc: provider discovery for %q failed: %w", issuer, err)
	}

	audience := os.Getenv("OIDC_AUDIENCE")
	if audience == "" {
		audience = "telemetryhealth-api"
	}

	verifier := provider.Verifier(&gooidc.Config{
		ClientID: audience,
	})

	oidcVerifierMu.Lock()
	oidcVerifierCache[issuer] = verifier
	oidcVerifierMu.Unlock()

	return verifier, nil
}

// verifyOIDCToken validates the raw JWT token against the configured OIDC issuer.
// Returns (actorID, actorRole, error).
// - actorID:   the "sub" claim (unique user ID)
// - actorRole: the role claim, mapped to one of "Org Admin", "Service Owner", "Read-Only"
func verifyOIDCToken(ctx context.Context, issuer, rawToken string) (string, string, error) {
	// Use a short-lived context for OIDC provider discovery to avoid blocking requests.
	discCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	verifier, err := getOrCreateVerifier(discCtx, issuer)
	if err != nil {
		return "", "", fmt.Errorf("oidc: verifier init: %w", err)
	}

	idToken, err := verifier.Verify(ctx, rawToken)
	if err != nil {
		return "", "", fmt.Errorf("oidc: token verification failed: %w", err)
	}

	// Extract standard claims.
	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return "", "", fmt.Errorf("oidc: claim extraction failed: %w", err)
	}

	// Extract configurable role claim.
	roleClaimKey := os.Getenv("OIDC_ROLE_CLAIM")
	if roleClaimKey == "" {
		roleClaimKey = "role"
	}
	roleClaim := extractClaim(idToken, roleClaimKey)
	actorRole := mapToRBACRole(roleClaim)

	actorID := claims.Sub
	if actorID == "" {
		actorID = claims.Email
	}

	return actorID, actorRole, nil
}

// extractClaim extracts a single string claim from an OIDC ID token by key.
func extractClaim(idToken *gooidc.IDToken, key string) string {
	var raw map[string]interface{}
	if err := idToken.Claims(&raw); err != nil {
		return ""
	}
	if v, ok := raw[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// mapToRBACRole maps an IdP role string to the three PRD-defined RBAC roles.
// PRD §10: "Org Admin, Service Owner (scoped to their services only), Read-Only"
func mapToRBACRole(raw string) string {
	switch strings.ToLower(raw) {
	case "org_admin", "admin", "org-admin":
		return "Org Admin"
	case "service_owner", "service-owner", "owner":
		return "Service Owner"
	default:
		return "Read-Only"
	}
}
