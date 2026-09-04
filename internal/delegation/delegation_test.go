package delegation_test

import (
	"errors"
	"testing"

	"github.com/agentmesh/agentmesh/internal/delegation"
	"github.com/agentmesh/agentmesh/internal/policy"
)

func TestDelegationCycleDetection(t *testing.T) {
	chain := delegation.NewChain("procurement-agent", 5)

	if err := chain.Push("finance-agent", "budget_check", nil); err != nil {
		t.Fatalf("unexpected error pushing finance-agent: %v", err)
	}

	if err := chain.Push("research-agent", "market_rate", nil); err != nil {
		t.Fatalf("unexpected error pushing research-agent: %v", err)
	}

	// Attempt to cycle back to procurement-agent
	err := chain.Push("procurement-agent", "cycle_attempt", nil)
	if !errors.Is(err, delegation.ErrCycleDetected) {
		t.Fatalf("expected ErrCycleDetected, got: %v", err)
	}
}

func TestDelegationMaxDepth(t *testing.T) {
	chain := delegation.NewChain("agent-0", 2) // Max depth 2 (stack size 2: root + 1 hop)

	if err := chain.Push("agent-1", "work", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := chain.Push("agent-2", "extra_work", nil)
	if !errors.Is(err, delegation.ErrMaxDepthExceeded) {
		t.Fatalf("expected ErrMaxDepthExceeded, got: %v", err)
	}
}

func TestAntiPrivilegeEscalation(t *testing.T) {
	// Policy allows finance-agent to execute payment, but DENIES research-agent
	pol := &policy.Policy{
		ID:       "pol_security",
		TenantID: "acme",
		Rules: []policy.Rule{
			{
				Name:    "Allow finance-agent payment",
				Effect:  policy.EffectAllow,
				Agents:  []string{"finance-agent"},
				Tools:   []string{"payment.execute"},
				Actions: []string{"execute"},
			},
			{
				Name:    "Explicit deny research-agent payment",
				Effect:  policy.EffectDeny,
				Agents:  []string{"research-agent"},
				Tools:   []string{"payment.execute"},
				Actions: []string{"execute"},
			},
		},
	}
	engine := policy.NewEngine([]*policy.Policy{pol})

	// Delegation path: research-agent -> finance-agent
	chain := delegation.NewChain("research-agent", 5)
	_ = chain.Push("finance-agent", "make_payment", nil)

	// Even though finance-agent is allowed payment.execute, the origin research-agent is denied!
	err := chain.CheckPrivilegeEscalation(engine, "acme", "payment.execute", "execute")
	if !errors.Is(err, delegation.ErrPrivilegeEscalation) {
		t.Fatalf("expected ErrPrivilegeEscalation, got: %v", err)
	}
}
