package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// AgentDeploymentSpec defines the declarative desired state of an agent deployment.
type AgentDeploymentSpec struct {
	AgentID          string            `json:"agentId"`
	OrganizationID   string            `json:"organizationId"`
	ProjectID        string            `json:"projectId"`
	Version          string            `json:"version"`
	Runtime          string            `json:"runtime"` // ADK_GO, GEMINI_NATIVE, PYTHON_MCP, etc.
	Model            string            `json:"model"`
	ModelVersion     string            `json:"modelVersion"`
	GraphVersion     string            `json:"graphVersion"`
	ContractVersion  string            `json:"contractVersion"`
	Tools            []string          `json:"tools"`
	DeploymentTarget string            `json:"deploymentTarget"` // GKE, CLOUD_RUN, VM, EDGE
	Replicas         int               `json:"replicas"`
	Region           string            `json:"region"`
	Environment      string            `json:"environment"` // DEV, STAGING, PRODUCTION
	HealthCheckURL   string            `json:"healthCheckUrl,omitempty"`
	PolicyReferences []string          `json:"policyReferences,omitempty"`
	ManagedBy        string            `json:"managedBy"` // AGENTMESH_CONTROLLER
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

// ComputeDeploymentHash computes an immutable hash of the deployment specification.
func (d *AgentDeploymentSpec) ComputeDeploymentHash() string {
	h := sha256.New()
	content := fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s:%s:%v:%s:%d:%s",
		d.OrganizationID,
		d.AgentID,
		d.Version,
		d.Runtime,
		d.Model,
		d.ModelVersion,
		d.GraphVersion,
		d.ContractVersion,
		d.Tools,
		d.DeploymentTarget,
		d.Replicas,
		d.Environment,
	)
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}
