package spec

import (
	"testing"
	"time"
)

func TestOptimizationActionHash(t *testing.T) {
	action1 := &AgentOptimizationAction{
		OrganizationID: "org-1",
		ProjectID:      "proj-1",
		CapabilityID:   "cap-1",
		TargetType:     "ROUTE",
		TargetID:       "route-1",
		ActionType:     ActionChangeRouteWeight,
		CurrentState:   map[string]any{"weight": 100},
		ProposedState:  map[string]any{"weight": 80},
		PolicyVersion:  "v1",
		RiskClass:      RiskLow,
	}

	action2 := &AgentOptimizationAction{
		OrganizationID: "org-1",
		ProjectID:      "proj-1",
		CapabilityID:   "cap-1",
		TargetType:     "ROUTE",
		TargetID:       "route-1",
		ActionType:     ActionChangeRouteWeight,
		CurrentState:   map[string]any{"weight": 100},
		ProposedState:  map[string]any{"weight": 80},
		PolicyVersion:  "v1",
		RiskClass:      RiskLow,
	}

	hash1 := action1.ComputeActionHash()
	hash2 := action2.ComputeActionHash()
	if hash1 != hash2 {
		t.Fatalf("Expected hashes to match: %s != %s", hash1, hash2)
	}

	// Changing parameter must alter hash
	action2.ProposedState = map[string]any{"weight": 70}
	if action1.ComputeActionHash() == action2.ComputeActionHash() {
		t.Fatalf("Expected hash to change when proposed state changes")
	}
}

func TestRoutingAndDeploymentSpecHashes(t *testing.T) {
	rs := &AgentRoutingSpec{
		CapabilityID:   "deep_research",
		OrganizationID: "org-test",
		Version:        "1.0",
		EligibleAgents: []string{"a1", "a2"},
		Weights:        map[string]int{"a1": 100},
	}
	h1 := rs.ComputeSpecHash()
	if h1 == "" {
		t.Fatalf("Expected non-empty routing spec hash")
	}

	ds := &AgentDeploymentSpec{
		AgentID:        "agent-1",
		OrganizationID: "org-test",
		Version:        "v1",
		Runtime:        "ADK_GO",
		Model:          "gemini-1.5-pro",
		CreatedAt:      time.Now().UTC(),
	}
	h2 := ds.ComputeDeploymentHash()
	if h2 == "" {
		t.Fatalf("Expected non-empty deployment spec hash")
	}
}
