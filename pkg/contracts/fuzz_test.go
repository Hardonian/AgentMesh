package contracts_test

import (
	"testing"

	"github.com/agentmesh/agentmesh/pkg/contracts"
)

func FuzzContractParse(f *testing.F) {
	seeds := []string{
		sampleYAML,
		`apiVersion: agentmesh.dev/v1
kind: AgentContract
metadata:
  name: test-agent
identity:
  protocols: [a2a]
`,
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Ensure parser never panics on arbitrary input
		c, err := contracts.ParseYAML(data)
		if err == nil && c != nil {
			_, _ = c.CanonicalJSON()
			_, _ = c.Hash()
		}
	})
}
