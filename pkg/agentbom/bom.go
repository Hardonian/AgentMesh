package agentbom

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/agentmesh/agentmesh/pkg/contracts"
)

const (
	ExpectedAPIVersion = "agentmesh.dev/v1"
	ExpectedKind       = "AgentBOM"
)

// AgentBOM is a machine-readable Software Bill of Materials (SBOM) for an AI agent,
// specifying its declared runtime, models, MCP tools, delegations, and data classifications.
type AgentBOM struct {
	APIVersion   string                 `json:"apiVersion"`
	Kind         string                 `json:"kind"`
	Metadata     BOMMetadata            `json:"metadata"`
	Agent        AgentIdentityInfo      `json:"agent"`
	Models       []ModelDependency      `json:"models,omitempty"`
	Protocols    []string               `json:"protocols"`
	Tools        []ToolDependency       `json:"tools,omitempty"`
	MCPServers   []MCPServerDependency  `json:"mcpServers,omitempty"`
	Delegates    []string               `json:"delegates,omitempty"`
	Permissions  []string               `json:"permissions,omitempty"`
	DataClasses  []string               `json:"dataClasses,omitempty"`
	Policies     []string               `json:"policies,omitempty"`
	Dependencies []ComponentDependency  `json:"dependencies,omitempty"`
}

type BOMMetadata struct {
	AgentName    string    `json:"agentName"`
	Version      string    `json:"version"`
	Organization string    `json:"organization,omitempty"`
	GeneratedAt  time.Time `json:"generatedAt"`
	Author       string    `json:"author,omitempty"`
}

type AgentIdentityInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Runtime   string `json:"runtime"`   // e.g. "go", "python"
	Framework string `json:"framework"` // e.g. "google-adk", "langgraph", "custom"
	Version   string `json:"version"`
}

type ModelDependency struct {
	Provider      string `json:"provider"` // e.g. "google-vertex", "gemini", "anthropic"
	ModelID       string `json:"modelId"`  // e.g. "gemini-1.5-pro"
	Version       string `json:"version,omitempty"`
	ContextWindow int    `json:"contextWindow,omitempty"`
	Purpose       string `json:"purpose,omitempty"`
}

type ToolDependency struct {
	Name               string `json:"name"`
	Provider           string `json:"provider,omitempty"`
	Server             string `json:"server,omitempty"`
	Method             string `json:"method,omitempty"`
	RiskClass          string `json:"riskClass,omitempty"` // "LOW", "MEDIUM", "HIGH", "CRITICAL"
	DataClassification string `json:"dataClassification,omitempty"`
}

type MCPServerDependency struct {
	Name      string `json:"name"`
	Endpoint  string `json:"endpoint"`
	Transport string `json:"transport"` // "stdio", "sse", "http"
	ToolCount int    `json:"toolCount"`
}

type ComponentDependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"` // "library", "sdk", "container"
}

// GenerateFromContract synthesizes an AgentBOM from an AgentContract.
func GenerateFromContract(c *contracts.AgentContract, runtime, framework string) (*AgentBOM, error) {
	if c == nil {
		return nil, errors.New("cannot generate AgentBOM from nil contract")
	}

	bom := &AgentBOM{
		APIVersion: ExpectedAPIVersion,
		Kind:       ExpectedKind,
		Metadata: BOMMetadata{
			AgentName:    c.Metadata.Name,
			Version:      c.Metadata.Version,
			Organization: c.Metadata.Organization,
			GeneratedAt:  time.Now().UTC(),
			Author:       "AgentMesh CLI",
		},
		Agent: AgentIdentityInfo{
			ID:        c.Metadata.Name,
			Name:      c.Metadata.Name,
			Runtime:   runtime,
			Framework: framework,
			Version:   c.Metadata.Version,
		},
		Protocols: c.Identity.Protocols,
		Delegates: c.Delegation.Allow,
	}

	for _, toolName := range c.Tools.Allow {
		classification := c.Tools.DataClassification[toolName]
		if classification == "" {
			classification = "INTERNAL"
		}
		bom.Tools = append(bom.Tools, ToolDependency{
			Name:               toolName,
			DataClassification: classification,
			RiskClass:          "MEDIUM",
		})
	}

	for _, req := range c.Approval.RequiredFor {
		bom.Permissions = append(bom.Permissions, fmt.Sprintf("approval_required:%s", req))
	}

	return bom, nil
}

// Validate checks AgentBOM structural requirements.
func (b *AgentBOM) Validate() error {
	if b.APIVersion != ExpectedAPIVersion {
		return fmt.Errorf("unsupported apiVersion %q, expected %q", b.APIVersion, ExpectedAPIVersion)
	}
	if b.Kind != ExpectedKind {
		return fmt.Errorf("unsupported kind %q, expected %q", b.Kind, ExpectedKind)
	}
	if b.Metadata.AgentName == "" {
		return errors.New("agentName is required in metadata")
	}
	if len(b.Protocols) == 0 {
		return errors.New("at least one protocol must be specified in protocols")
	}
	return nil
}

// ToJSON encodes the AgentBOM to indented JSON.
func (b *AgentBOM) ToJSON() ([]byte, error) {
	return json.MarshalIndent(b, "", "  ")
}

// FromJSON decodes an AgentBOM from JSON bytes.
func FromJSON(data []byte) (*AgentBOM, error) {
	var b AgentBOM
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("failed to parse AgentBOM JSON: %w", err)
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return &b, nil
}
