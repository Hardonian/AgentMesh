package routing_test

import (
	"context"
	"testing"

	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/pkg/contracts"
)

func TestGeoRouting_RegionalLatencyPenalty(t *testing.T) {
	penaltyLocal := routing.EstimateRegionalLatencyPenalty("us-central1", "us-central1")
	if penaltyLocal != 0 {
		t.Fatalf("expected 0ms local penalty, got: %d", penaltyLocal)
	}

	penaltyCrossUS := routing.EstimateRegionalLatencyPenalty("us-central1", "us-east4")
	if penaltyCrossUS <= 0 {
		t.Fatalf("expected positive cross-region penalty, got: %d", penaltyCrossUS)
	}

	penaltyTransatlantic := routing.EstimateRegionalLatencyPenalty("us-central1", "europe-west1")
	if penaltyTransatlantic <= penaltyCrossUS {
		t.Fatalf("expected transatlantic latency to be greater than cross-US: %d <= %d", penaltyTransatlantic, penaltyCrossUS)
	}
}

func TestGeoRouting_DataResidencyEnforcement(t *testing.T) {
	allowPol := &policy.Policy{
		ID:       "pol-allow-route",
		TenantID: "tenant-geo",
		Rules: []policy.Rule{
			{Name: "Allow route", Effect: policy.EffectAllow, Agents: []string{"*"}, Actions: []string{"route"}},
		},
	}
	router := routing.NewRouter(policy.NewEngine([]*policy.Policy{allowPol}))

	// US Candidate
	router.RegisterCandidate(&routing.AgentRouteCandidate{
		AgentID:     "agent-us",
		Region:      "us-central1",
		Status:      "HEALTHY",
		SuccessRate: 0.99,
		Contract: &contracts.AgentContract{
			Metadata:     contracts.Metadata{Name: "agent-us", Version: "1.0.0"},
			Capabilities: []string{"data_processing"},
		},
	})

	// EU Candidate
	router.RegisterCandidate(&routing.AgentRouteCandidate{
		AgentID:     "agent-eu",
		Region:      "europe-west1",
		Status:      "HEALTHY",
		SuccessRate: 0.99,
		Contract: &contracts.AgentContract{
			Metadata:     contracts.Metadata{Name: "agent-eu", Version: "1.0.0"},
			Capabilities: []string{"data_processing"},
		},
	})

	// Request restricted to EU only (GDPR residency)
	req := &routing.RouteRequestV2{
		TenantID:           "tenant-geo",
		CallerAgentID:      "portal",
		RequiredCapability: "data_processing",
		AllowedRegions:     []string{"europe-west1"},
		Strategy:           routing.StrategyBalanced,
	}

	dec, err := router.RouteV2(context.Background(), req)
	if err != nil {
		t.Fatalf("RouteV2 failed: %v", err)
	}

	if dec.SelectedAgentID != "agent-eu" {
		t.Fatalf("data residency violated: expected agent-eu, got %s", dec.SelectedAgentID)
	}

	// Verify agent-us was disqualified with residency reason
	for _, cand := range dec.Candidates {
		if cand.AgentID == "agent-us" && cand.Eligible {
			t.Fatal("agent-us should have been disqualified by data residency filter")
		}
	}
}
