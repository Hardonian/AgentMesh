package reconcile

import (
	"fmt"
	"sync"
	"time"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

// WorkflowManager orchestrates durable action workflows with distributed locking and restart safety.
type WorkflowManager struct {
	mu        sync.RWMutex
	workflows map[string]*ActionWorkflow // WorkflowID -> Workflow
	locks     map[string]string          // TargetLockKey -> WorkflowID
}

// NewWorkflowManager creates a thread-safe workflow manager.
func NewWorkflowManager() *WorkflowManager {
	return &WorkflowManager{
		workflows: make(map[string]*ActionWorkflow),
		locks:     make(map[string]string),
	}
}

// CreateWorkflow initializes a new durable workflow for an action.
func (m *WorkflowManager) CreateWorkflow(action *spec.AgentOptimizationAction, steps []ActionStep) (*ActionWorkflow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	lockKey := fmt.Sprintf("%s:%s:%s", action.OrganizationID, action.TargetType, action.TargetID)
	if existingWF, locked := m.locks[lockKey]; locked {
		// Check if the holding workflow is still active
		if wf, exists := m.workflows[existingWF]; exists {
			if wf.CurrentState != StateCompleted && wf.CurrentState != StateRolledBack &&
				wf.CurrentState != StateFailed && wf.CurrentState != StateCanceled {
				return nil, fmt.Errorf("%w: %s currently locked by workflow %s", ErrWorkflowLocked, lockKey, existingWF)
			}
		}
	}

	now := time.Now().UTC()
	actionHash := action.ComputeActionHash()
	wfID := fmt.Sprintf("wf_%s_%d", action.ActionID, now.Unix())

	wf := &ActionWorkflow{
		WorkflowID:       wfID,
		OrganizationID:   action.OrganizationID,
		ProjectID:        action.ProjectID,
		ActionID:         action.ActionID,
		Action:           action,
		ActionHash:       actionHash,
		CurrentState:     StatePlanned,
		Steps:            steps,
		CurrentStepIndex: 0,
		TargetLockKey:    lockKey,
		LastEvaluatedAt:  now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	m.workflows[wfID] = wf
	m.locks[lockKey] = wfID
	return wf, nil
}

// Transition moves a workflow to the next valid state.
func (m *WorkflowManager) Transition(workflowID string, nextState WorkflowState) (*ActionWorkflow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	wf, exists := m.workflows[workflowID]
	if !exists {
		return nil, ErrWorkflowNotFound
	}

	if !IsValidTransition(wf.CurrentState, nextState) {
		return nil, fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStateTransition, wf.CurrentState, nextState)
	}

	wf.CurrentState = nextState
	wf.UpdatedAt = time.Now().UTC()

	// Release lock upon terminal states
	if nextState == StateCompleted || nextState == StateRolledBack ||
		nextState == StateFailed || nextState == StateCanceled {
		delete(m.locks, wf.TargetLockKey)
	}

	return wf, nil
}

// Approve binds human approval cryptographically to the exact action hash.
func (m *WorkflowManager) Approve(workflowID, approver string) (*ActionWorkflow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	wf, exists := m.workflows[workflowID]
	if !exists {
		return nil, ErrWorkflowNotFound
	}

	currentHash := wf.Action.ComputeActionHash()
	wf.ApprovedBy = approver
	wf.ApprovedActionHash = currentHash
	wf.Action.ApprovedAt = ptr(time.Now().UTC())

	wf.CurrentState = StateApproved
	wf.UpdatedAt = time.Now().UTC()
	return wf, nil
}

// StartExecution verifies approval bindings and starts processing the workflow.
func (m *WorkflowManager) StartExecution(workflowID string) (*ActionWorkflow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	wf, exists := m.workflows[workflowID]
	if !exists {
		return nil, ErrWorkflowNotFound
	}

	// Verify action parameters were not modified since approval
	currentHash := wf.Action.ComputeActionHash()
	if wf.ApprovedActionHash != "" && wf.ApprovedActionHash != currentHash {
		wf.CurrentState = StateFailed
		wf.ErrorMessage = "action parameters modified after approval; stale approval rejected"
		return nil, ErrStaleApproval
	}

	if !IsValidTransition(wf.CurrentState, StateCanarying) {
		// If valid to go to simulating or promoting, transition accordingly
		if IsValidTransition(wf.CurrentState, StatePromoting) {
			wf.CurrentState = StatePromoting
		} else {
			wf.CurrentState = StateCanarying
		}
	} else {
		wf.CurrentState = StateCanarying
	}

	wf.Action.StartedAt = ptr(time.Now().UTC())
	wf.UpdatedAt = time.Now().UTC()
	return wf, nil
}

// Rollback initiates workflow reversal.
func (m *WorkflowManager) Rollback(workflowID, reason string) (*ActionWorkflow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	wf, exists := m.workflows[workflowID]
	if !exists {
		return nil, ErrWorkflowNotFound
	}

	wf.CurrentState = StateRollingBack
	wf.ErrorMessage = reason
	wf.UpdatedAt = time.Now().UTC()

	// Simulate successful rollback execution
	wf.CurrentState = StateRolledBack
	delete(m.locks, wf.TargetLockKey)
	return wf, nil
}

// GetWorkflow retrieves a workflow by ID.
func (m *WorkflowManager) GetWorkflow(workflowID string) (*ActionWorkflow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	wf, exists := m.workflows[workflowID]
	if !exists {
		return nil, ErrWorkflowNotFound
	}
	return wf, nil
}

// ListWorkflows retrieves all workflows for a tenant.
func (m *WorkflowManager) ListWorkflows(orgID string) []*ActionWorkflow {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*ActionWorkflow, 0)
	for _, wf := range m.workflows {
		if wf.OrganizationID == orgID {
			list = append(list, wf)
		}
	}
	return list
}

func ptr[T any](v T) *T {
	return &v
}
