package authz

// tenant_verifier.go — mTLS-based cryptographic tenant identity verification for the Ingest Gateway.
// PRD §10 Security, Goal G8: "100% of ingested control-plane telemetry cryptographically verified
// against client mTLS certificates before entering the stream pipeline."
// Improvement #2.1: Dev-mode bypass is now production-guarded.

import (
	"context"
	"errors"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// ValidateStartupConfig checks for dangerous configurations at startup time.
// Call this from main() before starting the gRPC server.
// If INSECURE_DEV_MODE is set AND ENV=production, this function panics with a clear message.
// This prevents the dev-mode bypass from silently appearing in production (PRD §10, Improvement #2.1).
func ValidateStartupConfig() {
	if os.Getenv("INSECURE_DEV_MODE") == "true" && os.Getenv("ENV") == "production" {
		panic(
			"FATAL: INSECURE_DEV_MODE=true is set in a production environment (ENV=production). " +
				"This completely bypasses mTLS tenant verification and violates PRD Goal G8. " +
				"Remove INSECURE_DEV_MODE from your production deployment config and configure " +
				"mTLS certificates via GRPC_TLS_CERT, GRPC_TLS_KEY, and GRPC_TLS_CA env vars.",
		)
	}
}

// TenantAuthInterceptor validates that the tenant_id in the gRPC metadata
// matches the SAN/SPIFFE ID presented in the client's mTLS certificate.
// PRD §10, Goal G8: cryptographic verification before any data enters the stream pipeline.
func TenantAuthInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := verifyTenant(ctx); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		return handler(ctx, req)
	}
}

// verifyTenant extracts peer TLS info and metadata and cryptographically matches
// the claimed tenant_id to the mTLS certificate's SAN/SPIFFE ID.
func verifyTenant(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return errors.New("missing gRPC metadata")
	}

	tenantIDs := md.Get("x-tenant-id")
	if len(tenantIDs) == 0 || tenantIDs[0] == "" {
		return errors.New("missing x-tenant-id header in gRPC metadata")
	}
	claimedTenant := tenantIDs[0]

	// INSECURE_DEV_MODE bypass — ONLY permitted when ENV != production.
	// In production this code path is blocked by ValidateStartupConfig() panicking at startup.
	if os.Getenv("INSECURE_DEV_MODE") == "true" {
		if os.Getenv("ENV") == "production" {
			// Belt-and-suspenders: even if startup check somehow passed, refuse here too.
			return fmt.Errorf("INSECURE_DEV_MODE is forbidden in production (ENV=production)")
		}
		// Log a prominent warning to stderr so it's visible in any log aggregator.
		fmt.Fprintln(os.Stderr, "WARNING: INSECURE_DEV_MODE is enabled — mTLS tenant verification is BYPASSED. Do NOT use in production!")
		return nil
	}

	// Production path: require mTLS peer info.
	p, ok := peer.FromContext(ctx)
	if !ok || p.AuthInfo == nil {
		return errors.New("no peer auth info found; mTLS is required for all collector connections")
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return errors.New("peer auth info is not TLS; only mTLS connections are accepted")
	}

	if len(tlsInfo.State.VerifiedChains) == 0 || len(tlsInfo.State.VerifiedChains[0]) == 0 {
		return errors.New("no verified certificate chain; client certificate must be signed by a trusted CA")
	}

	cert := tlsInfo.State.VerifiedChains[0][0]

	// Step 1: Check SPIFFE URIs (canonical format for this system).
	// Expected: spiffe://telemetryhealth.internal/tenant/<uuid>
	// PRD §10: "SPIFFE/SPIRE or mTLS for collector→control-plane identity"
	for _, uri := range cert.URIs {
		if uri.Scheme == "spiffe" {
			// Accept either /tenant/<id> or /<id> path formats.
			if uri.Path == "/tenant/"+claimedTenant || uri.Path == "/"+claimedTenant {
				return nil // Verified via SPIFFE ID
			}
		}
	}

	// Step 2: Fallback to DNS SAN matching (for non-SPIFFE PKI setups).
	for _, san := range cert.DNSNames {
		if san == claimedTenant {
			return nil // Verified via DNS SAN
		}
	}

	return fmt.Errorf(
		"tenant_id claim %q does not match any SAN or SPIFFE ID in the client mTLS certificate; "+
			"verify the collector's mTLS certificate is issued with the correct SPIFFE ID "+
			"(spiffe://telemetryhealth.internal/tenant/%s)",
		claimedTenant, claimedTenant,
	)
}
