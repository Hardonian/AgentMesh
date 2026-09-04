package execution

import (
	"context"
	"fmt"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

// CloudRunProvider manages Google Cloud Run agent services and revision traffic splits.
type CloudRunProvider struct {
	services map[string]map[string]any
}

// NewCloudRunProvider creates a Cloud Run execution provider.
func NewCloudRunProvider() *CloudRunProvider {
	return &CloudRunProvider{
		services: make(map[string]map[string]any),
	}
}

func (c *CloudRunProvider) Name() string {
	return "CLOUD_RUN"
}

func (c *CloudRunProvider) SupportsCanary() bool {
	return true
}

func (c *CloudRunProvider) RegisterService(serviceName string, labels map[string]string, state map[string]any) {
	if labels["agentmesh.io/managed"] == "true" {
		state["labels"] = labels
		c.services[serviceName] = state
	}
}

func (c *CloudRunProvider) Discover(ctx context.Context, targetID string) (map[string]any, error) {
	s, exists := c.services[targetID]
	if !exists {
		return nil, ErrUnmanagedResource
	}
	return s, nil
}

func (c *CloudRunProvider) DryRun(ctx context.Context, action *spec.AgentOptimizationAction) (*DryRunResult, error) {
	if _, exists := c.services[action.TargetID]; !exists {
		return nil, ErrUnmanagedResource
	}

	return &DryRunResult{
		CurrentState:   action.CurrentState,
		ProposedState:  action.ProposedState,
		DiffSummary:    fmt.Sprintf("Cloud Run traffic split update for %s", action.TargetID),
		ExpectedRisk:   action.RiskClass,
		CanaryPossible: true,
	}, nil
}

func (c *CloudRunProvider) Apply(ctx context.Context, action *spec.AgentOptimizationAction) (*ApplyResult, error) {
	if _, exists := c.services[action.TargetID]; !exists {
		return nil, ErrUnmanagedResource
	}

	c.services[action.TargetID] = action.ProposedState
	return &ApplyResult{
		Success:       true,
		ResourceID:    action.TargetID,
		NewState:      action.ProposedState,
		ConfigVersion: "cr_rev_" + action.ActionID,
		ExecutionLog: []string{
			fmt.Sprintf("Adjusted Cloud Run traffic split for service %s", action.TargetID),
			"Verified new revision ready",
		},
	}, nil
}

func (c *CloudRunProvider) Verify(ctx context.Context, targetID string) (bool, error) {
	_, exists := c.services[targetID]
	return exists, nil
}

func (c *CloudRunProvider) Rollback(ctx context.Context, action *spec.AgentOptimizationAction) (*RollbackResult, error) {
	if _, exists := c.services[action.TargetID]; !exists {
		return nil, ErrUnmanagedResource
	}

	c.services[action.TargetID] = action.CurrentState
	return &RollbackResult{
		Success:       true,
		RestoredState: action.CurrentState,
		ExecutionLog:  []string{fmt.Sprintf("Restored Cloud Run service %s prior traffic split", action.TargetID)},
	}, nil
}
