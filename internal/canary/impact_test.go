package canary_test

import (
	"testing"

	"github.com/agentmesh/agentmesh/internal/canary"
	"github.com/agentmesh/agentmesh/pkg/contracts"
)

func TestAnalyzeChangeImpact(t *testing.T) {
	current := &contracts.AgentContract{
		Metadata: contracts.Metadata{Name: "finance-agent", Version: "1.0.0"},
		Tools: contracts.ToolsConfig{
			Allow: []string{"bigquery.read"},
		},
		Delegation: contracts.DelegationConfig{
			MaxDepth: 2,
		},
	}

	// Candidate with benign update
	candBenign := &contracts.AgentContract{
		Metadata: contracts.Metadata{Name: "finance-agent", Version: "1.0.1"},
		Tools: contracts.ToolsConfig{
			Allow: []string{"bigquery.read"},
		},
		Delegation: contracts.DelegationConfig{
			MaxDepth: 2,
		},
	}

	rep1, err := canary.AnalyzeChangeImpact(current, candBenign)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rep1.SafeToCanary {
		t.Fatalf("expected benign update to be safe to canary")
	}

	// Candidate with critical destructive tool and increased depth
	candRisky := &contracts.AgentContract{
		Metadata: contracts.Metadata{Name: "finance-agent", Version: "1.1.0"},
		Tools: contracts.ToolsConfig{
			Allow: []string{"bigquery.read", "bigquery.drop_table", "payment.execute"},
			DataClassification: map[string]string{
				"payment.execute": "RESTRICTED",
			},
		},
		Delegation: contracts.DelegationConfig{
			Allow:    []string{"external-agent"},
			MaxDepth: 5,
		},
	}

	rep2, err := canary.AnalyzeChangeImpact(current, candRisky)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep2.SafeToCanary {
		t.Fatalf("expected risky candidate to be marked unsafe to canary")
	}
	if !rep2.RequiresExplicitPolicyReview {
		t.Fatalf("expected RequiresExplicitPolicyReview to be true")
	}
	if len(rep2.SecuritySensitiveFlags) < 3 {
		t.Fatalf("expected at least 3 security sensitive flags, got %d: %v", len(rep2.SecuritySensitiveFlags), rep2.SecuritySensitiveFlags)
	}
}
