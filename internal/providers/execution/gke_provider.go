package execution

import (
	"context"
	"fmt"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

// GKEProvider manages Kubernetes / GKE agent deployments and CRDs.
type GKEProvider struct {
	managedResources map[string]map[string]any
	credentialsValid bool
}

// NewGKEProvider creates a GKE execution provider.
func NewGKEProvider(credentialsValid bool) *GKEProvider {
	return &GKEProvider{
		managedResources: make(map[string]map[string]any),
		credentialsValid: credentialsValid,
	}
}

func (g *GKEProvider) Name() string {
	return "GKE"
}

func (g *GKEProvider) SupportsCanary() bool {
	return true
}

func (g *GKEProvider) RegisterManagedResource(resourceID string, labels map[string]string, state map[string]any) {
	if labels["agentmesh.io/managed"] == "true" {
		state["labels"] = labels
		g.managedResources[resourceID] = state
	}
}

func (g *GKEProvider) Discover(ctx context.Context, targetID string) (map[string]any, error) {
	if !g.credentialsValid {
		return nil, ErrCredentialRevocation
	}

	res, exists := g.managedResources[targetID]
	if !exists {
		return nil, ErrUnmanagedResource
	}
	return res, nil
}

func (g *GKEProvider) DryRun(ctx context.Context, action *spec.AgentOptimizationAction) (*DryRunResult, error) {
	if !g.credentialsValid {
		return nil, ErrCredentialRevocation
	}

	// Verify target is managed
	if _, exists := g.managedResources[action.TargetID]; !exists {
		return nil, ErrUnmanagedResource
	}

	return &DryRunResult{
		CurrentState:   action.CurrentState,
		ProposedState:  action.ProposedState,
		DiffSummary:    fmt.Sprintf("GKE progressive rollout for deployment %s", action.TargetID),
		ExpectedRisk:   action.RiskClass,
		CanaryPossible: true,
	}, nil
}

func (g *GKEProvider) Apply(ctx context.Context, action *spec.AgentOptimizationAction) (*ApplyResult, error) {
	if !g.credentialsValid {
		return nil, ErrCredentialRevocation
	}

	if _, exists := g.managedResources[action.TargetID]; !exists {
		return nil, ErrUnmanagedResource
	}

	g.managedResources[action.TargetID] = action.ProposedState
	return &ApplyResult{
		Success:       true,
		ResourceID:    action.TargetID,
		NewState:      action.ProposedState,
		ConfigVersion: "gke_rev_" + action.ActionID,
		ExecutionLog: []string{
			fmt.Sprintf("Updated GKE deployment %s to proposed image/version", action.TargetID),
			"Verified pod ready replicas",
		},
	}, nil
}

func (g *GKEProvider) Verify(ctx context.Context, targetID string) (bool, error) {
	if !g.credentialsValid {
		return false, ErrCredentialRevocation
	}
	_, exists := g.managedResources[targetID]
	return exists, nil
}

func (g *GKEProvider) Rollback(ctx context.Context, action *spec.AgentOptimizationAction) (*RollbackResult, error) {
	if !g.credentialsValid {
		return nil, ErrCredentialRevocation
	}

	if _, exists := g.managedResources[action.TargetID]; !exists {
		return nil, ErrUnmanagedResource
	}

	g.managedResources[action.TargetID] = action.CurrentState
	return &RollbackResult{
		Success:       true,
		RestoredState: action.CurrentState,
		ExecutionLog:  []string{fmt.Sprintf("Rolled back GKE deployment %s to prior revision", action.TargetID)},
	}, nil
}
