package outcome

import (
	"testing"
	"time"
)

func TestOperationalOutcomeGraph(t *testing.T) {
	og := NewOperationalOutcomeGraph()
	tenant := "acme-corp"

	// 1. Add Agent, Invocation, Tool, Policy nodes
	agentNode := &GraphNode{
		ID:       "finance-agent",
		Type:     NodeAgent,
		TenantID: tenant,
	}
	_ = og.AddNode(agentNode)

	toolNode := &GraphNode{
		ID:       "bigquery-tool",
		Type:     NodeTool,
		TenantID: tenant,
	}
	_ = og.AddNode(toolNode)

	invNode := &GraphNode{
		ID:       "inv-12345",
		Type:     NodeInvocation,
		TenantID: tenant,
		Properties: map[string]any{
			"duration_ms": float64(4500),
		},
	}
	_ = og.AddNode(invNode)

	// 2. Add edges: Invocation invoked BigQuery tool, and BigQuery tool failed
	_ = og.AddEdge(&GraphEdge{
		ID:       "edge-1",
		TenantID: tenant,
		Type:     EdgeInvocationFailed,
		FromID:   "inv-12345",
		ToID:     "bigquery-tool",
		Properties: map[string]any{
			"latency_ms": float64(4200),
		},
		CreatedAt: time.Now().UTC(),
	})

	// 3. Test Root Cause Analysis
	rca := og.AnalyzeRootCause(tenant, "inv-12345")
	if rca.LikelySource != SourceToolTimeout {
		t.Errorf("expected SourceToolTimeout, got %s", rca.LikelySource)
	}
	if rca.FailingEntity != "bigquery-tool" {
		t.Errorf("expected failing entity 'bigquery-tool', got %s", rca.FailingEntity)
	}
	if rca.Confidence < 0.8 {
		t.Errorf("expected high confidence, got %f", rca.Confidence)
	}

	// 4. Test Bottleneck Analysis
	bottleneck := og.AnalyzeBottlenecks(tenant, "inv-12345")
	if bottleneck.DominantLatencySource != "bigquery-tool" {
		t.Errorf("expected dominant latency source 'bigquery-tool', got %s", bottleneck.DominantLatencySource)
	}
	if len(bottleneck.Recommendations) == 0 {
		t.Errorf("expected optimization recommendations")
	}
}
