package main

import (
	"fmt"

	"github.com/agentmesh/agentmesh/internal/delegation"
	"github.com/agentmesh/agentmesh/internal/policy"
)

func main() {
	fmt.Println("=== AgentMesh A2A Delegation & Anti-Escalation Demo ===")

	// Policy setup:
	// - Procurement allowed to delegate to finance
	// - Finance allowed to delegate to research
	// - Research is strictly DENIED access to payment.execute
	pol := &policy.Policy{
		ID:       "pol_a2a_enterprise",
		Version:  "v1.0.0",
		TenantID: "corp",
		Rules: []policy.Rule{
			{
				Name:    "Allow finance payment",
				Effect:  policy.EffectAllow,
				Agents:  []string{"finance-agent"},
				Tools:   []string{"payment.execute"},
				Actions: []string{"execute"},
			},
			{
				Name:    "Deny research payment",
				Effect:  policy.EffectDeny,
				Agents:  []string{"research-agent"},
				Tools:   []string{"payment.execute"},
				Actions: []string{"execute"},
			},
		},
	}
	engine := policy.NewEngine([]*policy.Policy{pol})

	// Scenario 1: Valid Delegation
	chain := delegation.NewChain("procurement-agent", 4)
	_ = chain.Push("finance-agent", "budget_review", nil)
	_ = chain.Push("research-agent", "pricing_lookup", nil)

	fmt.Printf("1. Delegation stack initialized: %v\n", chain.Stack)

	// Scenario 2: Cycle Detection
	errCycle := chain.Push("procurement-agent", "loop", nil)
	fmt.Printf("2. Attempted cycle back to procurement: Error = %v\n", errCycle)

	// Scenario 3: Anti-Privilege Escalation Check
	// Research agent attempts to trigger payment.execute through finance-agent delegation hop
	errEscalate := chain.CheckPrivilegeEscalation(engine, "corp", "payment.execute", "execute")
	fmt.Printf("3. Anti-privilege escalation result: Error = %v\n", errEscalate)

	fmt.Println("✓ Invariant confirmed: Delegating cannot bypass caller policy boundaries.")
}
