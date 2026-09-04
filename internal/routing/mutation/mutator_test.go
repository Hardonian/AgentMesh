package mutation

import (
	"testing"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

func TestRouteMutator_SignedConfigAndRollback(t *testing.T) {
	mut := NewRouteMutator("test-secret-key-12345")

	rs := &spec.AgentRoutingSpec{
		CapabilityID:   "code_review",
		OrganizationID: "org-test",
		ProjectID:      "proj-test",
		Version:        "1.0",
		Weights: map[string]int{
			"agent-senior": 100,
			"agent-junior": 0,
		},
		Fallbacks: []string{"agent-fallback"},
	}

	// 1. Register Route (Sequence 1)
	cfg1, err := mut.RegisterRoute(rs)
	if err != nil {
		t.Fatalf("Failed to register route: %v", err)
	}
	if cfg1.SequenceVersion != 1 {
		t.Fatalf("Expected sequence 1, got %d", cfg1.SequenceVersion)
	}

	// Verify signature
	valid, err := mut.VerifyConfigSignature(cfg1)
	if err != nil || !valid {
		t.Fatalf("Expected valid signature on cfg1: %v", err)
	}

	// 2. Change weights (Sequence 2)
	cfg2, err := mut.ChangeRouteWeights("code_review", map[string]int{
		"agent-senior": 80,
		"agent-junior": 20,
	}, "pol-hash-v1")
	if err != nil {
		t.Fatalf("Failed to change weights: %v", err)
	}
	if cfg2.SequenceVersion != 2 {
		t.Fatalf("Expected sequence 2, got %d", cfg2.SequenceVersion)
	}
	if cfg2.PreviousConfigHash != cfg1.Signature {
		t.Fatalf("Expected chain link: %s != %s", cfg2.PreviousConfigHash, cfg1.Signature)
	}

	// 3. Promote junior (Sequence 3)
	cfg3, err := mut.PromoteCandidate("code_review", "agent-junior", "pol-hash-v2")
	if err != nil {
		t.Fatalf("Failed to promote candidate: %v", err)
	}
	if cfg3.Routes["agent-junior"] != 100 || cfg3.Routes["agent-senior"] != 0 {
		t.Fatalf("Expected agent-junior promoted to 100%%, got %v", cfg3.Routes)
	}

	// 4. Restore prior route (Single action rollback)
	cfgRollback, err := mut.RestorePriorRoute("code_review")
	if err != nil {
		t.Fatalf("Failed to rollback route: %v", err)
	}
	if cfgRollback.Routes["agent-junior"] != 20 || cfgRollback.Routes["agent-senior"] != 80 {
		t.Fatalf("Expected rollback to 80/20, got %v", cfgRollback.Routes)
	}
}
