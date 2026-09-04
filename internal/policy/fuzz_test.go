package policy

import (
	"context"
	"testing"
)

func FuzzPolicyEvaluate(f *testing.F) {
	f.Add("tenant-1", "agent-a", "query", "bigquery_run", "execute", "dataset/*", 1)
	f.Add("", "", "", "", "", "", 0)
	f.Add("tenant-2", "*", "*", "*", "*", "*", 5)

	pol := &Policy{
		ID:       "pol-1",
		Version:  "v1",
		TenantID: "tenant-1",
		Rules: []Rule{
			{
				Name:               "allow-query",
				Effect:             EffectAllow,
				Agents:             []string{"agent-*"},
				Tools:              []string{"bigquery_*"},
				Actions:            []string{"execute"},
				MaxDelegationDepth: 2,
			},
			{
				Name:    "deny-delete",
				Effect:  EffectDeny,
				Actions: []string{"delete"},
			},
		},
	}
	engine := NewEngine([]*Policy{pol})

	f.Fuzz(func(t *testing.T, tenant, agent, cap, tool, act, res string, depth int) {
		req := &EvaluationRequest{
			TenantID:        tenant,
			SubjectAgentID:  agent,
			Capability:      cap,
			Tool:            tool,
			Action:          act,
			Resource:        res,
			DelegationDepth: depth,
		}
		dec := engine.Evaluate(context.Background(), req)
		if dec == nil {
			t.Fatal("decision should never be nil")
		}
	})
}
