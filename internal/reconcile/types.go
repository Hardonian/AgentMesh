package reconcile

import (
	"errors"
	"time"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

// WorkflowState represents durable execution lifecycle states.
type WorkflowState string

const (
	StatePlanned            WorkflowState = "PLANNED"
	StatePolicyChecked      WorkflowState = "POLICY_CHECKED"
	StateWaitingForApproval WorkflowState = "WAITING_FOR_APPROVAL"
	StateApproved           WorkflowState = "APPROVED"
	StateSimulating         WorkflowState = "SIMULATING"
	StateShadowing          WorkflowState = "SHADOWING"
	StateCanarying          WorkflowState = "CANARYING"
	StatePromoting          WorkflowState = "PROMOTING"
	StateVerifying          WorkflowState = "VERIFYING"
	StateCompleted          WorkflowState = "COMPLETED"
	StateRollingBack        WorkflowState = "ROLLING_BACK"
	StateRolledBack         WorkflowState = "ROLLED_BACK"
	StateFailed             WorkflowState = "FAILED"
	StateCanceled           WorkflowState = "CANCELED"
	StatePaused             WorkflowState = "PAUSED"
)

var (
	ErrInvalidStateTransition = errors.New("invalid workflow state transition")
	ErrWorkflowLocked         = errors.New("target is locked by another active workflow")
	ErrStaleApproval          = errors.New("action parameters modified after approval; approval invalidated")
	ErrWorkflowNotFound       = errors.New("action workflow not found")
)

// ValidTransitions maps allowed state progressions.
var ValidTransitions = map[WorkflowState][]WorkflowState{
	StatePlanned:            {StatePolicyChecked, StateCanceled},
	StatePolicyChecked:      {StateWaitingForApproval, StateApproved, StateSimulating, StateFailed, StateCanceled},
	StateWaitingForApproval: {StateApproved, StateCanceled},
	StateApproved:           {StateSimulating, StateShadowing, StateCanarying, StatePromoting, StateRollingBack, StateCanceled},
	StateSimulating:         {StateShadowing, StateCanarying, StatePromoting, StateRollingBack, StateFailed, StatePaused},
	StateShadowing:          {StateCanarying, StatePromoting, StateRollingBack, StateFailed, StatePaused},
	StateCanarying:          {StatePromoting, StateRollingBack, StateFailed, StatePaused},
	StatePromoting:          {StateVerifying, StateRollingBack, StateFailed},
	StateVerifying:          {StateCompleted, StateRollingBack, StateFailed},
	StateRollingBack:        {StateRolledBack, StateFailed},
	StatePaused:             {StateCanarying, StatePromoting, StateRollingBack, StateCanceled},
	StateCompleted:          {},
	StateRolledBack:         {},
	StateFailed:             {},
	StateCanceled:           {},
}

// ActionStep represents an ordered execution step in an optimization workflow.
type ActionStep struct {
	StepNumber   int       `json:"stepNumber"`
	Name         string    `json:"name"`
	Status       string    `json:"status"` // PENDING, RUNNING, COMPLETED, FAILED, SKIPPED
	StartedAt    time.Time `json:"startedAt"`
	CompletedAt  time.Time `json:"completedAt"`
	Error        string    `json:"error,omitempty"`
	RollbackStep string    `json:"rollbackStep,omitempty"`
}

// ActionWorkflow records the durable progress of an optimization action.
type ActionWorkflow struct {
	WorkflowID          string                        `json:"workflowId"`
	OrganizationID      string                        `json:"organizationId"`
	ProjectID           string                        `json:"projectId"`
	ActionID            string                        `json:"actionId"`
	Action              *spec.AgentOptimizationAction `json:"action"`
	ActionHash          string                        `json:"actionHash"`
	CurrentState        WorkflowState                 `json:"currentState"`
	Steps               []ActionStep                  `json:"steps"`
	CurrentStepIndex    int                           `json:"currentStepIndex"`
	TargetLockKey       string                        `json:"targetLockKey"`
	ApprovedBy          string                        `json:"approvedBy,omitempty"`
	ApprovedActionHash  string                        `json:"approvedActionHash,omitempty"`
	ErrorMessage        string                        `json:"errorMessage,omitempty"`
	LastEvaluatedAt     time.Time                     `json:"lastEvaluatedAt"`
	CreatedAt           time.Time                     `json:"createdAt"`
	UpdatedAt           time.Time                     `json:"updatedAt"`
}

// IsValidTransition checks if moving from current to next state is permissible.
func IsValidTransition(current, next WorkflowState) bool {
	allowed, exists := ValidTransitions[current]
	if !exists {
		return false
	}
	for _, s := range allowed {
		if s == next {
			return true
		}
	}
	return false
}
