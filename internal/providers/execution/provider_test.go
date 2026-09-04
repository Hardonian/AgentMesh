package execution

import (
	"context"
	"testing"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

func TestExecutionProviders(t *testing.T) {
	ctx := context.Background()

	// 1. Proxy Provider
	proxyProv := NewProxyProvider()
	actProxy := &spec.AgentOptimizationAction{
		ActionID:      "act-p1",
		TargetID:      "route-finance",
		CurrentState:  map[string]any{"weight": 100},
		ProposedState: map[string]any{"weight": 80},
		RiskClass:     spec.RiskLow,
	}

	dryProxy, err := proxyProv.DryRun(ctx, actProxy)
	if err != nil || dryProxy == nil {
		t.Fatalf("Proxy dry-run failed: %v", err)
	}

	appProxy, err := proxyProv.Apply(ctx, actProxy)
	if err != nil || !appProxy.Success {
		t.Fatalf("Proxy apply failed: %v", err)
	}

	// 2. GKE Provider with Managed Ownership Check
	gkeProv := NewGKEProvider(true)

	// Attempt on unmanaged resource must fail
	actGKE := &spec.AgentOptimizationAction{
		ActionID:      "act-g1",
		TargetID:      "dep-agent-research",
		CurrentState:  map[string]any{"image": "v1"},
		ProposedState: map[string]any{"image": "v2"},
	}

	_, errUnmanaged := gkeProv.Apply(ctx, actGKE)
	if errUnmanaged != ErrUnmanagedResource {
		t.Fatalf("Expected ErrUnmanagedResource on unmanaged GKE resource, got %v", errUnmanaged)
	}

	// Register managed resource
	gkeProv.RegisterManagedResource("dep-agent-research", map[string]string{
		"agentmesh.io/managed": "true",
	}, map[string]any{"image": "v1"})

	// Now apply should succeed
	appGKE, err := gkeProv.Apply(ctx, actGKE)
	if err != nil || !appGKE.Success {
		t.Fatalf("Managed GKE apply failed: %v", err)
	}

	// 3. Credential Revocation Fails Closed
	gkeRevoked := NewGKEProvider(false) // revoked credentials
	_, errRevoked := gkeRevoked.Apply(ctx, actGKE)
	if errRevoked != ErrCredentialRevocation {
		t.Fatalf("Expected ErrCredentialRevocation, got %v", errRevoked)
	}

	// 4. Cloud Run Provider
	crProv := NewCloudRunProvider()
	crProv.RegisterService("service-adk-go", map[string]string{
		"agentmesh.io/managed": "true",
	}, map[string]any{"traffic": 100})

	actCR := &spec.AgentOptimizationAction{
		ActionID:      "act-cr1",
		TargetID:      "service-adk-go",
		CurrentState:  map[string]any{"traffic": 100},
		ProposedState: map[string]any{"traffic": 80},
	}
	appCR, err := crProv.Apply(ctx, actCR)
	if err != nil || !appCR.Success {
		t.Fatalf("Cloud Run apply failed: %v", err)
	}
}
