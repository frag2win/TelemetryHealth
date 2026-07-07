package authz

import (
	"context"
	"errors"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// TenantAuthInterceptor validates that the tenant_id in the gRPC metadata
// matches the SAN/SPIFFE ID presented in the client's mTLS certificate.
func TenantAuthInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := verifyTenant(ctx); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		return handler(ctx, req)
	}
}

// verifyTenant extracts peer TLS info and metadata
func verifyTenant(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return errors.New("missing metadata")
	}

	tenantIDs := md.Get("x-tenant-id")
	if len(tenantIDs) == 0 || tenantIDs[0] == "" {
		return errors.New("missing x-tenant-id header")
	}
	claimedTenant := tenantIDs[0]

	// Bypass mTLS verification for local development
	if os.Getenv("INSECURE_DEV_MODE") == "true" {
		return nil
	}

	p, ok := peer.FromContext(ctx)
	if !ok || p.AuthInfo == nil {
		return errors.New("no peer auth info found (mTLS required)")
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return errors.New("peer auth info is not TLS")
	}

	if len(tlsInfo.State.VerifiedChains) == 0 || len(tlsInfo.State.VerifiedChains[0]) == 0 {
		return errors.New("no verified certificate chain")
	}

	cert := tlsInfo.State.VerifiedChains[0][0]

	valid := false
	
	// Check URIs for SPIFFE ID matching the tenant
	for _, uri := range cert.URIs {
		// e.g. spiffe://telemetryhealth.internal/tenant/<uuid>
		if uri.String() == claimedTenant || uri.Path == "/"+claimedTenant {
			valid = true
			break
		}
	}

	// Check DNSNames (SANs) for the tenant
	for _, san := range cert.DNSNames {
		if san == claimedTenant {
			valid = true
			break
		}
	}

	if !valid {
		return errors.New("tenant_id claim does not match mTLS certificate SAN/SPIFFE ID")
	}

	return nil
}
