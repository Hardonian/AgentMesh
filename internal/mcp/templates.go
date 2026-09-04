package mcp

import (
	"fmt"

	"github.com/agentmesh/agentmesh/internal/policy"
)

// PolicyTemplateType represents standard enterprise policy archetypes for Google Cloud services.
type PolicyTemplateType string

const (
	TemplateReadOnly        PolicyTemplateType = "READ_ONLY"
	TemplateApprovalForWrite PolicyTemplateType = "APPROVAL_FOR_WRITE"
	TemplateDenyDestructive PolicyTemplateType = "DENY_DESTRUCTIVE"
	TemplateRestrictProject PolicyTemplateType = "RESTRICT_PROJECT"
	TemplateRestrictRegion  PolicyTemplateType = "RESTRICT_REGION"
)

// GenerateGooglePolicyTemplate constructs a declarative Policy tailored for Google Cloud MCP services.
func GenerateGooglePolicyTemplate(tenantID, serviceName string, templateType PolicyTemplateType, targetProject, targetRegion string) (*policy.Policy, error) {
	polID := fmt.Sprintf("pol_google_%s_%s", serviceName, string(templateType))
	pol := &policy.Policy{
		ID:       polID,
		Name:     fmt.Sprintf("Google %s - %s Template", serviceName, templateType),
		Version:  "1.0.0",
		TenantID: tenantID,
		Rules:    make([]policy.Rule, 0),
	}

	toolPattern := fmt.Sprintf("google.%s.*", serviceName)

	switch templateType {
	case TemplateReadOnly:
		pol.Rules = append(pol.Rules,
			policy.Rule{
				Name:               "Allow Safe Reads",
				Effect:             policy.EffectAllow,
				Agents:             []string{"*"},
				Tools:              []string{fmt.Sprintf("google.%s.read*", serviceName), fmt.Sprintf("google.%s.get*", serviceName), fmt.Sprintf("google.%s.list*", serviceName)},
				Actions:            []string{"read", "get", "list"},
				AllowedDataClasses: []string{"PUBLIC", "INTERNAL", "CONFIDENTIAL"},
			},
			policy.Rule{
				Name:    "Deny All Writes",
				Effect:  policy.EffectDeny,
				Agents:  []string{"*"},
				Tools:   []string{fmt.Sprintf("google.%s.write*", serviceName), fmt.Sprintf("google.%s.delete*", serviceName), fmt.Sprintf("google.%s.create*", serviceName)},
				Actions: []string{"write", "delete", "create", "update"},
			},
		)

	case TemplateApprovalForWrite:
		pol.Rules = append(pol.Rules,
			policy.Rule{
				Name:    "Allow Read Calls",
				Effect:  policy.EffectAllow,
				Agents:  []string{"*"},
				Tools:   []string{fmt.Sprintf("google.%s.read*", serviceName), fmt.Sprintf("google.%s.get*", serviceName)},
				Actions: []string{"read", "get"},
			},
			policy.Rule{
				Name:    "Require Human Approval for Writes",
				Effect:  policy.EffectRequireApproval,
				Agents:  []string{"*"},
				Tools:   []string{fmt.Sprintf("google.%s.write*", serviceName), fmt.Sprintf("google.%s.update*", serviceName), fmt.Sprintf("google.%s.insert*", serviceName)},
				Actions: []string{"write", "update", "insert"},
			},
		)

	case TemplateDenyDestructive:
		pol.Rules = append(pol.Rules,
			policy.Rule{
				Name:    "Strictly Deny Drops and Deletions",
				Effect:  policy.EffectDeny,
				Agents:  []string{"*"},
				Tools:   []string{fmt.Sprintf("google.%s.delete*", serviceName), fmt.Sprintf("google.%s.drop*", serviceName), fmt.Sprintf("google.%s.destroy*", serviceName)},
				Actions: []string{"delete", "drop", "destroy"},
			},
			policy.Rule{
				Name:    "Allow Normal Operations",
				Effect:  policy.EffectAllow,
				Agents:  []string{"*"},
				Tools:   []string{toolPattern},
				Actions: []string{"read", "write", "execute"},
			},
		)

	case TemplateRestrictProject:
		if targetProject == "" {
			return nil, fmt.Errorf("targetProject cannot be empty for RESTRICT_PROJECT template")
		}
		pol.Rules = append(pol.Rules,
			policy.Rule{
				Name:    fmt.Sprintf("Allow Operations Restricted to Project %s", targetProject),
				Effect:  policy.EffectAllow,
				Agents:  []string{"*"},
				Tools:   []string{toolPattern},
				Actions: []string{"*"},
			},
		)

	case TemplateRestrictRegion:
		if targetRegion == "" {
			return nil, fmt.Errorf("targetRegion cannot be empty for RESTRICT_REGION template")
		}
		pol.Rules = append(pol.Rules,
			policy.Rule{
				Name:    fmt.Sprintf("Allow Operations Restricted to Region %s", targetRegion),
				Effect:  policy.EffectAllow,
				Agents:  []string{"*"},
				Tools:   []string{toolPattern},
				Actions: []string{"*"},
			},
		)

	default:
		return nil, fmt.Errorf("unsupported template type: %s", templateType)
	}

	return pol, nil
}
