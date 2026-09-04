package policy

import (
	"fmt"
	"time"
)

// PolicyPackType defines enterprise archetypes.
type PolicyPackType string

const (
	PackReadOnlyAnalytics       PolicyPackType = "READ_ONLY_ANALYTICS"
	PackControlledCommunication PolicyPackType = "CONTROLLED_COMMUNICATION"
	PackInfrastructureOperator  PolicyPackType = "INFRASTRUCTURE_OPERATOR"
	PackFinanceApproval         PolicyPackType = "FINANCE_APPROVAL"
)

// GetPolicyPack constructs a ready-to-deploy enterprise policy template.
func GetPolicyPack(packType PolicyPackType, tenantID string) (*Policy, error) {
	now := time.Now().UTC()
	pol := &Policy{
		ID:        fmt.Sprintf("pack_%s_%s", stringsToLower(string(packType)), tenantID),
		Version:   "1.0.0",
		TenantID:  tenantID,
		CreatedAt: now,
		Rules:     make([]Rule, 0),
	}

	switch packType {
	case PackReadOnlyAnalytics:
		pol.Name = "Enterprise Read-Only Analytics Pack"
		pol.Description = "Permits data queries on Internal and Confidential data; bars all destructive or modifying actions."
		pol.Rules = []Rule{
			{
				Name:               "Allow Query Tools",
				Effect:             EffectAllow,
				Agents:             []string{"*"},
				Tools:              []string{"bigquery.read", "bigquery.query", "postgres.select", "storage.get"},
				Actions:            []string{"read", "query", "select"},
				AllowedDataClasses: []string{DataClassPublic, DataClassInternal, DataClassConfidential},
			},
			{
				Name:    "Explicit Deny Modifying Operations",
				Effect:  EffectDeny,
				Agents:  []string{"*"},
				Tools:   []string{"*.delete*", "*.drop*", "*.insert*", "*.update*", "*.write*"},
				Actions: []string{"write", "delete", "drop", "update"},
			},
			{
				Name:            "Bar Restricted Data",
				Effect:          EffectDeny,
				Agents:          []string{"*"},
				DenyDataClasses: []string{DataClassRestricted},
			},
		}

	case PackControlledCommunication:
		pol.Name = "Controlled External Communication Pack"
		pol.Description = "Enforces approval on external messaging (Slack, Email, Webhooks)."
		pol.Rules = []Rule{
			{
				Name:    "Require Approval for External Send",
				Effect:  EffectRequireApproval,
				Agents:  []string{"*"},
				Tools:   []string{"gmail.send", "slack.post", "http.post", "webhook.trigger"},
				Actions: []string{"send", "post", "trigger"},
			},
			{
				Name:    "Allow Internal Mesh Communication",
				Effect:  EffectAllow,
				Agents:  []string{"*"},
				Actions: []string{"invoke", "delegate"},
			},
		}

	case PackInfrastructureOperator:
		pol.Name = "Infrastructure Operator Pack"
		pol.Description = "Allows safe cluster and workload read; requires approval for deployments and deletions."
		pol.Rules = []Rule{
			{
				Name:    "Allow Cluster Read",
				Effect:  EffectAllow,
				Agents:  []string{"*"},
				Tools:   []string{"k8s.get*", "k8s.list*", "cloudrun.get*"},
				Actions: []string{"get", "list", "describe"},
			},
			{
				Name:    "Require Approval for Mutating Cluster Resources",
				Effect:  EffectRequireApproval,
				Agents:  []string{"*"},
				Tools:   []string{"k8s.apply*", "k8s.delete*", "cloudrun.deploy*"},
				Actions: []string{"apply", "delete", "deploy"},
			},
		}

	case PackFinanceApproval:
		pol.Name = "Finance & Expenditure Approval Pack"
		pol.Description = "Strictly gates financial and purchase tools behind cryptographic HITL approval tokens."
		pol.Rules = []Rule{
			{
				Name:    "Gate Payments and Purchases",
				Effect:  EffectRequireApproval,
				Agents:  []string{"*"},
				Tools:   []string{"payment.*", "billing.*", "purchase.*", "stripe.*"},
				Actions: []string{"execute", "pay", "charge", "refund"},
			},
			{
				Name:    "Allow Balance and Invoice Read",
				Effect:  EffectAllow,
				Agents:  []string{"*"},
				Tools:   []string{"payment.get_balance", "invoice.read"},
				Actions: []string{"read", "get"},
			},
		}

	default:
		return nil, fmt.Errorf("unknown policy pack type: %s", packType)
	}

	return pol, nil
}

func stringsToLower(s string) string {
	res := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		res[i] = c
	}
	return string(res)
}
