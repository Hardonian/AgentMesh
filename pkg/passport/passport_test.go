package passport

import (
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/pkg/contracts"
)

func TestAgentPassportV3AndRouterPassport(t *testing.T) {
	c := &contracts.AgentContract{
		Metadata: contracts.Metadata{
			Name:         "finance-agent",
			Version:      "1.0.0",
			Organization: "acme-corp",
		},
		Identity: contracts.IdentityConfig{
			Protocols: []string{"a2a", "mcp"},
		},
		SLO: contracts.SLOConfig{
			P95LatencyMs: 2500,
		},
	}

	p, err := GenerateFromContract(c, "go", "google-adk")
	if err != nil {
		t.Fatalf("failed to generate passport: %v", err)
	}

	p.Reliability.RouteSharePct = 0.45
	p.Reliability.FailoverCount = 2
	p.Reliability.SLOStatus = "HEALTHY"
	p.Evaluations.Freshness = "CURRENT"

	jsonBytes, err := p.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize passport: %v", err)
	}

	parsed, err := FromJSON(jsonBytes)
	if err != nil {
		t.Fatalf("failed to parse passport: %v", err)
	}

	if parsed.Reliability.RouteSharePct != 0.45 {
		t.Errorf("expected RouteSharePct 0.45, got %f", parsed.Reliability.RouteSharePct)
	}
	if parsed.Evaluations.Freshness != "CURRENT" {
		t.Errorf("expected Freshness CURRENT, got %s", parsed.Evaluations.Freshness)
	}

	// Router Passport
	rp := &RouterPassport{
		APIVersion:    "agentmesh.dev/v3alpha1",
		Kind:          "RouterPassport",
		RouterID:      "router-baseline-v1",
		Version:       "1.0.0",
		AlgorithmType: "DETERMINISTIC_BASELINE",
		Objective:     "BALANCED",
		Status:        "ACTIVE",
		AgreementRate: 0.94,
		IssuedAt:      time.Now().UTC(),
		SignedBy:      "AgentMesh Control Plane",
	}

	if rp.RouterID != "router-baseline-v1" || rp.Status != "ACTIVE" {
		t.Errorf("unexpected RouterPassport data: %+v", rp)
	}
}

func TestSanitizeForPublic(t *testing.T) {
	p := &AgentPassport{
		IsPublic: true,
		Identity: PassportIdentity{
			AgentID:      "agent-private",
			Organization: "secret-tenant-alpha",
		},
		Graph: GraphSummary{
			Tools:          []string{"db.write", "secret.decrypt"},
			Delegates:      []string{"admin-agent"},
			ApprovalPoints: []string{"hitl-1"},
		},
		Economics: EconomicMetrics{
			AverageCostUSD: 0.05,
			DailyCostUSD:   1000.0,
		},
	}

	sanitized := p.SanitizeForPublic()
	if sanitized == nil {
		t.Fatal("expected sanitized passport, got nil")
	}
	if sanitized.Identity.Organization != "[REDACTED_ORGANIZATION]" {
		t.Errorf("expected organization to be redacted, got: %s", sanitized.Identity.Organization)
	}
	if sanitized.Economics.AverageCostUSD != 0 || sanitized.Economics.DailyCostUSD != 0 {
		t.Errorf("expected economics to be zeroed, got: %+v", sanitized.Economics)
	}
	if sanitized.Graph.Tools[0] != "2 governed tools" {
		t.Errorf("expected count summary for tools, got: %v", sanitized.Graph.Tools)
	}
}
