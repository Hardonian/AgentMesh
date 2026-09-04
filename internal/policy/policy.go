package policy

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"
)

type Effect string

const (
	EffectAllow           Effect = "ALLOW"
	EffectDeny            Effect = "DENY"
	EffectRequireApproval Effect = "REQUIRE_APPROVAL"
)

// Data Classification Tags
const (
	DataClassPublic       = "PUBLIC"
	DataClassInternal     = "INTERNAL"
	DataClassConfidential = "CONFIDENTIAL"
	DataClassRestricted   = "RESTRICTED"
)

// Policy contains a set of declarative authorization rules.
type Policy struct {
	ID          string    `json:"id" yaml:"id"`
	Version     string    `json:"version" yaml:"version"`
	Name        string    `json:"name" yaml:"name"`
	TenantID    string    `json:"tenantId" yaml:"tenantId"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Rules       []Rule    `json:"rules" yaml:"rules"`
	CreatedAt   time.Time `json:"createdAt" yaml:"createdAt"`
}

// Rule defines a match condition and an authorization effect.
type Rule struct {
	Name               string   `json:"name" yaml:"name"`
	Effect             Effect   `json:"effect" yaml:"effect"` // ALLOW, DENY, REQUIRE_APPROVAL
	Agents             []string `json:"agents,omitempty" yaml:"agents,omitempty"`
	Capabilities       []string `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	Tools              []string `json:"tools,omitempty" yaml:"tools,omitempty"`
	Actions            []string `json:"actions,omitempty" yaml:"actions,omitempty"`
	Resources          []string `json:"resources,omitempty" yaml:"resources,omitempty"`
	MaxDelegationDepth int      `json:"maxDelegationDepth,omitempty" yaml:"maxDelegationDepth,omitempty"`
	AllowedDataClasses []string `json:"allowedDataClasses,omitempty" yaml:"allowedDataClasses,omitempty"`
	DenyDataClasses    []string `json:"denyDataClasses,omitempty" yaml:"denyDataClasses,omitempty"`
}

// EvaluationRequest provides the contextual attributes of an attempted invocation.
type EvaluationRequest struct {
	TenantID           string   `json:"tenantId"`
	SubjectAgentID     string   `json:"subjectAgentId"`
	Capability         string   `json:"capability,omitempty"`
	Tool               string   `json:"tool,omitempty"`
	Action             string   `json:"action,omitempty"`
	Resource           string   `json:"resource,omitempty"`
	DataClassification string   `json:"dataClassification,omitempty"`
	DelegationDepth    int      `json:"delegationDepth"`
	DelegationStack    []string `json:"delegationStack,omitempty"`
	ApprovedToken      string   `json:"approvedToken,omitempty"` // For bypass when approved
}

// Decision is the deterministic authorization outcome.
type Decision struct {
	Effect          Effect    `json:"effect"`
	PolicyID        string    `json:"policyId"`
	RuleName        string    `json:"ruleName"`
	Reason          string    `json:"reason"`
	Timestamp       time.Time `json:"timestamp"`
	DecisionVersion string    `json:"decisionVersion"`
}

// Engine evaluates requests deterministically against configured policies.
type Engine struct {
	mu       sync.RWMutex
	policies []*Policy
}

// NewEngine creates a new policy evaluation engine.
func NewEngine(policies []*Policy) *Engine {
	return &Engine{
		policies: policies,
	}
}

// SetPolicies safely swaps the active policies.
func (e *Engine) SetPolicies(policies []*Policy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policies = policies
}

