package agentbom_test

import (
	"testing"

	"github.com/agentmesh/agentmesh/pkg/agentbom"
	"github.com/agentmesh/agentmesh/pkg/contracts"
)

func TestAgentBOMGeneration(t *testing.T) {
	contractYAML := `
apiVersion: agentmesh.dev/v1
kind: AgentContract
metadata:
  name: research-agent
  version: "1.2.0"
  organization: open-research
identity:
  protocols:
    - a2a
    - mcp
capabilities:
  - web_search
  - document_summarize
tools:
  allow:
    - google.search
    - bigquery.read
delegation:
  allow:
    - summarizer-agent
budgets:
  max_cost_per_task: 0.10
`
	c, err := contracts.ParseYAML([]byte(contractYAML))
	if err != nil {
		t.Fatalf("failed to parse contract: %v", err)
	}

	bom, err := agentbom.GenerateFromContract(c, "go", "google-adk")
	if err != nil {
		t.Fatalf("failed to generate BOM: %v", err)
	}

	if bom.Metadata.AgentName != "research-agent" {
		t.Errorf("expected agentName 'research-agent', got %q", bom.Metadata.AgentName)
	}
	if len(bom.Tools) != 2 {
		t.Errorf("expected 2 tools in BOM, got %d", len(bom.Tools))
	}

	jsonData, err := bom.ToJSON()
	if err != nil {
		t.Fatalf("failed to convert BOM to JSON: %v", err)
	}

	parsedBOM, err := agentbom.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("failed to round-trip parse BOM JSON: %v", err)
	}
	if parsedBOM.Agent.Runtime != "go" {
		t.Errorf("expected runtime 'go', got %q", parsedBOM.Agent.Runtime)
	}
}
