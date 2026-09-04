package contracts_test

import (
	"testing"

	"github.com/agentmesh/agentmesh/pkg/contracts"
)

const sampleYAML = `
apiVersion: agentmesh.dev/v1
kind: AgentContract

metadata:
  name: procurement-agent
  organization: acme-corp
  version: "1.0.0"

identity:
  protocols:
    - a2a
    - mcp

capabilities:
  - vendor_search
  - quote_analysis
  - purchase_request

tools:
  allow:
    - bigquery.read
    - drive.read
    - internal.erp.quote
  deny:
    - gmail.send
    - payment.execute

delegation:
  allow:
    - research-agent
    - finance-agent
  maxDepth: 3

budgets:
  max_cost_per_task: 0.25
  max_tokens_per_task: 100000

slo:
  p95_latency_ms: 12000
  success_rate: 0.995

approval:
  required_for:
    - purchase_request
`

func TestParseYAMLAndHash(t *testing.T) {
	c, err := contracts.ParseYAML([]byte(sampleYAML))
	if err != nil {
		t.Fatalf("unexpected error parsing contract: %v", err)
	}

	if c.Metadata.Name != "procurement-agent" {
		t.Errorf("expected name 'procurement-agent', got %q", c.Metadata.Name)
	}

	hash1, err := c.Hash()
	if err != nil {
		t.Fatalf("unexpected error hashing contract: %v", err)
	}
	if hash1 == "" {
		t.Fatal("expected non-empty hash")
	}

	// Parsing again should produce identical hash (deterministic)
	c2, _ := contracts.ParseYAML([]byte(sampleYAML))
	hash2, _ := c2.Hash()
	if hash1 != hash2 {
		t.Errorf("contract hash is not deterministic: %s != %s", hash1, hash2)
	}
}

func TestValidationErrors(t *testing.T) {
	invalidYAML := `
apiVersion: invalid/v1
kind: AgentContract
metadata:
  name: test
identity:
  protocols:
    - a2a
`
	_, err := contracts.ParseYAML([]byte(invalidYAML))
	if err == nil {
		t.Error("expected error for invalid apiVersion, got nil")
	}
}

func TestContractDiff(t *testing.T) {
	c1, _ := contracts.ParseYAML([]byte(sampleYAML))
	
	modYAML := sampleYAML + `
  - extra_cap
`
	c2, _ := contracts.ParseYAML([]byte(sampleYAML))
	c2.Capabilities = append(c2.Capabilities, "new_capability")
	diff := c2.Diff(c1)
	if len(diff.AddedCapabilities) != 1 || diff.AddedCapabilities[0] != "new_capability" {
		t.Errorf("expected new_capability in diff, got %+v", diff.AddedCapabilities)
	}
	_ = modYAML
}
