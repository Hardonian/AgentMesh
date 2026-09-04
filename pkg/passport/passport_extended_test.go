package passport

import (
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/pkg/contracts"
)

func TestPassport_GenerateErrors(t *testing.T) {
	// Nil contract error
	_, err := GenerateFromContract(nil, "go", "google-adk")
	if err == nil {
		t.Fatal("expected error for nil contract")
	}
}

func TestPassport_RecordExecutionSample(t *testing.T) {
	c := &contracts.AgentContract{
		Metadata: contracts.Metadata{
			Name:         "analytics-agent",
			Version:      "2.1.0",
			Organization: "corp-inc",
		},
		Identity: contracts.IdentityConfig{
			Protocols: []string{"a2a"},
		},
		SLO: contracts.SLOConfig{
			P95LatencyMs: 1000,
		},
		Tools: contracts.ToolsConfig{
			Allow: []string{"query.run", "db.fetch"},
		},
		Approval: contracts.ApprovalConfig{
			RequiredFor: []string{"db.fetch"},
		},
		Delegation: contracts.DelegationConfig{
			Allow:    []string{"peer-agent"},
			MaxDepth: 3,
		},
	}

	p, err := GenerateFromContract(c, "python", "langchain")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify Badge when sample count == 0
	badge0 := p.GenerateBadge()
	if badge0 != "[AgentMesh|Passport: analytics-agent | ACTIVE | UNTESTED]" {
		t.Errorf("unexpected badge for 0 samples: %s", badge0)
	}

	// Record execution sample 1 (failure, low latency, low cost)
	p.RecordExecutionSample(false, 500, 0.01, false)
	if p.Reliability.SampleCount != 1 {
		t.Fatalf("expected sample count 1, got %d", p.Reliability.SampleCount)
	}
	if p.Reliability.TaskSuccessRate != 0.0 {
		t.Errorf("expected 0 task success, got %f", p.Reliability.TaskSuccessRate)
	}
	if p.Provenance["reliability"].Tier != EvidenceInferred {
		t.Errorf("expected tier INFERRED, got %s", p.Provenance["reliability"].Tier)
	}

	// Record execution sample 2-6 (successes, higher latency, higher cost to trigger tier change to MEASURED)
	for i := 0; i < 5; i++ {
		p.RecordExecutionSample(true, 1500, 0.05, true)
	}
	if p.Reliability.SampleCount != 6 {
		t.Fatalf("expected sample count 6, got %d", p.Reliability.SampleCount)
	}
	if p.Provenance["reliability"].Tier != EvidenceMeasured {
		t.Errorf("expected tier MEASURED, got %s", p.Provenance["reliability"].Tier)
	}
	if p.Reliability.P95LatencyMs < 1500 {
		t.Errorf("expected p95 latency at least 1500, got %d", p.Reliability.P95LatencyMs)
	}
	if p.Economics.MaxObservedCost < 0.05 {
		t.Errorf("expected max cost 0.05, got %f", p.Economics.MaxObservedCost)
	}

	// Verify Badge when sample count > 0
	badge1 := p.GenerateBadge()
	if badge1 == "" || badge1 == badge0 {
		t.Errorf("expected updated badge, got: %s", badge1)
	}

	// Push sample count past 100 to test confidence ceiling cap (confidence <= 1.0)
	for i := 0; i < 110; i++ {
		p.RecordExecutionSample(true, 800, 0.02, true)
	}
	if p.Provenance["reliability"].Confidence != 1.0 {
		t.Errorf("expected confidence capped at 1.0, got %f", p.Provenance["reliability"].Confidence)
	}
}

func TestPassport_SanitizePrivate(t *testing.T) {
	p := &AgentPassport{
		IsPublic: false,
	}
	if p.SanitizeForPublic() != nil {
		t.Error("expected nil when passport is not public")
	}
}

func TestPassport_InvalidJSON(t *testing.T) {
	_, err := FromJSON([]byte("{invalid-json-data"))
	if err == nil {
		t.Error("expected error parsing invalid JSON")
	}
}

func TestPassport_RouterPassportFields(t *testing.T) {
	rp := RouterPassport{
		APIVersion:       "agentmesh.dev/v3alpha1",
		Kind:             "RouterPassport",
		RouterID:         "router-test",
		Version:          "1.0.0",
		AlgorithmType:    "LEARNED_GBDT",
		Objective:        "COST_OPTIMAL",
		Status:           "CANDIDATE",
		AgreementRate:    0.98,
		CostReductionPct: 24.5,
		IssuedAt:         time.Now().UTC(),
		SignedBy:         "Control Plane CA",
	}

	if rp.RouterID != "router-test" || rp.AlgorithmType != "LEARNED_GBDT" {
		t.Errorf("unexpected router passport: %+v", rp)
	}
}
