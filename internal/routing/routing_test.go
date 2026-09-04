package routing_test

import (
	"context"
	"testing"

	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/pkg/contracts"
)

func TestCapabilityRouting(t *testing.T) {
	ctx := context.Background()

	polEngine := policy.NewEngine([]*policy.Policy{
		{
			ID:       "pol_default",
			Version:  "v1",
			TenantID: "acme",
			Rules: []policy.Rule{
				{
					Name:    "Allow all agent invocation",
					Effect:  policy.EffectAllow,
					Agents:  []string{"*"},
					Actions: []string{"invoke"},
				},
			},
		},
	})

	router := routing.NewRouter(polEngine)

	// Candidate A: Fast, but more expensive
	candA := &routing.AgentRouteCandidate{
		AgentID:     "agent-fast",
		EndpointURL: "http://agent-fast.mesh:8080",
		Status:      "HEALTHY",
		Contract: &contracts.AgentContract{
			Capabilities: []string{"financial_analysis"},
		},
		AverageCost:  0.08,
		P95LatencyMs: 400,
		SuccessRate:  0.98,
	}

	// Candidate B: Cheap, but slower
	candB := &routing.AgentRouteCandidate{
		AgentID:     "agent-cheap",
		EndpointURL: "http://agent-cheap.mesh:8080",
		Status:      "HEALTHY",
		Contract: &contracts.AgentContract{
			Capabilities: []string{"financial_analysis"},
		},
		AverageCost:  0.01,
		P95LatencyMs: 3000,
		SuccessRate:  0.95,
	}

	// Candidate C: High reliability
	candC := &routing.AgentRouteCandidate{
		AgentID:     "agent-reliable",
		EndpointURL: "http://agent-reliable.mesh:8080",
		Status:      "HEALTHY",
		Contract: &contracts.AgentContract{
			Capabilities: []string{"financial_analysis"},
		},
		AverageCost:  0.05,
		P95LatencyMs: 1200,
		SuccessRate:  0.999,
	}

	// Candidate D: Disabled
	candD := &routing.AgentRouteCandidate{
		AgentID:     "agent-disabled",
		EndpointURL: "http://agent-disabled.mesh:8080",
		Status:      "DISABLED",
		Contract: &contracts.AgentContract{
			Capabilities: []string{"financial_analysis"},
		},
		AverageCost:  0.001,
		P95LatencyMs: 100,
		SuccessRate:  1.0,
	}

	router.RegisterCandidate(candA)
	router.RegisterCandidate(candB)
	router.RegisterCandidate(candC)
	router.RegisterCandidate(candD)

	// Test 1: Lowest cost strategy should pick agent-cheap
	dec1, err := router.Route(ctx, &routing.RouteRequest{
		TenantID:           "acme",
		CallerAgentID:      "caller",
		RequiredCapability: "financial_analysis",
		Strategy:           routing.StrategyLowestCost,
	})
	if err != nil {
		t.Fatalf("unexpected route error: %v", err)
	}
	if dec1.SelectedAgentID != "agent-cheap" {
		t.Errorf("expected agent-cheap for LOWEST_COST, got %s", dec1.SelectedAgentID)
	}

	// Test 2: Lowest latency strategy should pick agent-fast
	dec2, err := router.Route(ctx, &routing.RouteRequest{
		TenantID:           "acme",
		CallerAgentID:      "caller",
		RequiredCapability: "financial_analysis",
		Strategy:           routing.StrategyLowestLatency,
	})
	if err != nil {
		t.Fatalf("unexpected route error: %v", err)
	}
	if dec2.SelectedAgentID != "agent-fast" {
		t.Errorf("expected agent-fast for LOWEST_LATENCY, got %s", dec2.SelectedAgentID)
	}

	// Test 3: Highest reliability strategy should pick agent-reliable
	dec3, err := router.Route(ctx, &routing.RouteRequest{
		TenantID:           "acme",
		CallerAgentID:      "caller",
		RequiredCapability: "financial_analysis",
		Strategy:           routing.StrategyHighestReliability,
	})
	if err != nil {
		t.Fatalf("unexpected route error: %v", err)
	}
	if dec3.SelectedAgentID != "agent-reliable" {
		t.Errorf("expected agent-reliable for HIGHEST_RELIABILITY, got %s", dec3.SelectedAgentID)
	}

	// Verify disabled agent was never selected and has exclusion reason
	for _, expl := range dec3.Explanations {
		if expl.AgentID == "agent-disabled" {
			if expl.Eligible {
				t.Error("agent-disabled should not be marked eligible")
			}
		}
	}
}
