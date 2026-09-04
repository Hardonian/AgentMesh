package canary

import (
	"fmt"
	"strings"
	"time"

	"github.com/agentmesh/agentmesh/pkg/contracts"
)

// AgentChangeReport captures structural differences and security-sensitive delta between versions.
type AgentChangeReport struct {
	AgentID                      string    `json:"agentId"`
	CurrentVersion               string    `json:"currentVersion"`
	CandidateVersion             string    `json:"candidateVersion"`
	NewTools                     []string  `json:"newTools"`
	RemovedTools                 []string  `json:"removedTools"`
	NewDelegations               []string  `json:"newDelegations"`
	RemovedDelegations           []string  `json:"removedDelegations"`
	IncreasedDelegationDepth     bool      `json:"increasedDelegationDepth"`
	NewRestrictedDataAccess      bool      `json:"newRestrictedDataAccess"`
	SecuritySensitiveFlags       []string  `json:"securitySensitiveFlags"`
	RequiresExplicitPolicyReview bool      `json:"requiresExplicitPolicyReview"`
	SafeToCanary                 bool      `json:"safeToCanary"`
	GeneratedAt                  time.Time `json:"generatedAt"`
}

// AnalyzeChangeImpact compares current vs candidate AgentContracts for progressive delivery safety.
func AnalyzeChangeImpact(current, candidate *contracts.AgentContract) (*AgentChangeReport, error) {
	if candidate == nil {
		return nil, fmt.Errorf("candidate contract cannot be nil")
	}

	currVer := "none"
	if current != nil {
		currVer = current.Metadata.Version
	}

	report := &AgentChangeReport{
		AgentID:                candidate.Metadata.Name,
		CurrentVersion:         currVer,
		CandidateVersion:       candidate.Metadata.Version,
		NewTools:               make([]string, 0),
		RemovedTools:           make([]string, 0),
		NewDelegations:         make([]string, 0),
		RemovedDelegations:     make([]string, 0),
		SecuritySensitiveFlags: make([]string, 0),
		SafeToCanary:           true,
		GeneratedAt:            time.Now().UTC(),
	}

	currTools := make(map[string]bool)
	if current != nil {
		for _, t := range current.Tools.Allow {
			currTools[t] = true
		}
	}
	candTools := make(map[string]bool)
	for _, t := range candidate.Tools.Allow {
		candTools[t] = true
		if !currTools[t] {
			report.NewTools = append(report.NewTools, t)
			// Check for security-sensitive new tools
			lower := strings.ToLower(t)
			if strings.Contains(lower, "write") || strings.Contains(lower, "update") || strings.Contains(lower, "insert") {
				report.SecuritySensitiveFlags = append(report.SecuritySensitiveFlags, fmt.Sprintf("New write-capable tool added: %s", t))
			}
			if strings.Contains(lower, "delete") || strings.Contains(lower, "drop") || strings.Contains(lower, "destroy") {
				report.SecuritySensitiveFlags = append(report.SecuritySensitiveFlags, fmt.Sprintf("CRITICAL: New destructive tool added: %s", t))
			}
			if strings.Contains(lower, "payment") || strings.Contains(lower, "transfer") || strings.Contains(lower, "financial") {
				report.SecuritySensitiveFlags = append(report.SecuritySensitiveFlags, fmt.Sprintf("CRITICAL: New financial transaction tool added: %s", t))
			}
			if strings.Contains(lower, "http") || strings.Contains(lower, "email") || strings.Contains(lower, "external") {
				report.SecuritySensitiveFlags = append(report.SecuritySensitiveFlags, fmt.Sprintf("New external communication tool added: %s", t))
			}
		}
	}
	for t := range currTools {
		if !candTools[t] {
			report.RemovedTools = append(report.RemovedTools, t)
		}
	}

	// Delegation changes
	currDelegates := make(map[string]bool)
	if current != nil {
		for _, d := range current.Delegation.Allow {
			currDelegates[d] = true
		}
	}
	candDelegates := make(map[string]bool)
	for _, d := range candidate.Delegation.Allow {
		candDelegates[d] = true
		if !currDelegates[d] {
			report.NewDelegations = append(report.NewDelegations, d)
			report.SecuritySensitiveFlags = append(report.SecuritySensitiveFlags, fmt.Sprintf("New delegation peer added: %s", d))
		}
	}
	for d := range currDelegates {
		if !candDelegates[d] {
			report.RemovedDelegations = append(report.RemovedDelegations, d)
		}
	}

	// Delegation depth
	if current != nil && candidate.Delegation.MaxDepth > current.Delegation.MaxDepth {
		report.IncreasedDelegationDepth = true
		report.SecuritySensitiveFlags = append(report.SecuritySensitiveFlags, fmt.Sprintf("Max delegation depth increased from %d to %d", current.Delegation.MaxDepth, candidate.Delegation.MaxDepth))
	}

	// Data classification RESTRICTED access
	for tool, class := range candidate.Tools.DataClassification {
		if strings.EqualFold(class, "RESTRICTED") {
			currClass := ""
			if current != nil {
				currClass = current.Tools.DataClassification[tool]
			}
			if !strings.EqualFold(currClass, "RESTRICTED") {
				report.NewRestrictedDataAccess = true
				report.SecuritySensitiveFlags = append(report.SecuritySensitiveFlags, fmt.Sprintf("New RESTRICTED data classification access on tool %s", tool))
			}
		}
	}

	if len(report.SecuritySensitiveFlags) > 0 {
		report.RequiresExplicitPolicyReview = true
		report.SafeToCanary = false
	}

	return report, nil
}
