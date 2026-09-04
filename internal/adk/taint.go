package adk

import (
	"fmt"
	"strings"

	"github.com/agentmesh/agentmesh/pkg/contracts"
)

// SecurityContext tracks authorization boundaries along a multi-hop delegation chain.
type SecurityContext struct {
	OriginalPrincipal     string              `json:"originalPrincipal"`
	CurrentAgent          string              `json:"currentAgent"`
	DelegationChain       []string            `json:"delegationChain"`
	EffectiveCapabilities []string            `json:"effectiveCapabilities"`
	EffectiveRestrictions map[string][]string `json:"effectiveRestrictions"` // agent -> deniedTools
	MaxAllowedDepth       int                 `json:"maxAllowedDepth"`
}

// NewSecurityContext initializes a root delegation context.
func NewSecurityContext(principal string, maxDepth int) *SecurityContext {
	return &SecurityContext{
		OriginalPrincipal:     principal,
		CurrentAgent:          principal,
		DelegationChain:       []string{principal},
		EffectiveCapabilities: make([]string, 0),
		EffectiveRestrictions: make(map[string][]string),
		MaxAllowedDepth:       maxDepth,
	}
}

// PushDelegation propagates security context to a downstream peer.
// Invariant: Delegation can only narrow or maintain restrictions, NEVER expand privileges.
func (sc *SecurityContext) PushDelegation(nextAgent string, nextContract *contracts.AgentContract) error {
	// Check depth
	if len(sc.DelegationChain) >= sc.MaxAllowedDepth {
		return fmt.Errorf("delegation depth exceeded: chain length %d >= max %d", len(sc.DelegationChain), sc.MaxAllowedDepth)
	}

	// Cycle check
	for _, agent := range sc.DelegationChain {
		if agent == nextAgent {
			return fmt.Errorf("delegation cycle detected: %s is already in chain %v", nextAgent, sc.DelegationChain)
		}
	}

	sc.DelegationChain = append(sc.DelegationChain, nextAgent)
	sc.CurrentAgent = nextAgent

	// Inherit restrictions from contract
	if nextContract != nil && len(nextContract.Tools.Deny) > 0 {
		sc.EffectiveRestrictions[nextAgent] = append(sc.EffectiveRestrictions[nextAgent], nextContract.Tools.Deny...)
	}

	return nil
}

// CanExecuteTool enforces anti-privilege escalation across the entire delegation chain.
// If ANY agent in the chain is barred from executing the tool, execution is DENIED.
func (sc *SecurityContext) CanExecuteTool(toolName string, originContract *contracts.AgentContract) (bool, string) {
	// Check origin principal restrictions first (Confused Deputy defense)
	if originContract != nil {
		for _, denied := range originContract.Tools.Deny {
			if matchesPattern(denied, toolName) {
				return false, fmt.Sprintf("confused deputy blocked: root agent %q is explicitly barred from tool %q", sc.OriginalPrincipal, toolName)
			}
		}
	}

	// Check restrictions on all intermediate agents
	for agent, deniedList := range sc.EffectiveRestrictions {
		for _, denied := range deniedList {
			if matchesPattern(denied, toolName) {
				return false, fmt.Sprintf("delegation taint blocked: upstream agent %q in chain is barred from tool %q", agent, toolName)
			}
		}
	}

	return true, "ALLOWED"
}

func matchesPattern(pattern, target string) bool {
	if pattern == "*" || pattern == target {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(target, prefix+".")
	}
	return false
}
