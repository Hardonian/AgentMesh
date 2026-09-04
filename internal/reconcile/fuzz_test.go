package reconcile

import (
	"encoding/json"
	"testing"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

func FuzzReconcileParse(f *testing.F) {
	f.Add([]byte(`{"capabilityId":"code-review","weights":{"agent-1":100}}`), []byte(`{"capabilityId":"code-review","weights":{"agent-2":100}}`))
	f.Add([]byte(`{}`), []byte(`{}`))
	f.Add([]byte(`null`), []byte(`null`))

	engine := NewEngine()

	f.Fuzz(func(t *testing.T, currentBytes, desiredBytes []byte) {
		var current spec.AgentRoutingSpec
		_ = json.Unmarshal(currentBytes, &current)

		var desired spec.AgentRoutingSpec
		_ = json.Unmarshal(desiredBytes, &desired)

		plan, err := engine.PlanRoutingReconciliation(&current, &desired)
		if err == nil && plan != nil {
			_ = plan.Steps
		}
	})
}
