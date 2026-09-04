package a2a_test

import (
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/a2a"
)

func TestPublicCompatibilityRegistry_PublishAndQuery(t *testing.T) {
	reg := a2a.NewPublicCompatibilityRegistry()

	profile := &a2a.A2ACompatibilityProfile{
		AgentID:         "internal-private-agent", // Must be stripped/anonymized!
		Version:         "1.0.0",
		ProtocolVersion: "v0.3.0",
		Status:          a2a.StatusCompatible,
		TesterVersion:   "agentmesh-lab-v2.0",
		TestedAt:        time.Now().UTC(),
		Results: map[string]a2a.TestCaseResult{
			"discovery": {Name: "discovery", Passed: true},
			"streaming": {Name: "streaming", Passed: true},
			"artifacts": {Name: "artifacts", Passed: false},
		},
	}

	entry, err := reg.PublishProfile("go", "google-adk", profile)
	if err != nil {
		t.Fatalf("PublishProfile failed: %v", err)
	}

	if entry.AnonymousID == "" {
		t.Fatal("expected non-empty anonymous ID")
	}
	if entry.AnonymousID == "internal-private-agent" {
		t.Fatal("private agent ID leaked into anonymous ID")
	}

	// Verify interoperability matrix computation
	matrix := reg.GetMatrix("v0.3.0")
	if matrix.TotalEntries != 1 {
		t.Fatalf("expected 1 entry in matrix, got: %d", matrix.TotalEntries)
	}

	runtimeKey := "go/google-adk"
	if status, ok := matrix.Matrix[runtimeKey]["discovery"]; !ok || status != string(a2a.StatusCompatible) {
		t.Fatalf("expected discovery COMPATIBLE, got: %s", status)
	}
	if status, ok := matrix.Matrix[runtimeKey]["artifacts"]; !ok || status != string(a2a.StatusIncompatible) {
		t.Fatalf("expected artifacts INCOMPATIBLE, got: %s", status)
	}
}
