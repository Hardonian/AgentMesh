package v1alpha1

import (
	"github.com/agentmesh/agentmesh/pkg/contracts"
)

// AgentMeshAgentSpec defines the desired state of an Agent in Kubernetes.
type AgentMeshAgentSpec struct {
	Name        string                  `json:"name"`
	Contract    contracts.AgentContract `json:"contract"`
	EndpointURL string                  `json:"endpointUrl"`
	Replicas    int32                   `json:"replicas,omitempty"`
}

// AgentMeshAgentStatus defines the observed operational state.
type AgentMeshAgentStatus struct {
	Phase           string `json:"phase"` // "Active", "Syncing", "Error"
	ContractHash    string `json:"contractHash"`
	RegisteredInCP  bool   `json:"registeredInControlPlane"`
	LastHeartbeatAt string `json:"lastHeartbeatAt"`
}

// AgentMeshAgent is the Schema for the agentmeshagents API.
type AgentMeshAgent struct {
	APIVersion string               `json:"apiVersion"`
	Kind       string               `json:"kind"`
	Metadata   map[string]any       `json:"metadata"`
	Spec       AgentMeshAgentSpec   `json:"spec"`
	Status     AgentMeshAgentStatus `json:"status,omitempty"`
}

// AgentMeshPolicySpec defines a policy resource in Kubernetes.
type AgentMeshPolicySpec struct {
	TenantID string           `json:"tenantId"`
	Rules    []PolicyRuleSpec `json:"rules"`
}

type PolicyRuleSpec struct {
	Name    string   `json:"name"`
	Effect  string   `json:"effect"` // ALLOW, DENY, REQUIRE_APPROVAL
	Agents  []string `json:"agents,omitempty"`
	Tools   []string `json:"tools,omitempty"`
	Actions []string `json:"actions,omitempty"`
}

// AgentMeshPolicy is the Schema for Kubernetes-native policy enforcement.
type AgentMeshPolicy struct {
	APIVersion string              `json:"apiVersion"`
	Kind       string              `json:"kind"`
	Metadata   map[string]any      `json:"metadata"`
	Spec       AgentMeshPolicySpec `json:"spec"`
}
