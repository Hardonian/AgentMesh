package agentbom

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/agentmesh/agentmesh/pkg/contracts"
)

const (
	ExpectedAPIVersion = "agentmesh.dev/v2alpha1"
	ExpectedKind       = "AgentBOM"
)

// AgentBOM is the enterprise-grade operational bill of materials for autonomous AI agents.
type AgentBOM struct {
	APIVersion         string                `json:"apiVersion"`
	Kind               string                `json:"kind"`
	Metadata           BOMMetadata           `json:"metadata"`
	Agent              AgentIdentityInfo     `json:"agent"`
	ContractVersion    string                `json:"contractVersion"`
	ContractHash       string                `json:"contractHash"`
	GraphHash          string                `json:"graphHash,omitempty"`
	SoftwareSBOMDigest string                `json:"softwareSbomDigest,omitempty"` // Linkage to SPDX/CycloneDX SBOM
	Models             []ModelDependency     `json:"models,omitempty"`
	Protocols          []string              `json:"protocols"`
	Tools              []ToolDependency      `json:"tools,omitempty"`
	MCPServers         []MCPServerDependency `json:"mcpServers,omitempty"`
	Delegates          []string              `json:"delegates,omitempty"`
	Permissions        []string              `json:"permissions,omitempty"`
	DataClasses        []string              `json:"dataClasses,omitempty"`
	A2AProfileRef      string                `json:"a2aProfileRef,omitempty"`
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
	Fingerprint        string `json:"fingerprint,omitempty"` // Stable tool schema fingerprint
	RiskClass          string `json:"riskClass,omitempty"`   // "READ", "WRITE", "DESTRUCTIVE", etc.
	DataClassification string `json:"dataClassification,omitempty"`
}

type MCPServerDependency struct {
	Name      string `json:"name"`
	Endpoint  string `json:"endpoint"`
	Transport string `json:"transport"` // "stdio", "sse", "http"
	ToolCount int    `json:"toolCount"`
}

// GenerateFromContract synthesizes an AgentBOM from an AgentContract.
func GenerateFromContract(c *contracts.AgentContract, runtime, framework string) (*AgentBOM, error) {
	if c == nil {
		return nil, errors.New("cannot generate AgentBOM from nil contract")
	}

	cHash, _ := c.Hash()

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
		ContractVersion: c.Metadata.Version,
		ContractHash:    cHash,
		Protocols:       c.Identity.Protocols,
		Delegates:       c.Delegation.Allow,
	}

	for _, toolName := range c.Tools.Allow {
		classification := c.Tools.DataClassification[toolName]
		if classification == "" {
			classification = "INTERNAL"
		}
		bom.Tools = append(bom.Tools, ToolDependency{
			Name:               toolName,
			DataClassification: classification,
			RiskClass:          "READ",
		})
	}

	for _, req := range c.Approval.RequiredFor {
		bom.Permissions = append(bom.Permissions, fmt.Sprintf("approval_required:%s", req))
	}

	return bom, nil
}

// Hash returns the deterministic SHA-256 digest of canonical AgentBOM representation.
func (b *AgentBOM) Hash() (string, error) {
	clone := *b
	clone.Metadata.GeneratedAt = time.Time{}

	// Sort slices for canonical repeatability
	sort.Strings(clone.Protocols)
	sort.Strings(clone.Delegates)
	sort.Strings(clone.Permissions)
	sort.Strings(clone.DataClasses)
	sort.Slice(clone.Tools, func(i, j int) bool {
		return clone.Tools[i].Name < clone.Tools[j].Name
	})

	bytes, err := json.Marshal(clone)
	if err != nil {
		return "", fmt.Errorf("failed to marshal canonical bom: %w", err)
	}

	h := sha256.Sum256(bytes)
	return hex.EncodeToString(h[:]), nil
}

// Validate checks AgentBOM structural requirements.
func (b *AgentBOM) Validate() error {
	if b.APIVersion == "" {
		return errors.New("apiVersion is required")
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
