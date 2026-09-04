package adk_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agentmesh/agentmesh/internal/adk"
	"github.com/agentmesh/agentmesh/pkg/contracts"
	"github.com/agentmesh/agentmesh/pkg/graph"
)

func TestDelegationTaintAndConfusedDeputy(t *testing.T) {
	// Origin principal: restricted-agent (cannot execute payment.*)
	originContract := &contracts.AgentContract{
		Metadata: contracts.Metadata{Name: "restricted-agent"},
		Tools: contracts.ToolsConfig{
			Deny: []string{"payment.*"},
		},
	}

	sc := adk.NewSecurityContext("restricted-agent", 4)

	// Delegate to privileged-agent
	privilegedContract := &contracts.AgentContract{
		Metadata: contracts.Metadata{Name: "privileged-agent"},
		Tools: contracts.ToolsConfig{
			Allow: []string{"payment.execute", "bigquery.read"},
		},
	}

	if err := sc.PushDelegation("privileged-agent", privilegedContract); err != nil {
		t.Fatalf("failed to push delegation: %v", err)
	}

	// Privileged agent attempts payment.execute on behalf of restricted-agent
	allowed, reason := sc.CanExecuteTool("payment.execute", originContract)
	if allowed {
		t.Fatalf("confused deputy vulnerability! restricted-agent was able to execute payment through privileged-agent")
	}
	if reason == "" {
		t.Fatal("expected reason for denied tool execution")
	}

	// Safe tool execution: bigquery.read
	allowedSafe, _ := sc.CanExecuteTool("bigquery.read", originContract)
	if !allowedSafe {
		t.Fatalf("expected bigquery.read to be allowed")
	}

	// Test cycle in delegation
	err := sc.PushDelegation("restricted-agent", originContract)
	if err == nil {
		t.Fatal("expected cycle error when pushing restricted-agent back to chain")
	}
}

func TestGraphRiskAnalysis(t *testing.T) {
	g := graph.NewAgentGraph("g_risk", "org", "proj", "risky-agent", "1.0.0")
	g.Entrypoint = "n1"
	g.Nodes = []graph.Node{
		{ID: "n1", Name: "Start", Type: graph.NodeTypeAgent},
		{ID: "n2", Name: "Delete Data", Type: graph.NodeTypeTool, Target: "bigquery.drop_table"},
		{ID: "n3", Name: "Transfer Funds", Type: graph.NodeTypeTool, Target: "payment.transfer"},
	}
	g.Tools = []string{"bigquery.drop_table", "payment.transfer"}
	g.Edges = []graph.Edge{
		{FromID: "n1", ToID: "n2"},
		{FromID: "n2", ToID: "n3"},
	}

	risk := adk.AnalyzeGraphRisk(g)
	if risk.OverallRisk != adk.RiskCritical {
		t.Fatalf("expected CRITICAL risk due to financial tool, got %s", risk.OverallRisk)
	}

	hasFinancialFinding := false
	for _, f := range risk.Findings {
		if f.Level == adk.RiskCritical && f.Dimension == "write-capable tools" {
			hasFinancialFinding = true
			break
		}
	}
	if !hasFinancialFinding {
		t.Fatalf("expected finding for financial tool, got: %v", risk.Findings)
	}
}

func TestStaticProjectInspector(t *testing.T) {
	// Create a temporary Go source file that resembles an ADK agent
	tmpDir, err := os.MkdirTemp("", "adk-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	goCode := `package main

func RunWorkflow() {
	Use("bigquery.read")
	DelegateTo("finance-agent")
	RequireApproval("purchase_request")
}
`
	srcPath := filepath.Join(tmpDir, "workflow.go")
	if err := os.WriteFile(srcPath, []byte(goCode), 0644); err != nil {
		t.Fatalf("failed to write test go file: %v", err)
	}

	inspector := adk.NewStaticProjectInspector()
	res, err := inspector.InspectProject(tmpDir, "test-agent", "test-org")
	if err != nil {
		t.Fatalf("failed to inspect project: %v", err)
	}

	if len(res.IdentifiedTools) != 1 || res.IdentifiedTools[0] != "bigquery.read" {
		t.Fatalf("expected tool 'bigquery.read', got: %v", res.IdentifiedTools)
	}
	if len(res.IdentifiedDelegations) != 1 || res.IdentifiedDelegations[0] != "finance-agent" {
		t.Fatalf("expected delegation 'finance-agent', got: %v", res.IdentifiedDelegations)
	}
	if len(res.IdentifiedApprovals) != 1 || res.IdentifiedApprovals[0] != "purchase_request" {
		t.Fatalf("expected approval 'purchase_request', got: %v", res.IdentifiedApprovals)
	}
}
