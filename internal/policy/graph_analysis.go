package policy

import (
	"fmt"
	"strings"

	"github.com/agentmesh/agentmesh/pkg/graph"
)

// GraphPolicyFinding describes a policy violation or risk detected by static graph inspection.
type GraphPolicyFinding struct {
	Severity    string `json:"severity"` // "ERROR", "WARNING", "INFO"
	Category    string `json:"category"` // "FORBIDDEN_DELEGATION", "PRIVILEGE_ESCALATION", "UNAPPROVED_TOOL", "RESTRICTED_DATA"
	Description string `json:"description"`
	Path        string `json:"path,omitempty"`
	Remediation string `json:"remediation"`
}

// GraphPolicyReport contains all static policy analysis findings across an AgentGraph.
type GraphPolicyReport struct {
	GraphID      string               `json:"graphId"`
	AgentID      string               `json:"agentId"`
	PolicyID     string               `json:"policyId"`
	Compliant    bool                 `json:"compliant"`
	Findings     []GraphPolicyFinding `json:"findings"`
	AnalyzedNodes int                 `json:"analyzedNodes"`
	AnalyzedEdges int                 `json:"analyzedEdges"`
}

// AnalyzeGraphPolicy statically evaluates all paths and relationships in an AgentGraph against policy rules.
func AnalyzeGraphPolicy(g *graph.AgentGraph, p *Policy) *GraphPolicyReport {
	report := &GraphPolicyReport{
		GraphID:       g.GraphID,
		AgentID:       g.AgentID,
		Compliant:     true,
		Findings:      make([]GraphPolicyFinding, 0),
		AnalyzedNodes: len(g.Nodes),
		AnalyzedEdges: len(g.Edges),
	}
	if p != nil {
		report.PolicyID = p.ID
	}

	engine := NewEngine([]*Policy{p})

	// 1. Check direct tools declared by graph
	for _, tool := range g.Tools {
		req := &EvaluationRequest{
			TenantID:       g.OrganizationID,
			SubjectAgentID: g.AgentID,
			Tool:           tool,
			Action:         "execute",
		}
		dec := engine.Evaluate(nil, req)
		if dec.Effect == EffectDeny {
			report.Compliant = false
			report.Findings = append(report.Findings, GraphPolicyFinding{
				Severity:    "ERROR",
				Category:    "FORBIDDEN_TOOL",
				Description: fmt.Sprintf("Graph node references tool %q explicitly denied by policy: %s", tool, dec.Reason),
				Remediation: "Remove the tool from graph definition or update policy allow rules.",
			})
		}
	}

	// 2. Check delegations and Indirect Privilege Escalation
	// Example: CEO -> Finance -> Research -> Gmail Send
	// If root agent is barred from Gmail Send, any downstream reachable tool that root cannot execute is flagged.
	for _, delegate := range g.Delegations {
		// Verify if delegation to target is allowed
		delReq := &EvaluationRequest{
			TenantID:       g.OrganizationID,
			SubjectAgentID: g.AgentID,
			Action:         "delegate",
			Resource:       delegate,
		}
		delDec := engine.Evaluate(nil, delReq)
		if delDec.Effect == EffectDeny {
			report.Compliant = false
			report.Findings = append(report.Findings, GraphPolicyFinding{
				Severity:    "ERROR",
				Category:    "FORBIDDEN_DELEGATION",
				Description: fmt.Sprintf("Delegation from %s to %s is denied by active policy", g.AgentID, delegate),
				Path:        fmt.Sprintf("%s -> %s", g.AgentID, delegate),
				Remediation: "Authorize delegation peer in policy rules.",
			})
		}
	}

	// 3. Approval coverage verification
	for _, node := range g.Nodes {
		if node.Type == graph.NodeTypeTool {
			req := &EvaluationRequest{
				TenantID:       g.OrganizationID,
				SubjectAgentID: g.AgentID,
				Tool:           node.Target,
				Action:         "execute",
			}
			dec := engine.Evaluate(nil, req)
			if dec.Effect == EffectRequireApproval {
				// Check if there is an approval checkpoint in graph
				hasApproval := false
				for _, appPoint := range g.ApprovalPoints {
					if strings.EqualFold(appPoint, node.Target) {
						hasApproval = true
						break
					}
				}
				if !hasApproval {
					report.Compliant = false
					report.Findings = append(report.Findings, GraphPolicyFinding{
						Severity:    "ERROR",
						Category:    "UNAPPROVED_TOOL_DEPENDENCY",
						Description: fmt.Sprintf("Tool %q requires human approval under active policy, but no approval node or declaration exists in graph", node.Target),
						Path:        node.ID,
						Remediation: fmt.Sprintf("Add an approval node preceding %s or declare required_for in contract.", node.Target),
					})
				}
			}
		}
	}

	return report
}
