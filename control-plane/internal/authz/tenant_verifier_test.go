package authz

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/url"
	"testing"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func TestVerifyTenant_Success_DNSName(t *testing.T) {
	ctx := context.Background()
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("x-tenant-id", "tenant-123"))

	cert := &x509.Certificate{
		DNSNames: []string{"tenant-123"},
	}

	tlsInfo := credentials.TLSInfo{
		State: tls.ConnectionState{
			VerifiedChains: [][]*x509.Certificate{{cert}},
		},
	}

	p := &peer.Peer{AuthInfo: tlsInfo}
	ctx = peer.NewContext(ctx, p)

	err := verifyTenant(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestVerifyTenant_Success_SPIFFE(t *testing.T) {
	ctx := context.Background()
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("x-tenant-id", "tenant-123"))

	uri, _ := url.Parse("spiffe://telemetryhealth.internal/tenant-123")
	cert := &x509.Certificate{
		URIs: []*url.URL{uri},
	}

	tlsInfo := credentials.TLSInfo{
		State: tls.ConnectionState{
			VerifiedChains: [][]*x509.Certificate{{cert}},
		},
	}

	p := &peer.Peer{AuthInfo: tlsInfo}
	ctx = peer.NewContext(ctx, p)

	err := verifyTenant(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestVerifyTenant_MissingHeader(t *testing.T) {
	ctx := context.Background()
	// No metadata
	err := verifyTenant(ctx)
	if err == nil {
		t.Fatal("expected error for missing metadata")
	}
}

func TestVerifyTenant_Mismatch(t *testing.T) {
	ctx := context.Background()
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("x-tenant-id", "tenant-123"))

	cert := &x509.Certificate{
		DNSNames: []string{"tenant-999"}, // Mismatch
	}

	tlsInfo := credentials.TLSInfo{
		State: tls.ConnectionState{
			VerifiedChains: [][]*x509.Certificate{{cert}},
		},
	}

	p := &peer.Peer{AuthInfo: tlsInfo}
	ctx = peer.NewContext(ctx, p)

	err := verifyTenant(ctx)
	if err == nil {
		t.Fatal("expected error for tenant mismatch")
	}
}
