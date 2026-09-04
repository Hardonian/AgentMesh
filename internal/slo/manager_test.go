package slo

import (
	"testing"

	"github.com/agentmesh/agentmesh/internal/reliability"
)

func TestSLOEvaluationAndCapabilityHealth(t *testing.T) {
	mgr := NewManager()
	tenant := "acme-corp"
	agentA := "agent-fast"
	agentB := "agent-slow"
	capID := "finance_forecast"

	// Register strict SLO for agent A
	_ = mgr.SetSLO(&AgentSLO{
		TenantID:          tenant,
		AgentID:           agentA,
		CapabilityID:      capID,
		TargetSuccessRate: 0.99,
		MaxP95LatencyMs:   2000,
	})

	// Profile for agent A (healthy)
	profA := &reliability.ReliabilityProfile{
		TenantID:           tenant,
		AgentID:            agentA,
		CapabilityID:       capID,
		TotalSamples:       50,
		OverallSuccessRate: 0.995,
		P95LatencyMs:       1500,
		AverageCostUSD:     0.02,
	}

	evalA, err := mgr.EvaluateSLO(profA)
	if err != nil {
		t.Fatalf("failed to eval SLO: %v", err)
	}
	if evalA.CurrentStatus != StatusHealthy {
		t.Errorf("expected StatusHealthy for Agent A, got %s", evalA.CurrentStatus)
	}
	if evalA.RemainingErrorBudget <= 0 {
		t.Errorf("expected positive error budget for Agent A, got %f", evalA.RemainingErrorBudget)
	}

	// Profile for agent B (breached due to high latency)
	profB := &reliability.ReliabilityProfile{
		TenantID:           tenant,
		AgentID:            agentB,
		CapabilityID:       capID,
		TotalSamples:       50,
		OverallSuccessRate: 0.95, // below default 0.99
		P95LatencyMs:       6000, // above default 5000ms
		AverageCostUSD:     0.05,
	}

	evalB, err := mgr.EvaluateSLO(profB)
	if err != nil {
		t.Fatalf("failed to eval SLO: %v", err)
	}
	if evalB.CurrentStatus != StatusBreached {
		t.Errorf("expected StatusBreached for Agent B, got %s", evalB.CurrentStatus)
	}

	// Compute Capability Health (1 healthy, 1 breached -> DEGRADED)
	ch := mgr.ComputeCapabilityHealth(tenant, capID, []*reliability.ReliabilityProfile{profA, profB})
	if ch.Status != CapDegraded {
		t.Errorf("expected capability status DEGRADED, got %s", ch.Status)
	}
	if ch.HealthyAgents != 1 || ch.BreachedAgents != 1 {
		t.Errorf("expected 1 healthy, 1 breached, got %d healthy, %d breached", ch.HealthyAgents, ch.BreachedAgents)
	}
}
