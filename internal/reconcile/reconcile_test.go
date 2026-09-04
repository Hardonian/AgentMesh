package reconcile

import (
	"testing"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

func TestReconciliationEngineAndWorkflow(t *testing.T) {
	engine := NewEngine()
	mgr := NewWorkflowManager()

	current := &spec.AgentRoutingSpec{
		CapabilityID:   "financial_analysis",
		OrganizationID: "org-alpha",
		ProjectID:      "proj-1",
		Version:        "v1",
		Weights: map[string]int{
			"agent-a": 100,
			"agent-b": 0,
		},
	}

	desired := &spec.AgentRoutingSpec{
		CapabilityID:   "financial_analysis",
		OrganizationID: "org-alpha",
		ProjectID:      "proj-1",
		Version:        "v2",
		Weights: map[string]int{
			"agent-a": 20,
			"agent-b": 80,
		},
	}

	plan, err := engine.PlanRoutingReconciliation(current, desired)
	if err != nil {
		t.Fatalf("Failed to plan reconciliation: %v", err)
	}

	if len(plan.Steps) == 0 {
		t.Fatalf("Expected non-empty reconciliation steps")
	}

	// 1. Create durable workflow
	wf, err := mgr.CreateWorkflow(plan.Action, plan.Steps)
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	if wf.CurrentState != StatePlanned {
		t.Fatalf("Expected initial state PLANNED, got %s", wf.CurrentState)
	}

	// 2. Lock prevents concurrent workflow on same target
	_, errLocked := mgr.CreateWorkflow(plan.Action, plan.Steps)
	if errLocked == nil {
		t.Fatalf("Expected error due to target lock, got nil")
	}

	// 3. Transition to POLICY_CHECKED
	wf, err = mgr.Transition(wf.WorkflowID, StatePolicyChecked)
	if err != nil {
		t.Fatalf("Failed valid transition: %v", err)
	}

	// 4. Invalid transition directly to COMPLETED must fail
	_, errInvalid := mgr.Transition(wf.WorkflowID, StateCompleted)
	if errInvalid == nil {
		t.Fatalf("Expected invalid transition error")
	}

	// 5. Move to WAITING_FOR_APPROVAL then APPROVE
	wf, err = mgr.Transition(wf.WorkflowID, StateWaitingForApproval)
	if err != nil {
		t.Fatalf("Failed transition to waiting for approval: %v", err)
	}

	wf, err = mgr.Approve(wf.WorkflowID, "admin-user")
	if err != nil {
		t.Fatalf("Failed to approve workflow: %v", err)
	}
	if wf.CurrentState != StateApproved {
		t.Fatalf("Expected APPROVED state, got %s", wf.CurrentState)
	}

	// 6. Test Stale Approval Protection: Tamper with action after approval
	wf.Action.ProposedState["weight"] = 99
	_, errStale := mgr.StartExecution(wf.WorkflowID)
	if errStale != ErrStaleApproval {
		t.Fatalf("Expected ErrStaleApproval, got %v", errStale)
	}

	// 7. Rollback releases lock
	wfRollback, err := mgr.Rollback(wf.WorkflowID, "metric degradation")
	if err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}
	if wfRollback.CurrentState != StateRolledBack {
		t.Fatalf("Expected ROLLED_BACK state, got %s", wfRollback.CurrentState)
	}

	// Now a new workflow can acquire lock on the target
	_, errNew := mgr.CreateWorkflow(plan.Action, plan.Steps)
	if errNew != nil {
		t.Fatalf("Expected lock to be released after rollback, got %v", errNew)
	}
}
