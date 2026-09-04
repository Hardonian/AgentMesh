package cost_test

import (
	"testing"

	"github.com/agentmesh/agentmesh/internal/cost"
)

func TestTaskCostTreeAndAnomalyDetection(t *testing.T) {
	tree := cost.NewTaskCostTree("task_1", "root-agent")
	tree.AddModelCall("gemini-1.5-pro", "gemini", 1000, 200, 0.00225)
	tree.AddToolCall("google.bigquery.query", "bigquery-mcp", 0.005)
	tree.AddDelegation("sub-agent", 0.010)

	if tree.TotalCostUSD <= 0.017 {
		t.Fatalf("expected total cost ~0.01725, got %f", tree.TotalCostUSD)
	}
	if len(tree.DelegationCosts) != 1 || tree.DelegationCosts[0].PercentOfTotal <= 0 {
		t.Fatalf("expected delegation percentage calculation")
	}

	// Anomaly test: normal cost
	anomNormal := cost.DetectCostAnomaly(0.02, 0.02, 3.0)
	if anomNormal.IsAnomaly {
		t.Fatalf("did not expect anomaly on normal cost")
	}

	// Anomaly test: spike
	anomSpike := cost.DetectCostAnomaly(0.25, 0.02, 3.0)
	if !anomSpike.IsAnomaly {
		t.Fatalf("expected anomaly detection when cost is 12.5x historical average")
	}
}
