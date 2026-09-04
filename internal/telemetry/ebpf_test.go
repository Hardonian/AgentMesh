package telemetry_test

import (
	"testing"

	"github.com/agentmesh/agentmesh/internal/telemetry"
)

func TestEBPFOpsObserver_RecordAndAggregate(t *testing.T) {
	observer := telemetry.NewEBPFOpsObserver()

	// Record activity between Agent A and Agent B
	observer.RecordSocketActivity("agent-alpha", "agent-beta", "A2A", 1024, 12.5, 0)
	observer.RecordSocketActivity("agent-alpha", "agent-beta", "A2A", 2048, 11.0, 0)
	// Record activity between Agent A and an MCP tool
	observer.RecordSocketActivity("agent-alpha", "tool-bigquery", "MCP", 512, 45.0, 1)

	flows := observer.GetFlows()
	if len(flows) != 2 {
		t.Fatalf("expected 2 socket flows, got: %d", len(flows))
	}

	summary := observer.GetAgentSummary("agent-alpha")
	if summary.TotalOutboundBytes != (1024 + 2048 + 512) {
		t.Fatalf("expected 3584 outbound bytes, got: %d", summary.TotalOutboundBytes)
	}
	if summary.ActivePeerCount != 2 {
		t.Fatalf("expected 2 active peers, got: %d", summary.ActivePeerCount)
	}
	if summary.AverageSocketLatency <= 0 {
		t.Fatalf("expected non-zero average socket latency, got: %.2f", summary.AverageSocketLatency)
	}
}
