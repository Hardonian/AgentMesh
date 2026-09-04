package reconcile

import (
	"errors"
	"fmt"
	"time"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

// ReconciliationPlan contains the sequence of steps and safety checks to reach desired state.
type ReconciliationPlan struct {
	PlanID               string                        `json:"planId"`
	CapabilityID         string                        `json:"capabilityId"`
	CurrentSpecHash      string                        `json:"currentSpecHash"`
	DesiredSpecHash      string                        `json:"desiredSpecHash"`
	Action               *spec.AgentOptimizationAction `json:"action"`
	Impact               *ChangeImpactReport           `json:"impact"`
	Steps                []ActionStep                  `json:"steps"`
	RollbackSteps        []string                      `json:"rollbackSteps"`
	RequiresConfirmation bool                          `json:"requiresConfirmation"`
	CreatedAt            time.Time                     `json:"createdAt"`
}

// Engine produces safe reconciliation plans to transition between states.
type Engine struct{}

// NewEngine creates a new reconciliation engine.
func NewEngine() *Engine {
	return &Engine{}
}

// PlanRoutingReconciliation builds a multi-step progressive delivery plan between current and desired routing specs.
func (e *Engine) PlanRoutingReconciliation(current, desired *spec.AgentRoutingSpec) (*ReconciliationPlan, error) {
	if current == nil || desired == nil {
		return nil, errors.New("current and desired specs cannot be nil")
	}
	if current.Weights == nil {
		current.Weights = make(map[string]int)
	}
	if desired.Weights == nil {
		desired.Weights = make(map[string]int)
	}

	now := time.Now().UTC()
	planID := fmt.Sprintf("plan_route_%s_%d", desired.CapabilityID, now.Unix())

	// Compute primary target differences
	var targetAgent string
	var currentWeight, desiredWeight int
	for agentID, dWeight := range desired.Weights {
		cWeight := current.Weights[agentID]
		if dWeight != cWeight {
			targetAgent = agentID
			currentWeight = cWeight
			desiredWeight = dWeight
			break
		}
	}

	action := &spec.AgentOptimizationAction{
		ActionID:       fmt.Sprintf("act_%s_%d", desired.CapabilityID, now.Unix()),
		OrganizationID: desired.OrganizationID,
		ProjectID:      desired.ProjectID,
		CapabilityID:   desired.CapabilityID,
		TargetType:     "ROUTE",
		TargetID:       desired.CapabilityID,
		ActionType:     spec.ActionChangeRouteWeight,
		CurrentState: map[string]any{
			"agentId": targetAgent,
			"weight":  currentWeight,
		},
		ProposedState: map[string]any{
			"agentId": targetAgent,
			"weight":  desiredWeight,
		},
		Reason:                fmt.Sprintf("reconcile routing weight for agent %s from %d%% to %d%%", targetAgent, currentWeight, desiredWeight),
		PolicyVersion:         "v1",
		RouteAlgorithmVersion: "BASELINE_ROUTER_V1",
		BlastRadius: spec.BlastRadius{
			TrafficPercent:       desiredWeight,
			CapabilitiesAffected: []string{desired.CapabilityID},
			AgentsAffected:       []string{targetAgent},
		},
		RollbackPlan: spec.RollbackPlan{
			TargetLastKnownGood: fmt.Sprintf("%s:%d", targetAgent, currentWeight),
			Steps: []string{
				fmt.Sprintf("revert traffic weight of %s back to %d%%", targetAgent, currentWeight),
				"distribute signed configuration rollback",
			},
			ProviderCapability: "PROXY_CONFIG",
			ExpectedDurationMs: 500,
		},
		CreatedAt:     now,
		SchemaVersion: "v1",
	}

	impact := AnalyzeActionImpact(action)
	action.RiskClass = impact.RiskClass

	// Build progressive delivery steps
	steps := make([]ActionStep, 0)
	stepNum := 1

	// Step 1: Pre-flight policy validation
	steps = append(steps, ActionStep{
		StepNumber:   stepNum,
		Name:         "validate_policy_eligibility",
		Status:       "PENDING",
		RollbackStep: "noop",
	})
	stepNum++

	// If candidate weight is significant, add shadow and canary progression
	if desiredWeight > 10 {
		steps = append(steps, ActionStep{
			StepNumber:   stepNum,
			Name:         "shadow_traffic_evaluation",
			Status:       "PENDING",
			RollbackStep: "disable_shadow",
		})
		stepNum++

		steps = append(steps, ActionStep{
			StepNumber:   stepNum,
			Name:         "canary_initial_stage_5pct",
			Status:       "PENDING",
			RollbackStep: fmt.Sprintf("restore_weight_%d", currentWeight),
		})
		stepNum++

		steps = append(steps, ActionStep{
			StepNumber:   stepNum,
			Name:         "canary_interim_stage_25pct",
			Status:       "PENDING",
			RollbackStep: fmt.Sprintf("restore_weight_%d", currentWeight),
		})
		stepNum++
	}

	// Final Step: Full promotion to desired weight
	steps = append(steps, ActionStep{
		StepNumber:   stepNum,
		Name:         fmt.Sprintf("promote_final_weight_%dpct", desiredWeight),
		Status:       "PENDING",
		RollbackStep: fmt.Sprintf("restore_weight_%d", currentWeight),
	})
	stepNum++

	// Verification Step: Observe metrics
	steps = append(steps, ActionStep{
		StepNumber:   stepNum,
		Name:         "observe_and_verify_stability",
		Status:       "PENDING",
		RollbackStep: fmt.Sprintf("restore_weight_%d", currentWeight),
	})

	return &ReconciliationPlan{
		PlanID:               planID,
		CapabilityID:         desired.CapabilityID,
		CurrentSpecHash:      current.ComputeSpecHash(),
		DesiredSpecHash:      desired.ComputeSpecHash(),
		Action:               action,
		Impact:               impact,
		Steps:                steps,
		RollbackSteps:        action.RollbackPlan.Steps,
		RequiresConfirmation: impact.RequiresHumanApproval,
		CreatedAt:            now,
	}, nil
}
