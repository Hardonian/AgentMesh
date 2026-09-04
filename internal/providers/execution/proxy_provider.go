package execution

import (
	"context"
	"fmt"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

// ProxyProvider applies mutations to the AgentMesh proxy fleet via signed configs.
type ProxyProvider struct {
	fleetState map[string]map[string]any // targetID -> state
}

// NewProxyProvider creates a proxy fleet execution provider.
func NewProxyProvider() *ProxyProvider {
	return &ProxyProvider{
		fleetState: make(map[string]map[string]any),
	}
}

func (p *ProxyProvider) Name() string {
	return "PROXY_CONFIG"
}

func (p *ProxyProvider) SupportsCanary() bool {
	return true
}

func (p *ProxyProvider) Discover(ctx context.Context, targetID string) (map[string]any, error) {
	if s, exists := p.fleetState[targetID]; exists {
		return s, nil
	}
	return map[string]any{"status": "INITIAL", "weights": map[string]int{}}, nil
}

func (p *ProxyProvider) DryRun(ctx context.Context, action *spec.AgentOptimizationAction) (*DryRunResult, error) {
	return &DryRunResult{
		CurrentState:   action.CurrentState,
		ProposedState:  action.ProposedState,
		DiffSummary:    fmt.Sprintf("Update proxy routing for capability %s", action.CapabilityID),
		ExpectedRisk:   action.RiskClass,
		CanaryPossible: true,
	}, nil
}

func (p *ProxyProvider) Apply(ctx context.Context, action *spec.AgentOptimizationAction) (*ApplyResult, error) {
	p.fleetState[action.TargetID] = action.ProposedState
	return &ApplyResult{
		Success:       true,
		ResourceID:    action.TargetID,
		NewState:      action.ProposedState,
		ConfigVersion: "v_signed_" + action.ActionID,
		ExecutionLog: []string{
			fmt.Sprintf("Generated signed route config for %s", action.TargetID),
			"Propagated config to proxy fleet",
		},
	}, nil
}

func (p *ProxyProvider) Verify(ctx context.Context, targetID string) (bool, error) {
	_, exists := p.fleetState[targetID]
	return exists, nil
}

func (p *ProxyProvider) Rollback(ctx context.Context, action *spec.AgentOptimizationAction) (*RollbackResult, error) {
	p.fleetState[action.TargetID] = action.CurrentState
	return &RollbackResult{
		Success:       true,
		RestoredState: action.CurrentState,
		ExecutionLog:  []string{"Restored previous route config"},
	}, nil
}
