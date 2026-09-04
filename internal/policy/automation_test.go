package policy

import (
	"testing"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

func TestEvaluateActionPolicy(t *testing.T) {
	pol := &AutomationPolicy{
		ID:             "pol-1",
		OrganizationID: "org-1",
		Mode:           ModeGuardedAutomation,
		Allow: []spec.ActionType{
			spec.ActionChangeRouteWeight,
		},
		ApprovalRequired: []spec.ActionType{
			spec.ActionChangeModelTarget,
		},
		Deny: []spec.ActionType{
			spec.ActionDisableAgent,
		},
		BlastRadius: BlastRadiusRule{
			MaxCanaryPercent: 25,
			MaxSpendUSD:      100.0,
		},
	}

	// 1. Allowed action in Guarded mode
	act1 := &spec.AgentOptimizationAction{
		ActionType: spec.ActionChangeRouteWeight,
		RiskClass:  spec.RiskLow,
		BlastRadius: spec.BlastRadius{
			TrafficPercent: 10,
			MaxCostUSD:     10.0,
		},
	}
	dec1 := EvaluateActionPolicy(act1, pol)
	if dec1.Status != PolicyStatusApproved {
		t.Fatalf("Expected APPROVED for route weight change, got %s: %s", dec1.Status, dec1.Reason)
	}

	// 2. Action requiring approval in Guarded mode
	act2 := &spec.AgentOptimizationAction{
		ActionType: spec.ActionChangeModelTarget,
		RiskClass:  spec.RiskMedium,
		BlastRadius: spec.BlastRadius{
			TrafficPercent: 10,
		},
	}
	dec2 := EvaluateActionPolicy(act2, pol)
	if dec2.Status != PolicyStatusRequiresApproval {
		t.Fatalf("Expected REQUIRES_APPROVAL for model change, got %s", dec2.Status)
	}

	// 3. Denied action
	act3 := &spec.AgentOptimizationAction{
		ActionType: spec.ActionDisableAgent,
		RiskClass:  spec.RiskHigh,
	}
	dec3 := EvaluateActionPolicy(act3, pol)
	if dec3.Status != PolicyStatusDenied {
		t.Fatalf("Expected DENIED for disable agent, got %s", dec3.Status)
	}

	// 4. Blast radius exceeding cap forces approval
	act4 := &spec.AgentOptimizationAction{
		ActionType: spec.ActionChangeRouteWeight,
		RiskClass:  spec.RiskLow,
		BlastRadius: spec.BlastRadius{
			TrafficPercent: 50, // exceeds max 25%
		},
	}
	dec4 := EvaluateActionPolicy(act4, pol)
	if dec4.Status != PolicyStatusRequiresApproval {
		t.Fatalf("Expected REQUIRES_APPROVAL when blast radius exceeded, got %s", dec4.Status)
	}

	// 5. Freeze stops automation
	pol.Frozen = true
	dec5 := EvaluateActionPolicy(act1, pol)
	if dec5.Status != PolicyStatusDenied {
		t.Fatalf("Expected DENIED when policy is frozen, got %s", dec5.Status)
	}
}

func TestFreezeManager(t *testing.T) {
	mgr := NewFreezeManager()

	frozen, _ := mgr.IsFrozen("org-1", "proj-1", "cap-1")
	if frozen {
		t.Fatalf("Expected not frozen initially")
	}

	mgr.Freeze("TENANT", "org-1", "security incident", "sec-admin", nil)
	frozen, reason := mgr.IsFrozen("org-1", "proj-1", "cap-1")
	if !frozen {
		t.Fatalf("Expected frozen after tenant freeze")
	}
	if reason == "" {
		t.Fatalf("Expected non-empty freeze reason")
	}

	// Other tenant should not be frozen
	frozenOther, _ := mgr.IsFrozen("org-2", "proj-1", "cap-1")
	if frozenOther {
		t.Fatalf("Expected org-2 not to be frozen")
	}

	mgr.Unfreeze("TENANT", "org-1")
	frozen, _ = mgr.IsFrozen("org-1", "proj-1", "cap-1")
	if frozen {
		t.Fatalf("Expected not frozen after unfreeze")
	}

	// Global kill switch
	mgr.Freeze("GLOBAL", "all", "fleet-wide maintenance", "ops", nil)
	frozen, _ = mgr.IsFrozen("any-org", "any-proj", "any-cap")
	if !frozen {
		t.Fatalf("Expected frozen under global kill switch")
	}
}
