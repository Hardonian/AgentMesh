package learned

import (
	"context"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/reliability"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/internal/routing/intelligence"
	"github.com/agentmesh/agentmesh/pkg/task"
)

func TestGatekeeperAndFallback(t *testing.T) {
	gate := NewGatekeeper(50, 2)
	ctx := context.Background()

	// 1. Empty outcomes -> must be DISABLED_INSUFFICIENT_DATA
	status, reason := gate.EvaluateGate(nil)
	if status != GateDisabledInsufficientData {
		t.Errorf("expected GateDisabledInsufficientData on empty outcomes, got %s (%s)", status, reason)
	}

	// 2. 20 outcomes (less than 50) -> still DISABLED_INSUFFICIENT_DATA
	outcomes20 := make([]*routing.CanonicalRoutingOutcome, 20)
	for i := 0; i < 20; i++ {
		outcomes20[i] = &routing.CanonicalRoutingOutcome{
			SelectedAgentID: "agent-a",
		}
	}
	status, _ = gate.EvaluateGate(outcomes20)
	if status != GateDisabledInsufficientData {
		t.Errorf("expected GateDisabledInsufficientData for 20 outcomes")
	}

	// 3. 50 outcomes but only 1 agent -> DISABLED_INSUFFICIENT_DATA
	outcomes50SingleAgent := make([]*routing.CanonicalRoutingOutcome, 50)
	for i := 0; i < 50; i++ {
		outcomes50SingleAgent[i] = &routing.CanonicalRoutingOutcome{
			SelectedAgentID: "agent-a",
		}
	}
	status, _ = gate.EvaluateGate(outcomes50SingleAgent)
	if status != GateDisabledInsufficientData {
		t.Errorf("expected GateDisabledInsufficientData when agent diversity < 2")
	}

	// 4. Test LearnedRouter fallback when gate is not met
	lr := NewLearnedRouter("model-v1", "1.0.0", nil)
	fp := task.NewTaskFingerprint("search", 100, 100, false, nil, "PUBLIC", "us-central1", 1000, 0.01, false, nil, false)
	candA := &intelligence.CandidateAgent{AgentID: "agent-a", HealthStatus: "HEALTHY"}

	res, err := lr.Route(ctx, fp, "acme", intelligence.ObjectiveBalanced, nil, []*intelligence.CandidateAgent{candA}, outcomes20, "v1", false)
	if err != nil {
		t.Fatalf("unexpected routing error: %v", err)
	}
	if res.AlgorithmID != "BASELINE_FALLBACK" {
		t.Errorf("expected BASELINE_FALLBACK when data insufficient, got %s", res.AlgorithmID)
	}
}

func TestLearnedRouterSufficientDataAndPolicy(t *testing.T) {
	ctx := context.Background()
	lr := NewLearnedRouter("model-v1", "1.0.0", nil)
	fp := task.NewTaskFingerprint("forecast", 500, 500, false, nil, "PUBLIC", "us-central1", 2000, 0.05, false, nil, false)

	// Create 55 outcomes with 2 distinct agents
	outcomes55 := make([]*routing.CanonicalRoutingOutcome, 55)
	for i := 0; i < 55; i++ {
		agent := "agent-a"
		if i%2 == 0 {
			agent = "agent-b"
		}
		outcomes55[i] = &routing.CanonicalRoutingOutcome{
			OutcomeID:       "out",
			SelectedAgentID: agent,
			Success:         true,
			CreatedAt:       time.Now().UTC(),
		}
	}

	candA := &intelligence.CandidateAgent{
		AgentID:      "agent-a",
		HealthStatus: "HEALTHY",
		ReliabilityProfile: &reliability.ReliabilityProfile{
			TotalSamples:       100,
			OverallSuccessRate: 0.98,
			P50LatencyMs:       400,
			AverageCostUSD:     0.02,
		},
	}
	candB := &intelligence.CandidateAgent{
		AgentID:      "agent-b",
		HealthStatus: "HEALTHY",
		ReliabilityProfile: &reliability.ReliabilityProfile{
			TotalSamples:       100,
			OverallSuccessRate: 0.85,
			P50LatencyMs:       900,
			AverageCostUSD:     0.04,
		},
	}

	// Policy: explicitly deny agent-a (learned model must not override!)
	pol := &policy.Policy{
		ID:       "pol-strict",
		TenantID: "acme",
		Rules: []policy.Rule{
			{Name: "deny-a", Effect: policy.EffectDeny, Agents: []string{"agent-a"}},
			{Name: "allow-all", Effect: policy.EffectAllow, Actions: []string{"*"}},
		},
	}
	eng := policy.NewEngine([]*policy.Policy{pol})

	res, err := lr.Route(ctx, fp, "acme", intelligence.ObjectiveBalanced, eng, []*intelligence.CandidateAgent{candA, candB}, outcomes55, "v1", false)
	if err != nil {
		t.Fatalf("unexpected routing error: %v", err)
	}
	if res.SelectedAgentID != "agent-b" {
		t.Errorf("policy override failure: expected policy DENY to exclude agent-a and select agent-b, got %s", res.SelectedAgentID)
	}
	if res.AlgorithmID != "model-v1" {
		t.Errorf("expected model-v1 algorithm when data sufficient, got %s", res.AlgorithmID)
	}
}

func TestModelRegistryPromotionAndRollback(t *testing.T) {
	reg := NewModelRegistry()
	tenant := "acme"

	m1 := &RoutingModelRecord{
		ModelID:  "model-1",
		TenantID: tenant,
		Version:  "1.0.0",
		Status:   StatusCandidate,
	}
	m2 := &RoutingModelRecord{
		ModelID:  "model-2",
		TenantID: tenant,
		Version:  "2.0.0",
		Status:   StatusCandidate,
	}

	_ = reg.RegisterModel(m1)
	_ = reg.RegisterModel(m2)

	// Promote m1
	if err := reg.Promote(tenant, "model-1"); err != nil {
		t.Fatalf("failed to promote m1: %v", err)
	}
	cur, ok := reg.GetActiveModel(tenant)
	if !ok || cur.ModelID != "model-1" {
		t.Errorf("expected model-1 active")
	}

	// Shadow mode for m2
	if err := reg.SetShadow(tenant, "model-2"); err != nil {
		t.Fatalf("failed to set shadow for m2: %v", err)
	}
	m2Record := reg.ListModels(tenant)[1]
	if m2Record.ModelID == "model-2" && m2Record.Status != StatusShadow {
		t.Errorf("expected model-2 in shadow mode")
	}

	// Promote m2
	if err := reg.Promote(tenant, "model-2"); err != nil {
		t.Fatalf("failed to promote m2: %v", err)
	}
	cur, _ = reg.GetActiveModel(tenant)
	if cur.ModelID != "model-2" {
		t.Errorf("expected model-2 active")
	}

	// Rollback to m1
	rolledBack, err := reg.Rollback(tenant)
	if err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	if rolledBack.ModelID != "model-1" {
		t.Errorf("expected rollback to model-1, got %s", rolledBack.ModelID)
	}
	cur, _ = reg.GetActiveModel(tenant)
	if cur.ModelID != "model-1" {
		t.Errorf("expected active model to be model-1 after rollback")
	}
}
