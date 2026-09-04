package graph_test

import (
	"testing"

	"github.com/agentmesh/agentmesh/pkg/graph"
)

func TestAgentGraphValidationAndCycles(t *testing.T) {
	g := graph.NewAgentGraph("g_test", "org_acme", "proj_alpha", "procurement-agent", "1.0.0")
	g.Entrypoint = "node_start"

	g.Nodes = append(g.Nodes,
		graph.Node{ID: "node_start", Name: "Start", Type: graph.NodeTypeDecision},
		graph.Node{ID: "node_search", Name: "Vendor Search", Type: graph.NodeTypeTool, Target: "bigquery.read"},
		graph.Node{ID: "node_delegate", Name: "Delegate to Finance", Type: graph.NodeTypeDelegation, Target: "finance-agent"},
	)
	g.Tools = append(g.Tools, "bigquery.read")
	g.Delegations = append(g.Delegations, "finance-agent")

	// Linear flow: start -> search -> delegate
	g.Edges = append(g.Edges,
		graph.Edge{FromID: "node_start", ToID: "node_search"},
		graph.Edge{FromID: "node_search", ToID: "node_delegate"},
	)

	if err := g.Validate(); err != nil {
		t.Fatalf("expected valid graph, got: %v", err)
	}

	cycles := g.FindCycles()
	if len(cycles) > 0 {
		t.Fatalf("expected 0 cycles in DAG, got %d: %v", len(cycles), cycles)
	}

	// Add cyclic edge: delegate -> start
	g.Edges = append(g.Edges, graph.Edge{FromID: "node_delegate", ToID: "node_start"})
	cycles = g.FindCycles()
	if len(cycles) == 0 {
		t.Fatal("expected cycle detection when back-edge added, but got 0")
	}

	hash1, err := g.Hash()
	if err != nil {
		t.Fatalf("failed to hash graph: %v", err)
	}
	hash2, _ := g.Hash()
	if hash1 != hash2 {
		t.Fatalf("expected deterministic hash, got %s != %s", hash1, hash2)
	}
}

func TestGraphDiff(t *testing.T) {
	g1 := graph.NewAgentGraph("g1", "org", "p", "ag", "1.0.0")
	g1.Entrypoint = "n1"
	g1.Nodes = append(g1.Nodes, graph.Node{ID: "n1", Name: "Start", Type: graph.NodeTypeAgent})
	g1.Tools = []string{"bigquery.read"}

	g2 := graph.NewAgentGraph("g2", "org", "p", "ag", "1.1.0")
	g2.Entrypoint = "n1"
	g2.Nodes = append(g2.Nodes,
		graph.Node{ID: "n1", Name: "Start", Type: graph.NodeTypeAgent},
		graph.Node{ID: "n2", Name: "New Tool", Type: graph.NodeTypeTool, Target: "gcs.write"},
	)
	g2.Tools = []string{"bigquery.read", "gcs.write"}
	g2.Delegations = []string{"finance-agent"}

	diff := graph.DiffGraphs(g1, g2)
	if len(diff.AddedTools) != 1 || diff.AddedTools[0] != "gcs.write" {
		t.Fatalf("expected added tool gcs.write, got: %v", diff.AddedTools)
	}
	if len(diff.AddedDelegations) != 1 || diff.AddedDelegations[0] != "finance-agent" {
		t.Fatalf("expected added delegation finance-agent, got: %v", diff.AddedDelegations)
	}
	if len(diff.AddedNodes) != 1 || diff.AddedNodes[0] != "n2" {
		t.Fatalf("expected added node n2, got: %v", diff.AddedNodes)
	}
}
