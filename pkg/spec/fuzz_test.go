package spec

import (
	"encoding/json"
	"testing"
)

func FuzzSpecUnmarshal(f *testing.F) {
	f.Add([]byte(`{"capabilityId":"code-review","organizationId":"tenant-1","weights":{"agent-1":100}}`))
	f.Add([]byte(`{"actionId":"act-1","actionType":"CHANGE_ROUTE_WEIGHT","riskClass":"LOW"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"weights":{"a":-10,"b":9999999}}`))
	f.Add([]byte(`null`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var rSpec AgentRoutingSpec
		if err := json.Unmarshal(data, &rSpec); err == nil {
			_ = rSpec.ComputeSpecHash()
		}

		var optAction AgentOptimizationAction
		if err := json.Unmarshal(data, &optAction); err == nil {
			_ = optAction.ComputeActionHash()
		}
	})
}
