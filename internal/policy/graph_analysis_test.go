package policy_test

import (
	"context"
	"testing"

	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/pkg/graph"
)

func TestGraphPolicyAnalysis(t *testing.T) {
	pol := &policy.Policy{
		ID:       "pol_strict",
		TenantID: "org_acme",
		Rules: []policy.Rule{
			{
				Name:    "Deny Gmail Send to Finance",
				Effect:  policy.EffectDeny,
				Agents:  []string{"finance-agent"},
				Tools:   []string{"gmail.send"},
				Actions: []string{"execute"},
			},
			{
				Name:    "Require Approval for BigQuery Drop",
				Effect:  policy.EffectRequireApproval,
				Agents:  []string{"*"},
				Tools:   []string{"bigquery.drop_table"},
				Actions: []string{"execute"},
			},
		},
	}

	g := graph.NewAgentGraph("g_finance", "org_acme", "proj", "finance-agent", "1.0.0")
	g.Entrypoint = "n1"
	g.Nodes = []graph.Node{
		{ID: "n1", Name: "Start", Type: graph.NodeTypeAgent},
		{ID: "n2", Name: "Send Mail", Type: graph.NodeTypeTool, Target: "gmail.send"},
		{ID: "n3", Name: "Drop Table", Type: graph.NodeTypeTool, Target: "bigquery.drop_table"},
	}
	g.Tools = []string{"gmail.send", "bigquery.drop_table"}
	g.Edges = []graph.Edge{
		{FromID: "n1", ToID: "n2"},
		{FromID: "n1", ToID: "n3"},
	}

	report := policy.AnalyzeGraphPolicy(g, pol)
	if report.Compliant {
		t.Fatalf("expected graph to be non-compliant due to denied gmail.send tool")
	}

	if len(report.Findings) < 2 {
		t.Fatalf("expected at least 2 findings (forbidden tool & unapproved tool), got %d: %v", len(report.Findings), report.Findings)
	}
}

func TestPolicySimulatorAndShadowCanary(t *testing.T) {
	basePol := &policy.Policy{
		ID:       "pol_base",
		TenantID: "org",
		Rules: []policy.Rule{
			{Name: "Allow Reads", Effect: policy.EffectAllow, Tools: []string{"*.read"}},
		},
	}
	candPol := &policy.Policy{
		ID:       "pol_cand",
		TenantID: "org",
		Rules: []policy.Rule{
			{Name: "Allow Reads", Effect: policy.EffectAllow, Tools: []string{"*.read"}},
			{Name: "Allow Writes", Effect: policy.EffectAllow, Tools: []string{"*.write"}},
		},
	}

	canary := &policy.PolicyCanary{
		ID:              "can_1",
		TenantID:        "org",
		BaselinePolicy:  basePol,
		CandidatePolicy: candPol,
		ShadowMode:      true,
	}

	se := policy.NewShadowEvaluator(canary)

	// Evaluate write tool: baseline DENIES (default deny), candidate ALLOWS
	req := &policy.EvaluationRequest{
		TenantID:       "org",
		SubjectAgentID: "ag_1",
		Tool:           "bigquery.write",
		Action:         "execute",
	}

	baseDec, candDec, err := se.EvaluateShadow(context.Background(), "can_1", req)
	if err != nil {
		t.Fatalf("failed shadow evaluation: %v", err)
	}

	if baseDec.Effect != policy.EffectDeny {
		t.Fatalf("expected baseline to enforce DENY, got %s", baseDec.Effect)
	}
	if candDec.Effect != policy.EffectAllow {
		t.Fatalf("expected candidate shadow to be ALLOW, got %s", candDec.Effect)
	}

	// Promote candidate policy
	promoted, err := se.Promote("can_1")
	if err != nil || promoted.ID != "pol_cand" {
		t.Fatalf("failed to promote candidate policy: %v", err)
	}

	// Now baseline should be pol_cand and allow write
	baseDec2, _, _ := se.EvaluateShadow(context.Background(), "can_1", req)
	if baseDec2.Effect != policy.EffectAllow {
		t.Fatalf("expected promoted policy to allow write, got %s", baseDec2.Effect)
	}

	// Rollback
	rolledBack, err := se.Rollback("can_1")
	if err != nil || rolledBack.ID != "pol_base" {
		t.Fatalf("failed rollback: %v", err)
	}
}

func TestEnterprisePolicyPacks(t *testing.T) {
	pack, err := policy.GetPolicyPack(policy.PackReadOnlyAnalytics, "org_test")
	if err != nil {
		t.Fatalf("failed to get policy pack: %v", err)
	}
	if len(pack.Rules) != 3 {
		t.Fatalf("expected 3 rules in ReadOnlyAnalytics pack, got %d", len(pack.Rules))
	}
}
