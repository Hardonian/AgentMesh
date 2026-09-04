package delegation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/pkg/contracts"
)

var (
	ErrCycleDetected       = errors.New("delegation cycle detected")
	ErrMaxDepthExceeded    = errors.New("maximum delegation depth exceeded")
	ErrPrivilegeEscalation = errors.New("delegation privilege escalation: origin caller lacks permission for delegated action")
	ErrDisallowedDelegate  = errors.New("target agent is not in the caller's allowed delegation list")
)

// DelegationHop represents a single step in a multi-agent delegation path.
type DelegationHop struct {
	CallerAgentID string `json:"callerAgentId"`
	TargetAgentID string `json:"targetAgentId"`
	Capability    string `json:"capability"`
}

// Chain manages the delegation stack for an ongoing task.
type Chain struct {
	Stack      []string // Ordered list of Agent IDs from origin to current
	HopDetails []DelegationHop
	MaxDepth   int
}

// NewChain starts a new delegation chain with the origin agent.
func NewChain(originAgentID string, maxDepth int) *Chain {
	if maxDepth <= 0 {
		maxDepth = 5 // Safe default
	}
	return &Chain{
		Stack:    []string{originAgentID},
		MaxDepth: maxDepth,
	}
}

// Push attempts to add a new delegation hop from current agent to target agent.
// Validates:
// 1. No cycles (target is not already in the stack).
// 2. Max depth is not breached.
// 3. Target is in caller's allowed delegation list (if contract provided).
func (c *Chain) Push(targetAgentID, capability string, callerContract *contracts.AgentContract) error {
	// 1. Cycle Detection
	for _, agentID := range c.Stack {
		if strings.EqualFold(agentID, targetAgentID) {
			return fmt.Errorf("%w: agent %q is already in the delegation path (%s)",
				ErrCycleDetected, targetAgentID, strings.Join(c.Stack, " -> "))
		}
	}

	// 2. Depth Check
	if len(c.Stack) >= c.MaxDepth {
		return fmt.Errorf("%w: current depth %d reached limit %d",
			ErrMaxDepthExceeded, len(c.Stack), c.MaxDepth)
	}

	// 3. Contract Allowed Delegation Check
	if callerContract != nil && len(callerContract.Delegation.Allow) > 0 {
		allowed := false
		for _, allowedTarget := range callerContract.Delegation.Allow {
			if allowedTarget == "*" || strings.EqualFold(allowedTarget, targetAgentID) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("%w: caller %q is not permitted to delegate to %q",
				ErrDisallowedDelegate, c.CurrentAgent(), targetAgentID)
		}
	}

	current := c.CurrentAgent()
	c.Stack = append(c.Stack, targetAgentID)
	c.HopDetails = append(c.HopDetails, DelegationHop{
		CallerAgentID: current,
		TargetAgentID: targetAgentID,
		Capability:    capability,
	})

	return nil
}

// CurrentAgent returns the active agent at the head of the delegation stack.
func (c *Chain) CurrentAgent() string {
	if len(c.Stack) == 0 {
		return ""
	}
	return c.Stack[len(c.Stack)-1]
}

// OriginAgent returns the root agent that initiated the task.
func (c *Chain) OriginAgent() string {
	if len(c.Stack) == 0 {
		return ""
	}
	return c.Stack[0]
}

// Depth returns the current delegation depth (0 for root).
func (c *Chain) Depth() int {
	return len(c.Stack) - 1
}

// CheckPrivilegeEscalation verifies that no hop in the delegation chain grants
// access to a tool that the origin caller or intermediate agents are prohibited from using.
func (c *Chain) CheckPrivilegeEscalation(engine *policy.Engine, tenantID, tool, action string) error {
	// Check policy against the origin agent first, then all intermediaries
	for depth, agentID := range c.Stack {
		dec := engine.Evaluate(context.TODO(), &policy.EvaluationRequest{
			TenantID:        tenantID,
			SubjectAgentID:  agentID,
			Tool:            tool,
			Action:          action,
			DelegationDepth: depth,
		})

		if dec.Effect == policy.EffectDeny {
			return fmt.Errorf("%w: agent %q (at depth %d in chain [%s]) is denied tool %q (%s)",
				ErrPrivilegeEscalation, agentID, depth, strings.Join(c.Stack, " -> "), tool, dec.Reason)
		}
	}
	return nil
}
