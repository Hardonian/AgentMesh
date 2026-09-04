package execution

import (
	"context"
	"errors"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

var (
	ErrUnmanagedResource     = errors.New("resource is not managed by AgentMesh (missing agentmesh.io/managed label)")
	ErrUnsupportedAction     = errors.New("action not supported by execution provider")
	ErrProviderFailure       = errors.New("execution provider apply failed")
	ErrCredentialRevocation  = errors.New("provider credential revoked or invalid; failing closed")
)

// DryRunResult captures the expected diff and impact before applying.
type DryRunResult struct {
	CurrentState   map[string]any `json:"currentState"`
	ProposedState  map[string]any `json:"proposedState"`
	DiffSummary    string         `json:"diffSummary"`
	ExpectedRisk   spec.RiskClass `json:"expectedRisk"`
	CanaryPossible bool           `json:"canaryPossible"`
}

// ApplyResult records the outcome of a provider mutation.
type ApplyResult struct {
	Success       bool           `json:"success"`
	ResourceID    string         `json:"resourceId"`
	NewState      map[string]any `json:"newState"`
	ConfigVersion string         `json:"configVersion"`
	ExecutionLog  []string       `json:"executionLog"`
}

// RollbackResult records the restoration of prior state.
type RollbackResult struct {
	Success       bool     `json:"success"`
	RestoredState map[string]any `json:"restoredState"`
	ExecutionLog  []string `json:"executionLog"`
}

// ExecutionProvider defines the lifecycle contract for mutating execution targets.
type ExecutionProvider interface {
	Name() string
	SupportsCanary() bool
	Discover(ctx context.Context, targetID string) (map[string]any, error)
	DryRun(ctx context.Context, action *spec.AgentOptimizationAction) (*DryRunResult, error)
	Apply(ctx context.Context, action *spec.AgentOptimizationAction) (*ApplyResult, error)
	Verify(ctx context.Context, targetID string) (bool, error)
	Rollback(ctx context.Context, action *spec.AgentOptimizationAction) (*RollbackResult, error)
}
