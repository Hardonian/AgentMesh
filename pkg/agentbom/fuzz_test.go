package agentbom

import (
	"testing"
)

func FuzzAgentBOMUnmarshal(f *testing.F) {
	f.Add([]byte(`{"apiVersion":"agentmesh.dev/v2alpha1","kind":"AgentBOM","metadata":{"agentName":"test-agent"},"protocols":["mcp"]}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"apiVersion":"invalid"}`))
	f.Add([]byte(`{"apiVersion":"agentmesh.dev/v2alpha1","kind":"AgentBOM","metadata":{"agentName":""},"protocols":[]}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		bom, err := FromJSON(data)
		if err != nil {
			return
		}
		if bom != nil {
			_ = bom.Validate()
			_, _ = bom.Hash()
			_, _ = bom.ToJSON()
		}
	})
}
