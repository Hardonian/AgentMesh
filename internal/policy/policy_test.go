package policy_test

import (
	"context"
	"testing"

	"github.com/agentmesh/agentmesh/internal/policy"
)

func TestPolicyEngine(t *testing.T) {
	ctx := context.Background()

	pol := &policy.Policy{
		ID:       "pol_corp_security",
		Version:  "v1.0.0",
		Name:     "Corporate AI Safety Policy",
		TenantID: "tenant_acme",
		Rules: []policy.Rule{
			{
				Name:               "Allow BigQuery Read for Finance",
				Effect:             policy.EffectAllow,
				Agents:             []string{"finance-agent"},
				Tools:              []string{"bigquery.read"},
				Actions:            []string{"read"},
				AllowedDataClasses: []string{policy.DataClassInternal, policy.DataClassConfidential},
			},
			{
				Name:    "Require Approval for BigQuery Delete",
				Effect:  policy.EffectRequireApproval,
				Agents:  []string{"*"},
				Tools:   []string{"bigquery.delete", "payment.*"},
				Actions: []string{"delete", "execute"},
			},
			{
				Name:            "Explicit Deny Restricted Data to Research",
				Effect:          policy.EffectDeny,
				Agents:          []string{"research-agent"},
				DenyDataClasses: []string{policy.DataClassRestricted},
			},
			{
				Name:               "Limit Delegation Depth",
				Effect:             policy.EffectAllow,
				Agents:             []string{"research-agent"},
				Tools:              []string{"web.search"},
				MaxDelegationDepth: 2,
			},
		},
	}

	engine := policy.NewEngine([]*policy.Policy{pol})

	// 1. Finance agent reading BigQuery should be ALLOWED
	dec1 := engine.Evaluate(ctx, &policy.EvaluationRequest{
		TenantID:           "tenant_acme",
		SubjectAgentID:     "finance-agent",
		Tool:               "bigquery.read",
		Action:             "read",
		DataClassification: policy.DataClassInternal,
	})
	if dec1.Effect != policy.EffectAllow {
		t.Errorf("expected ALLOW for finance read, got %s (reason: %s)", dec1.Effect, dec1.Reason)
	}

	// 2. Default deny for unregistered tool
	dec2 := engine.Evaluate(ctx, &policy.EvaluationRequest{
		TenantID:       "tenant_acme",
		SubjectAgentID: "finance-agent",
		Tool:           "gmail.send",
		Action:         "send",
	})
	if dec2.Effect != policy.EffectDeny {
		t.Errorf("expected default DENY for gmail.send, got %s", dec2.Effect)
	}

	// 3. Approval required for delete
	dec3 := engine.Evaluate(ctx, &policy.EvaluationRequest{
		TenantID:       "tenant_acme",
		SubjectAgentID: "finance-agent",
		Tool:           "bigquery.delete",
		Action:         "delete",
	})
	if dec3.Effect != policy.EffectRequireApproval {
		t.Errorf("expected REQUIRE_APPROVAL for bigquery.delete, got %s", dec3.Effect)
	}

	// 4. Restricted data class denied to research-agent
	dec4 := engine.Evaluate(ctx, &policy.EvaluationRequest{
		TenantID:           "tenant_acme",
		SubjectAgentID:     "research-agent",
		Tool:               "web.search",
		Action:             "read",
		DataClassification: policy.DataClassRestricted,
	})
	if dec4.Effect != policy.EffectDeny {
		t.Errorf("expected DENY for restricted data, got %s", dec4.Effect)
	}

	// 5. Delegation depth exceeded
	dec5 := engine.Evaluate(ctx, &policy.EvaluationRequest{
		TenantID:        "tenant_acme",
		SubjectAgentID:  "research-agent",
		Tool:            "web.search",
		DelegationDepth: 3, // exceeds limit of 2
	})
	if dec5.Effect != policy.EffectDeny {
		t.Errorf("expected DENY for delegation depth > 2, got %s", dec5.Effect)
	}

	// 6. Cross-tenant isolation (Tenant B cannot use Tenant A's allow rule)
	dec6 := engine.Evaluate(ctx, &policy.EvaluationRequest{
		TenantID:       "tenant_evil_corp",
		SubjectAgentID: "finance-agent",
		Tool:           "bigquery.read",
		Action:         "read",
	})
	if dec6.Effect != policy.EffectDeny {
		t.Errorf("expected DENY for cross-tenant request, got %s", dec6.Effect)
	}
}
