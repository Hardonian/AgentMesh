package policy

import (
	"fmt"
	"time"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

// ExecutionMode defines the level of automation allowed for a tenant or project.
type ExecutionMode string

const (
	ModeAdvisory             ExecutionMode = "ADVISORY"              // Default: No mutations, recommendations only
	ModeApprovalRequired     ExecutionMode = "APPROVAL_REQUIRED"     // Prepares action; human must approve exact hash
	ModeGuardedAutomation    ExecutionMode = "GUARDED_AUTOMATION"    // Executes low-risk actions matching policy automatically
	ModeFullPolicyAutomation ExecutionMode = "FULL_POLICY_AUTOMATION" // Full automated progression for mature tenants
)

// AutomationPolicy defines declarative constraints on autonomous operations.
type AutomationPolicy struct {
	ID               string           `json:"id" yaml:"id"`
	OrganizationID   string           `json:"organizationId" yaml:"organizationId"`
	ProjectID        string           `json:"projectId" yaml:"projectId"`
	Mode             ExecutionMode    `json:"mode" yaml:"mode"`
	Allow            []spec.ActionType `json:"allow" yaml:"allow"`
	ApprovalRequired []spec.ActionType `json:"approvalRequired" yaml:"approvalRequired"`
	Deny             []spec.ActionType `json:"deny" yaml:"deny"`
	Requirements     RequirementsRule `json:"requirements" yaml:"requirements"`
	BlastRadius      BlastRadiusRule  `json:"blastRadius" yaml:"blastRadius"`
	Economics        EconomicsRule    `json:"economics" yaml:"economics"`
	Quality          QualityRule      `json:"quality" yaml:"quality"`
	ChangeWindows    []ChangeWindow   `json:"changeWindows,omitempty" yaml:"changeWindows,omitempty"`
	Frozen           bool             `json:"frozen" yaml:"frozen"`
	CreatedAt        time.Time        `json:"createdAt" yaml:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt" yaml:"updatedAt"`
}

// RequirementsRule specifies operational evidence gates.
type RequirementsRule struct {
	MinReliability   float64 `json:"minReliability" yaml:"minReliability"`     // e.g. 0.99
	MinEvalPassRate  float64 `json:"minEvalPassRate" yaml:"minEvalPassRate"`   // e.g. 0.97
	MinSampleCount   int64   `json:"minSampleCount" yaml:"minSampleCount"`     // e.g. 50
}

// BlastRadiusRule caps the scope of autonomous changes.
type BlastRadiusRule struct {
	MaxCanaryPercent int     `json:"maxCanaryPercent" yaml:"maxCanaryPercent"` // e.g. 25
	MaxSpendUSD      float64 `json:"maxSpendUsd" yaml:"maxSpendUsd"`           // e.g. 100.0
	MaxRegions       int     `json:"maxRegions" yaml:"maxRegions"`             // e.g. 1
}

// EconomicsRule enforces financial improvement thresholds.
type EconomicsRule struct {
	MinCostImprovementPercent float64 `json:"minCostImprovementPercent" yaml:"minCostImprovementPercent"` // e.g. 10.0
}

// QualityRule enforces latency and accuracy bounds.
type QualityRule struct {
	MaxRegressionPercent float64 `json:"maxRegressionPercent" yaml:"maxRegressionPercent"` // e.g. 1.0
}

// ChangeWindow defines permissible times for production changes.
type ChangeWindow struct {
	DayOfWeek int `json:"dayOfWeek"` // 0 = Sunday, 1 = Monday, etc.
	StartHour int `json:"startHour"` // UTC hour 0-23
	EndHour   int `json:"endHour"`   // UTC hour 0-23
}

// PolicyEvaluationStatus indicates the outcome of an action policy check.
type PolicyEvaluationStatus string

const (
	PolicyStatusApproved         PolicyEvaluationStatus = "APPROVED"
	PolicyStatusRequiresApproval PolicyEvaluationStatus = "REQUIRES_APPROVAL"
	PolicyStatusDenied           PolicyEvaluationStatus = "DENIED"
)

// ActionPolicyDecision provides the deterministic verdict and explanation.
type ActionPolicyDecision struct {
	Status    PolicyEvaluationStatus `json:"status"`
	Reason    string                 `json:"reason"`
	RulesHit  []string               `json:"rulesHit"`
	Timestamp time.Time              `json:"timestamp"`
}

// EvaluateActionPolicy evaluates an AgentOptimizationAction against an AutomationPolicy.
// Fails closed: Denials and unlisted actions in non-full modes default to REQUIRE_APPROVAL or DENY.
func EvaluateActionPolicy(action *spec.AgentOptimizationAction, pol *AutomationPolicy) *ActionPolicyDecision {
	now := time.Now().UTC()

	// 1. Check if automation is frozen globally or for project
	if pol.Frozen {
		// Emergency rollbacks may be permitted if specifically marked
		if action.ActionType == spec.ActionRollbackAgentVersion {
			return &ActionPolicyDecision{
				Status:    PolicyStatusApproved,
				Reason:    "emergency rollback allowed while frozen",
				RulesHit:  []string{"freeze_override_rollback"},
				Timestamp: now,
			}
		}
		return &ActionPolicyDecision{
			Status:    PolicyStatusDenied,
			Reason:    "automation is frozen for this organization/project",
			RulesHit:  []string{"kill_switch_active"},
			Timestamp: now,
		}
	}

	// 2. Check explicit Deny list (Deny always wins)
	for _, deniedAction := range pol.Deny {
		if action.ActionType == deniedAction {
			return &ActionPolicyDecision{
				Status:    PolicyStatusDenied,
				Reason:    fmt.Sprintf("action %s is explicitly denied by automation policy", action.ActionType),
				RulesHit:  []string{"deny_list"},
				Timestamp: now,
			}
		}
	}

	// 3. Check Blast Radius Constraints
	if pol.BlastRadius.MaxCanaryPercent > 0 && action.BlastRadius.TrafficPercent > pol.BlastRadius.MaxCanaryPercent {
		return &ActionPolicyDecision{
			Status: PolicyStatusRequiresApproval,
			Reason: fmt.Sprintf("blast radius traffic percent (%d%%) exceeds policy limit (%d%%)",
				action.BlastRadius.TrafficPercent, pol.BlastRadius.MaxCanaryPercent),
			RulesHit:  []string{"blast_radius_traffic_cap"},
			Timestamp: now,
		}
	}

	if pol.BlastRadius.MaxSpendUSD > 0 && action.BlastRadius.MaxCostUSD > pol.BlastRadius.MaxSpendUSD {
		return &ActionPolicyDecision{
			Status: PolicyStatusRequiresApproval,
			Reason: fmt.Sprintf("estimated cost ($%.2f) exceeds policy limit ($%.2f)",
				action.BlastRadius.MaxCostUSD, pol.BlastRadius.MaxSpendUSD),
			RulesHit:  []string{"blast_radius_cost_cap"},
			Timestamp: now,
		}
	}

	// 4. Check Risk Class: CRITICAL risk always requires approval
	if action.RiskClass == spec.RiskCritical {
		return &ActionPolicyDecision{
			Status:    PolicyStatusRequiresApproval,
			Reason:    "critical risk actions require explicit human approval",
			RulesHit:  []string{"critical_risk_requires_approval"},
			Timestamp: now,
		}
	}

	// 5. Check Execution Mode
	switch pol.Mode {
	case ModeAdvisory:
		return &ActionPolicyDecision{
			Status:    PolicyStatusRequiresApproval,
			Reason:    "organization mode is ADVISORY; all mutations require human approval",
			RulesHit:  []string{"mode_advisory"},
			Timestamp: now,
		}

	case ModeApprovalRequired:
		return &ActionPolicyDecision{
			Status:    PolicyStatusRequiresApproval,
			Reason:    "organization mode is APPROVAL_REQUIRED",
			RulesHit:  []string{"mode_approval_required"},
			Timestamp: now,
		}

	case ModeGuardedAutomation:
		// Check explicit ApprovalRequired list
		for _, appAction := range pol.ApprovalRequired {
			if action.ActionType == appAction {
				return &ActionPolicyDecision{
					Status:    PolicyStatusRequiresApproval,
					Reason:    fmt.Sprintf("action %s requires explicit approval in guarded mode", action.ActionType),
					RulesHit:  []string{"approval_required_list"},
					Timestamp: now,
				}
			}
		}

		// Check if in Allow list
		for _, allowedAction := range pol.Allow {
			if action.ActionType == allowedAction {
				return &ActionPolicyDecision{
					Status:    PolicyStatusApproved,
					Reason:    fmt.Sprintf("action %s is pre-approved by guarded automation policy", action.ActionType),
					RulesHit:  []string{"allow_list"},
					Timestamp: now,
				}
			}
		}

		// Default fallback for guarded mode
		return &ActionPolicyDecision{
			Status:    PolicyStatusRequiresApproval,
			Reason:    fmt.Sprintf("action %s not in allow list; defaulting to approval required", action.ActionType),
			RulesHit:  []string{"guarded_default"},
			Timestamp: now,
		}

	case ModeFullPolicyAutomation:
		// Check ApprovalRequired list
		for _, appAction := range pol.ApprovalRequired {
			if action.ActionType == appAction {
				return &ActionPolicyDecision{
					Status:    PolicyStatusRequiresApproval,
					Reason:    fmt.Sprintf("action %s designated for approval in full automation", action.ActionType),
					RulesHit:  []string{"approval_required_list"},
					Timestamp: now,
				}
			}
		}
		return &ActionPolicyDecision{
			Status:    PolicyStatusApproved,
			Reason:    "action authorized by full policy automation",
			RulesHit:  []string{"mode_full_policy_automation"},
			Timestamp: now,
		}

	default:
		return &ActionPolicyDecision{
			Status:    PolicyStatusRequiresApproval,
			Reason:    fmt.Sprintf("unknown mode %s; defaulting to approval required", pol.Mode),
			RulesHit:  []string{"unknown_mode_fallback"},
			Timestamp: now,
		}
	}
}