// Evaluate evaluates an authorization request.
// Invariant: Privileged access is default deny.
// Explicit DENY rules always win over ALLOW.
// REQUIRE_APPROVAL triggers if matched and not denied.
func (e *Engine) Evaluate(ctx context.Context, req *EvaluationRequest) *Decision {
	e.mu.RLock()
	defer e.mu.RUnlock()

	now := time.Now().UTC()
	var matchedApproval *Decision
	var matchedAllow *Decision

	for _, pol := range e.policies {
		if pol.TenantID != "" && req.TenantID != "" && pol.TenantID != req.TenantID {
			continue // Tenant isolation
		}

		for _, rule := range pol.Rules {
			if !matchesAgent(rule.Agents, req.SubjectAgentID) {
				continue
			}
			if !matchesString(rule.Capabilities, req.Capability) {
				continue
			}
			if !matchesPattern(rule.Tools, req.Tool) {
				continue
			}
			if !matchesString(rule.Actions, req.Action) {
				continue
			}
			if !matchesPattern(rule.Resources, req.Resource) {
				continue
			}

			// Check delegation depth constraints
			if rule.MaxDelegationDepth > 0 && req.DelegationDepth > rule.MaxDelegationDepth {
				return &Decision{
					Effect:          EffectDeny,
					PolicyID:        pol.ID,
					RuleName:        rule.Name,
					Reason:          fmt.Sprintf("delegation depth %d exceeds maximum allowed %d", req.DelegationDepth, rule.MaxDelegationDepth),
					Timestamp:       now,
					DecisionVersion: pol.Version,
				}
			}

			// Check data classification deny
			if req.DataClassification != "" && len(rule.DenyDataClasses) > 0 {
				for _, dc := range rule.DenyDataClasses {
					if strings.EqualFold(dc, req.DataClassification) {
						return &Decision{
							Effect:          EffectDeny,
							PolicyID:        pol.ID,
							RuleName:        rule.Name,
							Reason:          fmt.Sprintf("data classification %q is explicitly denied", req.DataClassification),
							Timestamp:       now,
							DecisionVersion: pol.Version,
						}
					}
				}
			}

			// Check data classification allowlist if specified
			if req.DataClassification != "" && len(rule.AllowedDataClasses) > 0 {
				allowed := false
				for _, dc := range rule.AllowedDataClasses {
					if strings.EqualFold(dc, req.DataClassification) {
						allowed = true
						break
					}
				}
				if !allowed {
					return &Decision{
						Effect:          EffectDeny,
						PolicyID:        pol.ID,
						RuleName:        rule.Name,
						Reason:          fmt.Sprintf("data classification %q is not in allowed set", req.DataClassification),
						Timestamp:       now,
						DecisionVersion: pol.Version,
					}
				}
			}

			// Process effect
			switch rule.Effect {
			case EffectDeny:
				// Explicit deny always overrides everything immediately
				return &Decision{
					Effect:          EffectDeny,
					PolicyID:        pol.ID,
					RuleName:        rule.Name,
					Reason:          fmt.Sprintf("explicit deny rule %q triggered", rule.Name),
					Timestamp:       now,
					DecisionVersion: pol.Version,
				}
			case EffectRequireApproval:
				if matchedApproval == nil {
					matchedApproval = &Decision{
						Effect:          EffectRequireApproval,
						PolicyID:        pol.ID,
						RuleName:        rule.Name,
						Reason:          fmt.Sprintf("human approval required by rule %q", rule.Name),
						Timestamp:       now,
						DecisionVersion: pol.Version,
					}
				}
			case EffectAllow:
				if matchedAllow == nil {
					matchedAllow = &Decision{
						Effect:          EffectAllow,
						PolicyID:        pol.ID,
						RuleName:        rule.Name,
						Reason:          fmt.Sprintf("allowed by rule %q", rule.Name),
						Timestamp:       now,
						DecisionVersion: pol.Version,
					}
				}
			}
		}
	}

	// Prioritize approval requirement over normal allow
	if matchedApproval != nil {
		return matchedApproval
	}
	if matchedAllow != nil {
		return matchedAllow
	}

	// Default Deny
	return &Decision{
		Effect:          EffectDeny,
		PolicyID:        "default",
		RuleName:        "default-deny",
		Reason:          "default deny: no matching allow rule found",
		Timestamp:       now,
		DecisionVersion: "v1",
	}
}

func matchesAgent(patterns []string, agentID string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, p := range patterns {
		if p == "*" || p == agentID {
			return true
		}
	}
	return false
}

func matchesString(patterns []string, value string) bool {
	if len(patterns) == 0 {
		return true
	}
	if value == "" {
		return false
	}
	for _, p := range patterns {
		if p == "*" || strings.EqualFold(p, value) {
			return true
		}
	}
	return false
}

func matchesPattern(patterns []string, value string) bool {
	if len(patterns) == 0 {
		return true
	}
	if value == "" {
		return false
	}
	for _, p := range patterns {
		if p == "*" || p == value {
			return true
		}
		// Glob pattern match e.g. "bigquery.*"
		if matched, _ := path.Match(p, value); matched {
			return true
		}
		if strings.HasSuffix(p, ".*") {
			prefix := strings.TrimSuffix(p, ".*")
			if strings.HasPrefix(value, prefix+".") {
				return true
			}
		}
	}
	return false
}

// ValidatePolicy checks structural constraints on a policy object.
func ValidatePolicy(p *Policy) error {
	if p.ID == "" {
		return errors.New("policy ID is required")
	}
	if len(p.Rules) == 0 {
		return errors.New("policy must contain at least one rule")
	}
	for idx, r := range p.Rules {
		if r.Name == "" {
			return fmt.Errorf("rule [%d] must have a name", idx)
		}
		if r.Effect != EffectAllow && r.Effect != EffectDeny && r.Effect != EffectRequireApproval {
			return fmt.Errorf("rule %q has invalid effect %q", r.Name, r.Effect)
		}
	}
	return nil
}
