package optimize

import (
	"testing"
	"time"
)

func TestSchedulerContinuousOptimization(t *testing.T) {
	sched := NewScheduler(10.0, 50, 100*time.Millisecond)

	// 1. Trivial improvement below threshold -> NO_CHANGE
	res1 := sched.EvaluateOpportunity(
		"org-test", "proj-test", "cap-finance",
		"agent-current", 0.050, 1000,
		"agent-candidate", 0.049, 990, // < 10% cost, < 50ms latency
		0.99,
	)
	if res1.Status != StatusNoChange {
		t.Fatalf("Expected StatusNoChange for trivial improvement, got %s", res1.Status)
	}

	// 2. Significant improvement (20% cost savings) -> ACTION_ELIGIBLE
	res2 := sched.EvaluateOpportunity(
		"org-test", "proj-test", "cap-finance",
		"agent-current", 0.050, 1000,
		"agent-candidate", 0.040, 850, // 20% savings, 150ms faster
		0.99,
	)
	if res2.Status != StatusActionEligible {
		t.Fatalf("Expected StatusActionEligible, got %s: %s", res2.Status, res2.Reason)
	}
	if res2.ActionProposal == nil {
		t.Fatalf("Expected non-nil action proposal")
	}

	// 3. Cooldown prevents oscillation
	sched.RecordMutationTimestamp("cap-finance")
	resCooldown := sched.EvaluateOpportunity(
		"org-test", "proj-test", "cap-finance",
		"agent-current", 0.050, 1000,
		"agent-candidate", 0.030, 800,
		0.99,
	)
	if resCooldown.Status != StatusNoChange {
		t.Fatalf("Expected StatusNoChange during cooldown, got %s", resCooldown.Status)
	}

	// Wait for cooldown to expire
	time.Sleep(120 * time.Millisecond)
	resAfterCooldown := sched.EvaluateOpportunity(
		"org-test", "proj-test", "cap-finance",
		"agent-current", 0.050, 1000,
		"agent-candidate", 0.030, 800,
		0.99,
	)
	if resAfterCooldown.Status != StatusActionEligible {
		t.Fatalf("Expected StatusActionEligible after cooldown, got %s", resAfterCooldown.Status)
	}
}
