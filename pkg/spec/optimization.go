package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// ActionType defines the discrete set of permitted optimization operations.
type ActionType string

const (
	ActionChangeRouteWeight    ActionType = "CHANGE_ROUTE_WEIGHT"
	ActionPinAgent             ActionType = "PIN_AGENT"
	ActionUnpinAgent           ActionType = "UNPIN_AGENT"
	ActionChangeAgentVersion   ActionType = "CHANGE_AGENT_VERSION"
	ActionChangeModelTarget    ActionType = "CHANGE_MODEL_TARGET"
	ActionChangeModelFallback  ActionType = "CHANGE_MODEL_FALLBACK"
	ActionChangeToolProvider   ActionType = "CHANGE_TOOL_PROVIDER"
	ActionChangeProxyConfig    ActionType = "CHANGE_PROXY_CONFIG"
	ActionChangeCanaryPercent  ActionType = "CHANGE_CANARY_PERCENT"
	ActionPromoteAgentVersion  ActionType = "PROMOTE_AGENT_VERSION"
	ActionRollbackAgentVersion ActionType = "ROLLBACK_AGENT_VERSION"
	ActionDisableAgent         ActionType = "DISABLE_AGENT"
	ActionEnableAgent          ActionType = "ENABLE_AGENT"
	ActionChangeSLORoutePolicy ActionType = "CHANGE_SLO_ROUTE_POLICY"
	ActionNoop                 ActionType = "NOOP"
)

// RiskClass defines the risk tier of an optimization action.
type RiskClass string

const (
	RiskLow      RiskClass = "LOW"
	RiskMedium   RiskClass = "MEDIUM"
	RiskHigh     RiskClass = "HIGH"
	RiskCritical RiskClass = "CRITICAL"
)

// BlastRadius models the boundaries of impact for an action.
type BlastRadius struct {
	TrafficPercent       int      `json:"trafficPercent"`
	CapabilitiesAffected []string `json:"capabilitiesAffected"`
	Regions              []string `json:"regions"`
	AgentsAffected       []string `json:"agentsAffected"`
	ToolsAffected        []string `json:"toolsAffected"`
	EstimatedTasks       int64    `json:"estimatedTasks"`
	MaxCostUSD           float64  `json:"maxCostUsd"`
}

// RollbackPlan defines deterministic steps to restore the prior state.
type RollbackPlan struct {
	TargetLastKnownGood string   `json:"targetLastKnownGood"`
	Steps               []string `json:"steps"`
	ProviderCapability  string   `json:"providerCapability"`
	ValidationCriteria  string   `json:"validationCriteria"`
	ExpectedDurationMs  int64    `json:"expectedDurationMs"`
}

// ApprovalRequirement specifies human governance constraints.
type ApprovalRequirement struct {
	Required        bool      `json:"required"`
	TwoPerson       bool      `json:"twoPerson"`
	RequiredRole    string    `json:"requiredRole,omitempty"`
	ApprovedBy      []string  `json:"approvedBy,omitempty"`
	ApprovedAt      time.Time `json:"approvedAt,omitempty"`
	ActionHashBound string    `json:"actionHashBound,omitempty"`
}

// AgentOptimizationAction represents a canonical, typed optimization request.
type AgentOptimizationAction struct {
	ActionID                string              `json:"actionId"`
	OrganizationID          string              `json:"organizationId"`
	ProjectID               string              `json:"projectId"`
	CapabilityID            string              `json:"capabilityId"`
	TargetType              string              `json:"targetType"` // AGENT, ROUTE, MODEL, TOOL, PROXY
	TargetID                string              `json:"targetId"`
	ActionType              ActionType          `json:"actionType"`
	CurrentState            map[string]any      `json:"currentState"`
	ProposedState           map[string]any      `json:"proposedState"`
	Reason                  string              `json:"reason"`
	Evidence                map[string]any      `json:"evidence"`
	Confidence              float64             `json:"confidence"`
	PolicyVersion           string              `json:"policyVersion"`
	RouteAlgorithmVersion   string              `json:"routeAlgorithmVersion"`
	RiskClass               RiskClass           `json:"riskClass"`
	BlastRadius             BlastRadius         `json:"blastRadius"`
	CostDeltaEstimate       float64             `json:"costDeltaEstimate"`
	LatencyDeltaEstimateMs  int64               `json:"latencyDeltaEstimateMs"`
	QualityDeltaEstimate    float64             `json:"qualityDeltaEstimate"`
	ReliabilityDeltaEstimate float64            `json:"reliabilityDeltaEstimate"`
	RollbackPlan            RollbackPlan        `json:"rollbackPlan"`
	ApprovalRequirement     ApprovalRequirement `json:"approvalRequirement"`
	CreatedAt               time.Time           `json:"createdAt"`
	ApprovedAt              *time.Time          `json:"approvedAt,omitempty"`
	StartedAt               *time.Time          `json:"startedAt,omitempty"`
	CompletedAt             *time.Time          `json:"completedAt,omitempty"`
	Result                  string              `json:"result,omitempty"` // SUCCESS, FAILED, ROLLED_BACK
	SchemaVersion           string              `json:"schemaVersion"`
}

// ComputeActionHash returns an immutable, deterministic SHA-256 fingerprint of the action's core parameters.
// This hash binds approvals cryptographically; if any parameter changes, the hash changes.
func (a *AgentOptimizationAction) ComputeActionHash() string {
	h := sha256.New()
	content := fmt.Sprintf("%s:%s:%s:%s:%s:%s:%v:%v:%s:%s:%s",
		a.OrganizationID,
		a.ProjectID,
		a.CapabilityID,
		a.TargetType,
		a.TargetID,
		a.ActionType,
		a.CurrentState,
		a.ProposedState,
		a.PolicyVersion,
		a.RouteAlgorithmVersion,
		a.RiskClass,
	)
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}
