package v1alpha1_test

import (
	"encoding/json"
	"testing"

	"github.com/agentmesh/agentmesh/operator/api/v1alpha1"
	"github.com/agentmesh/agentmesh/pkg/contracts"
)

func TestAgentMeshAgent_JSONRoundtrip(t *testing.T) {
	agent := v1alpha1.AgentMeshAgent{
		APIVersion: "agentmesh.dev/v1alpha1",
		Kind:       "AgentMeshAgent",
		Metadata: map[string]any{
			"name":      "procurement-agent",
			"namespace": "production-agents",
		},
		Spec: v1alpha1.AgentMeshAgentSpec{
			Name: "procurement-agent",
			Contract: contracts.AgentContract{
				Metadata: contracts.Metadata{Name: "procurement-agent"},
			},
			EndpointURL: "http://procurement.default.svc:8080",
			Replicas:    2,
		},
		Status: v1alpha1.AgentMeshAgentStatus{
			Phase:          "Active",
			RegisteredInCP: true,
			ContractHash:   "hash-123",
		},
	}

	bytes, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("failed to marshal AgentMeshAgent: %v", err)
	}

	var decoded v1alpha1.AgentMeshAgent
	if err := json.Unmarshal(bytes, &decoded); err != nil {
		t.Fatalf("failed to unmarshal AgentMeshAgent: %v", err)
	}

	if decoded.Spec.Name != "procurement-agent" || decoded.Status.Phase != "Active" {
		t.Errorf("unexpected decoded AgentMeshAgent: %+v", decoded)
	}
}

func TestAgentMeshPolicy_JSONRoundtrip(t *testing.T) {
	policy := v1alpha1.AgentMeshPolicy{
		APIVersion: "agentmesh.dev/v1alpha1",
		Kind:       "AgentMeshPolicy",
		Metadata: map[string]any{
			"name": "default-deny-policy",
		},
		Spec: v1alpha1.AgentMeshPolicySpec{
			TenantID: "tenant-corp",
			Rules: []v1alpha1.PolicyRuleSpec{
				{
					Name:    "rule-allow-read",
					Effect:  "ALLOW",
					Agents:  []string{"finance-agent"},
					Tools:   []string{"bigquery.read"},
					Actions: []string{"query"},
				},
			},
		},
	}

	bytes, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("failed to marshal AgentMeshPolicy: %v", err)
	}

	var decoded v1alpha1.AgentMeshPolicy
	if err := json.Unmarshal(bytes, &decoded); err != nil {
		t.Fatalf("failed to unmarshal AgentMeshPolicy: %v", err)
	}

	if decoded.Spec.TenantID != "tenant-corp" || len(decoded.Spec.Rules) != 1 {
		t.Errorf("unexpected decoded AgentMeshPolicy: %+v", decoded)
	}
}
