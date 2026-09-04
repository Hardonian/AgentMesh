package adk

import (
	"fmt"
	"strings"

	"github.com/agentmesh/agentmesh/pkg/graph"
)

// RiskLevel categorizes identified graph risks.
type RiskLevel string

const (
	RiskLow      RiskLevel = "LOW"
	RiskMedium   RiskLevel = "MEDIUM"
	RiskHigh     RiskLevel = "HIGH"
	RiskCritical RiskLevel = "CRITICAL"
)

// RiskFinding describes a concrete risk discovered in an operational AgentGraph.
type RiskFinding struct {
	Level       RiskLevel `json:"level"`
	Dimension   string    `json:"dimension"`
	Description string    `json:"description"`
	NodeID      string    `json:"nodeId,omitempty"`
	Remediation string    `json:"remediation"`
}

// AgentGraphRisk aggregates all dimensional findings for an AgentGraph.
type AgentGraphRisk struct {
	OverallRisk RiskLevel     `json:"overallRisk"`
	GraphID     string        `json:"graphId"`
	AgentID     string        `json:"agentId"`
	Findings    []RiskFinding `json:"findings"`
}

// AnalyzeGraphRisk evaluates the 9 risk dimensions across an AgentGraph.
func AnalyzeGraphRisk(g *graph.AgentGraph) *AgentGraphRisk {
	report := &AgentGraphRisk{
		OverallRisk: RiskLow,
		GraphID:     g.GraphID,
		AgentID:     g.AgentID,
		Findings:    make([]RiskFinding, 0),
	}

	// 1. Cycles
	cycles := g.FindCycles()
	if len(cycles) > 0 {
		report.Findings = append(report.Findings, RiskFinding{
			Level:       RiskCritical,
			Dimension:   "cycles",
			Description: fmt.Sprintf("Detected %d execution cycle(s) in workflow topology: %v", len(cycles), cycles[0]),
			Remediation: "Ensure all delegation and decision paths form a Directed Acyclic Graph (DAG) or implement explicit recursion guards.",
		})
	}

	// 2. Delegation Depth & Peer Count
	if len(g.Delegations) > 5 {
		report.Findings = append(report.Findings, RiskFinding{
			Level:       RiskHigh,
			Dimension:   "delegation depth",
			Description: fmt.Sprintf("High number of direct delegate peers (%d) increases blast radius and delegation complexity.", len(g.Delegations)),
			Remediation: "Consolidate delegation responsibilities or establish intermediate coordinator agents.",
		})
	}

	// 3. Privileged & Destructive Tools
	for _, tool := range g.Tools {
		lower := strings.ToLower(tool)
		if strings.Contains(lower, "delete") || strings.Contains(lower, "drop") || strings.Contains(lower, "destroy") {
			report.Findings = append(report.Findings, RiskFinding{
				Level:       RiskHigh,
				Dimension:   "privileged tools",
				Description: fmt.Sprintf("Graph references destructive tool %q", tool),
				Remediation: "Enforce REQUIRE_APPROVAL policy effect or sandbox destructive operations.",
			})
		}
		if strings.Contains(lower, "payment") || strings.Contains(lower, "transfer") || strings.Contains(lower, "billing") {
			report.Findings = append(report.Findings, RiskFinding{
				Level:       RiskCritical,
				Dimension:   "write-capable tools",
				Description: fmt.Sprintf("Graph accesses financial transaction tool %q", tool),
				Remediation: "Attach cryptographic HITL approval token requirement with exact parameter hash verification.",
			})
		}
	}

	// 4. External Network Access
	for _, tool := range g.Tools {
		lower := strings.ToLower(tool)
		if strings.Contains(lower, "http") || strings.Contains(lower, "fetch") || strings.Contains(lower, "webhook") || strings.Contains(lower, "email") || strings.Contains(lower, "slack") {
			report.Findings = append(report.Findings, RiskFinding{
				Level:       RiskMedium,
				Dimension:   "external network access",
				Description: fmt.Sprintf("Tool %q communicates outside the mesh perimeter", tool),
				Remediation: "Apply egress firewall rules and restrict destination host allowlists.",
			})
		}
	}

	// 5. Approval Coverage
	if len(g.Tools) > 0 && len(g.ApprovalPoints) == 0 {
		hasSensitive := false
		for _, tool := range g.Tools {
			if strings.Contains(strings.ToLower(tool), "write") || strings.Contains(strings.ToLower(tool), "update") {
				hasSensitive = true
				break
			}
		}
		if hasSensitive {
			report.Findings = append(report.Findings, RiskFinding{
				Level:       RiskMedium,
				Dimension:   "approval coverage",
				Description: "Write tools are referenced but zero human-in-the-loop approval checkpoints are declared.",
				Remediation: "Introduce a NodeTypeHumanApproval node or declare approval requirement in AgentContract.",
			})
		}
	}

	// 6. Single-Agent Privilege Concentration
	if len(g.Tools) > 10 {
		report.Findings = append(report.Findings, RiskFinding{
			Level:       RiskMedium,
			Dimension:   "single-agent privilege concentration",
			Description: fmt.Sprintf("Agent references %d distinct tools, violating principle of least privilege.", len(g.Tools)),
			Remediation: "Decompose into modular sub-agents with specialized, narrow tool sets.",
		})
	}

	// Calculate overall risk
	for _, f := range report.Findings {
		if f.Level == RiskCritical {
			report.OverallRisk = RiskCritical
			break
		} else if f.Level == RiskHigh && report.OverallRisk != RiskCritical {
			report.OverallRisk = RiskHigh
		} else if f.Level == RiskMedium && report.OverallRisk != RiskCritical && report.OverallRisk != RiskHigh {
			report.OverallRisk = RiskMedium
		}
	}

	return report
}
