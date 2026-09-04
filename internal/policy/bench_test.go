package policy_test

import (
	"context"
	"testing"

	"github.com/agentmesh/agentmesh/internal/policy"
)

func BenchmarkPolicyEvaluation(b *testing.B) {
	pol := &policy.Policy{
		ID:       "pol_bench",
		Version:  "v1",
		TenantID: "corp",
		Rules: []policy.Rule{
			{Name: "Rule 1", Effect: policy.EffectAllow, Agents: []string{"finance-agent"}, Tools: []string{"bigquery.read"}},
			{Name: "Rule 2", Effect: policy.EffectRequireApproval, Agents: []string{"*"}, Tools: []string{"bigquery.delete"}},
			{Name: "Rule 3", Effect: policy.EffectDeny, Agents: []string{"research-agent"}, Tools: []string{"payment.*"}},
		},
	}
	engine := policy.NewEngine([]*policy.Policy{pol})
	req := &policy.EvaluationRequest{
		TenantID:       "corp",
		SubjectAgentID: "finance-agent",
		Tool:           "bigquery.read",
		Action:         "read",
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.Evaluate(ctx, req)
	}
}
