package identity_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/identity"
)

func TestWorkloadIdentity_TokenExchange(t *testing.T) {
	cfg := &identity.WorkloadIdentityConfig{
		ProjectNumber:       "123456789012",
		PoolID:              "k8s-pool",
		ProviderID:          "k8s-provider",
		ServiceAccountEmail: "mesh-proxy@my-project.iam.gserviceaccount.com",
		TokenLifetime:       10 * time.Minute,
		IsSimulated:         true,
	}

	mgr := identity.NewWorkloadIdentityManager(cfg)

	req := &identity.TokenExchangeRequest{
		SubjectToken:     "eyJhbGciOiJSUzI1NiIsImtpZCI6IjEifQ.sample-k8s-sa-jwt",
		SubjectTokenType: "urn:ietf:params:oauth:token-type:jwt",
	}

	token, err := mgr.ExchangeToken(context.Background(), req)
	if err != nil {
		t.Fatalf("ExchangeToken failed: %v", err)
	}

	if !strings.HasPrefix(token.AccessToken, "ya29.") {
		t.Fatalf("expected Google access token prefix ya29., got: %s", token.AccessToken)
	}

	if !token.IsSimulated {
		t.Fatal("expected simulated token when IsSimulated=true")
	}

	// Verify caching on second request
	cachedToken, err := mgr.ExchangeToken(context.Background(), req)
	if err != nil {
		t.Fatalf("cached ExchangeToken failed: %v", err)
	}
	if cachedToken.AccessToken != token.AccessToken {
		t.Fatalf("expected cached token to match, got %s != %s", cachedToken.AccessToken, token.AccessToken)
	}
}

func TestWorkloadIdentity_EmptyTokenFails(t *testing.T) {
	mgr := identity.NewWorkloadIdentityManager(nil)
	_, err := mgr.ExchangeToken(context.Background(), &identity.TokenExchangeRequest{})
	if err == nil {
		t.Fatal("expected error on empty subject token")
	}
}
