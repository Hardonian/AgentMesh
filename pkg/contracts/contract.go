package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	ExpectedAPIVersion = "agentmesh.dev/v1"
	ExpectedKind       = "AgentContract"
)

// AgentContract is the canonical specification governing an agent's identity,
// capabilities, tool permissions, delegation rules, budgets, and SLOs.
type AgentContract struct {
	APIVersion   string           `json:"apiVersion" yaml:"apiVersion"`
	Kind         string           `json:"kind" yaml:"kind"`
	Metadata     Metadata         `json:"metadata" yaml:"metadata"`
	Identity     IdentityConfig   `json:"identity" yaml:"identity"`
	Capabilities []string         `json:"capabilities" yaml:"capabilities"`
	Tools        ToolsConfig      `json:"tools" yaml:"tools"`
	Delegation   DelegationConfig `json:"delegation" yaml:"delegation"`
	Budgets      BudgetConfig     `json:"budgets" yaml:"budgets"`
	SLO          SLOConfig        `json:"slo,omitempty" yaml:"slo,omitempty"`
	Approval     ApprovalConfig   `json:"approval,omitempty" yaml:"approval,omitempty"`
}

type Metadata struct {
	Name         string            `json:"name" yaml:"name"`
	Organization string            `json:"organization,omitempty" yaml:"organization,omitempty"`
	Version      string            `json:"version,omitempty" yaml:"version,omitempty"`
	Description  string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels       map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type IdentityConfig struct {
	Protocols      []string `json:"protocols" yaml:"protocols"` // e.g., ["a2a", "mcp"]
	ServiceAccount string   `json:"serviceAccount,omitempty" yaml:"serviceAccount,omitempty"`
}

type ToolsConfig struct {
	Allow              []string          `json:"allow,omitempty" yaml:"allow,omitempty"`
	Deny               []string          `json:"deny,omitempty" yaml:"deny,omitempty"`
	DataClassification map[string]string `json:"dataClassification,omitempty" yaml:"dataClassification,omitempty"`
}

type DelegationConfig struct {
	Allow    []string `json:"allow,omitempty" yaml:"allow,omitempty"`
	Deny     []string `json:"deny,omitempty" yaml:"deny,omitempty"`
	MaxDepth int      `json:"maxDepth,omitempty" yaml:"maxDepth,omitempty"`
}

type BudgetConfig struct {
	MaxCostPerTask      float64 `json:"maxCostPerTask,omitempty" yaml:"max_cost_per_task,omitempty"`
	MaxTokensPerTask    int64   `json:"maxTokensPerTask,omitempty" yaml:"max_tokens_per_task,omitempty"`
	MaxToolCallsPerTask int     `json:"maxToolCallsPerTask,omitempty" yaml:"max_tool_calls_per_task,omitempty"`
	MaxDailyCost        float64 `json:"maxDailyCost,omitempty" yaml:"max_daily_cost,omitempty"`
}

type SLOConfig struct {
	P95LatencyMs int64   `json:"p95LatencyMs,omitempty" yaml:"p95_latency_ms,omitempty"`
	SuccessRate  float64 `json:"successRate,omitempty" yaml:"success_rate,omitempty"`
}

type ApprovalConfig struct {
	RequiredFor    []string `json:"requiredFor,omitempty" yaml:"required_for,omitempty"`
	TimeoutSeconds int      `json:"timeoutSeconds,omitempty" yaml:"timeout_seconds,omitempty"`
}

// ParseYAML decodes and validates an AgentContract from YAML format.
func ParseYAML(data []byte) (*AgentContract, error) {
	var c AgentContract
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML contract: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// ParseJSON decodes and validates an AgentContract from JSON format.
func ParseJSON(data []byte) (*AgentContract, error) {
	var c AgentContract
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON contract: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate checks all schema constraints and semantic invariants.
func (c *AgentContract) Validate() error {
	if c.APIVersion == "" {
		return errors.New("missing apiVersion")
	}
	if c.APIVersion != ExpectedAPIVersion {
		return fmt.Errorf("unsupported apiVersion %q, expected %q", c.APIVersion, ExpectedAPIVersion)
	}
	if c.Kind != ExpectedKind {
		return fmt.Errorf("unsupported kind %q, expected %q", c.Kind, ExpectedKind)
	}
	if strings.TrimSpace(c.Metadata.Name) == "" {
		return errors.New("contract metadata.name is required")
	}
	if len(c.Identity.Protocols) == 0 {
		return errors.New("identity.protocols must specify at least one protocol (e.g., 'a2a', 'mcp')")
	}
	for _, p := range c.Identity.Protocols {
		pLower := strings.ToLower(p)
		if pLower != "a2a" && pLower != "mcp" && pLower != "http" && pLower != "grpc" {
			return fmt.Errorf("invalid protocol %q", p)
		}
	}
	if c.Delegation.MaxDepth < 0 {
		return errors.New("delegation.maxDepth cannot be negative")
	}
	if c.Budgets.MaxCostPerTask < 0 || c.Budgets.MaxTokensPerTask < 0 {
		return errors.New("budgets cannot be negative")
	}
	if c.SLO.SuccessRate < 0 || c.SLO.SuccessRate > 1.0 {
		if c.SLO.SuccessRate > 1.0 && c.SLO.SuccessRate <= 100.0 {
			// Auto normalize percentage if specified 0-100
			c.SLO.SuccessRate = c.SLO.SuccessRate / 100.0
		} else if c.SLO.SuccessRate != 0 {
			return errors.New("slo.successRate must be between 0.0 and 1.0")
		}
	}
	return nil
}

// CanonicalJSON produces a deterministic, canonically ordered JSON encoding.
func (c *AgentContract) CanonicalJSON() ([]byte, error) {
	clone := *c
	// Sort slices for determinism
	clone.Capabilities = append([]string(nil), clone.Capabilities...)
	sort.Strings(clone.Capabilities)

	clone.Tools.Allow = append([]string(nil), clone.Tools.Allow...)
	sort.Strings(clone.Tools.Allow)

	clone.Tools.Deny = append([]string(nil), clone.Tools.Deny...)
	sort.Strings(clone.Tools.Deny)

	clone.Delegation.Allow = append([]string(nil), clone.Delegation.Allow...)
	sort.Strings(clone.Delegation.Allow)

	clone.Delegation.Deny = append([]string(nil), clone.Delegation.Deny...)
	sort.Strings(clone.Delegation.Deny)

	clone.Approval.RequiredFor = append([]string(nil), clone.Approval.RequiredFor...)
	sort.Strings(clone.Approval.RequiredFor)

	clone.Identity.Protocols = append([]string(nil), clone.Identity.Protocols...)
	sort.Strings(clone.Identity.Protocols)

	return json.Marshal(clone)
}

// Hash returns the hex-encoded SHA-256 digest of the canonical contract representation.
func (c *AgentContract) Hash() (string, error) {
	canonicalBytes, err := c.CanonicalJSON()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(canonicalBytes)
	return hex.EncodeToString(h[:]), nil
}

// ContractDiff represents differences between two contracts.
type ContractDiff struct {
	AddedCapabilities   []string `json:"addedCapabilities,omitempty"`
	RemovedCapabilities []string `json:"removedCapabilities,omitempty"`
	AddedAllowedTools   []string `json:"addedAllowedTools,omitempty"`
	RemovedAllowedTools []string `json:"removedAllowedTools,omitempty"`
	AddedDeniedTools    []string `json:"addedDeniedTools,omitempty"`
	RemovedDeniedTools  []string `json:"removedDeniedTools,omitempty"`
	BudgetChanged       bool     `json:"budgetChanged"`
	SLOChanged          bool     `json:"sloChanged"`
}

// Diff compares this contract against an older contract.
func (c *AgentContract) Diff(old *AgentContract) *ContractDiff {
	diff := &ContractDiff{}
	if old == nil {
		diff.AddedCapabilities = c.Capabilities
		diff.AddedAllowedTools = c.Tools.Allow
		diff.AddedDeniedTools = c.Tools.Deny
		diff.BudgetChanged = true
		diff.SLOChanged = true
		return diff
	}

	diff.AddedCapabilities, diff.RemovedCapabilities = sliceDiff(old.Capabilities, c.Capabilities)
	diff.AddedAllowedTools, diff.RemovedAllowedTools = sliceDiff(old.Tools.Allow, c.Tools.Allow)
	diff.AddedDeniedTools, diff.RemovedDeniedTools = sliceDiff(old.Tools.Deny, c.Tools.Deny)

	if c.Budgets != old.Budgets {
		diff.BudgetChanged = true
	}
	if c.SLO != old.SLO {
		diff.SLOChanged = true
	}
	return diff
}

func sliceDiff(oldS, newS []string) (added, removed []string) {
	oldMap := make(map[string]bool)
	for _, s := range oldS {
		oldMap[s] = true
	}
	newMap := make(map[string]bool)
	for _, s := range newS {
		newMap[s] = true
		if !oldMap[s] {
			added = append(added, s)
		}
	}
	for _, s := range oldS {
		if !newMap[s] {
			removed = append(removed, s)
		}
	}
	return added, removed
}
