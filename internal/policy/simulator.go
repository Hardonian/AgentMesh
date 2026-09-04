package policy

import (
	"context"
	"fmt"
	"time"
)

// SimulationResult provides full audit transparency into a simulated policy evaluation.
type SimulationResult struct {
	Decision      *Decision `json:"decision"`
	SimulatedAt   time.Time `json:"simulatedAt"`
	MatchedPolicy string    `json:"matchedPolicy,omitempty"`
	Explanation   string    `json:"explanation"`
}

// Simulate evaluates a hypothetical request without recording audit traces or modifying state.
func (e *Engine) Simulate(ctx context.Context, req *EvaluationRequest) *SimulationResult {
	if ctx == nil {
		ctx = context.Background()
	}

	dec := e.Evaluate(ctx, req)
	expl := fmt.Sprintf("Evaluation of agent %q targeting %q (action: %q) resulted in %s: %s",
		req.SubjectAgentID, req.Tool, req.Action, dec.Effect, dec.Reason)

	return &SimulationResult{
		Decision:    dec,
		SimulatedAt: time.Now().UTC(),
		Explanation: expl,
	}
}
