package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

const (
	CurrentSchemaVersion = "agentmesh.dev/v1alpha1"
)

// NodeType represents the semantic role of an execution node within an AgentGraph.
type NodeType string

const (
	NodeTypeAgent         NodeType = "AGENT"
	NodeTypeTool          NodeType = "TOOL"
	NodeTypeDecision      NodeType = "DECISION"
	NodeTypeDelegation    NodeType = "DELEGATION"
	NodeTypeHumanApproval NodeType = "HUMAN_APPROVAL"
	NodeTypeModelCall     NodeType = "MODEL_CALL"
	NodeTypeExternalCall  NodeType = "EXTERNAL_CALL"
)

// Node represents a single step, tool, model, or branch point in an agent workflow.
type Node struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        NodeType          `json:"type"`
	Target      string            `json:"target,omitempty"` // Tool name, agent ID, or model name
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Edge represents a directional flow between two nodes.
type Edge struct {
	FromID    string `json:"fromId"`
	ToID      string `json:"toId"`
	Condition string `json:"condition,omitempty"`
}

// AgentGraph is the canonical, normalized representation of an AI agent's operational topology.
type AgentGraph struct {
	GraphID        string    `json:"graph_id"`
	OrganizationID string    `json:"organization_id"`
	ProjectID      string    `json:"project_id"`
	AgentID        string    `json:"agent_id"`
	Version        string    `json:"version"`
	SchemaVersion  string    `json:"schema_version"`
	Entrypoint     string    `json:"entrypoint"`
	Exitpoints     []string  `json:"exitpoints"`
	Nodes          []Node    `json:"nodes"`
	Edges          []Edge    `json:"edges"`
	Tools          []string  `json:"tools"`
	Delegations    []string  `json:"delegations"`
	ApprovalPoints []string  `json:"approval_points"`
	ModelCalls     []string  `json:"model_calls"`
	ExternalCalls  []string  `json:"external_calls"`
	CreatedAt      time.Time `json:"created_at"`
}

// NewAgentGraph constructs a valid AgentGraph structure.
func NewAgentGraph(graphID, orgID, projectID, agentID, version string) *AgentGraph {
	return &AgentGraph{
		GraphID:        graphID,
		OrganizationID: orgID,
		ProjectID:      projectID,
		AgentID:        agentID,
		Version:        version,
		SchemaVersion:  CurrentSchemaVersion,
		Exitpoints:     make([]string, 0),
		Nodes:          make([]Node, 0),
		Edges:          make([]Edge, 0),
		Tools:          make([]string, 0),
		Delegations:    make([]string, 0),
		ApprovalPoints: make([]string, 0),
		ModelCalls:     make([]string, 0),
		ExternalCalls:  make([]string, 0),
		CreatedAt:      time.Now().UTC(),
	}
}

// Validate checks for structural integrity, single entrypoint, valid node references, and bounds.
func (g *AgentGraph) Validate() error {
	if g.GraphID == "" {
		return errors.New("graph_id is required")
	}
	if g.AgentID == "" {
		return errors.New("agent_id is required")
	}
	if g.Entrypoint == "" {
		return errors.New("entrypoint is required")
	}
	if len(g.Nodes) == 0 {
		return errors.New("graph must contain at least one node")
	}
	if len(g.Nodes) > 5000 {
		return errors.New("graph node limit exceeded (max 5000 nodes)")
	}

	nodeMap := make(map[string]bool)
	entrypointFound := false

	for _, node := range g.Nodes {
		if node.ID == "" {
			return errors.New("node id cannot be empty")
		}
		if nodeMap[node.ID] {
			return fmt.Errorf("duplicate node id: %s", node.ID)
		}
		nodeMap[node.ID] = true
		if node.ID == g.Entrypoint {
			entrypointFound = true
		}
	}

	if !entrypointFound {
		return fmt.Errorf("entrypoint node '%s' does not exist in graph nodes", g.Entrypoint)
	}

	for _, edge := range g.Edges {
		if !nodeMap[edge.FromID] {
			return fmt.Errorf("edge fromId '%s' does not exist", edge.FromID)
		}
		if !nodeMap[edge.ToID] {
			return fmt.Errorf("edge toId '%s' does not exist", edge.ToID)
		}
	}

	return nil
}

// Canonicalize sorts all nodes, edges, tools, and slices to produce deterministic representation.
func (g *AgentGraph) Canonicalize() {
	sort.Slice(g.Nodes, func(i, j int) bool {
		return g.Nodes[i].ID < g.Nodes[j].ID
	})
	sort.Slice(g.Edges, func(i, j int) bool {
		if g.Edges[i].FromID == g.Edges[j].FromID {
			return g.Edges[i].ToID < g.Edges[j].ToID
		}
		return g.Edges[i].FromID < g.Edges[j].FromID
	})
	sort.Strings(g.Exitpoints)
	sort.Strings(g.Tools)
	sort.Strings(g.Delegations)
	sort.Strings(g.ApprovalPoints)
	sort.Strings(g.ModelCalls)
	sort.Strings(g.ExternalCalls)
}

// Hash returns the deterministic SHA-256 digest of the canonical graph structure.
func (g *AgentGraph) Hash() (string, error) {
	clone := *g
	clone.Canonicalize()
	// Zero non-semantic timestamp for repeatable hashing
	clone.CreatedAt = time.Time{}

	bytes, err := json.Marshal(clone)
	if err != nil {
		return "", fmt.Errorf("marshal canonical graph: %w", err)
	}

	h := sha256.Sum256(bytes)
	return hex.EncodeToString(h[:]), nil
}

// FindCycles performs Tarjan/DFS traversal to detect cycles within the graph.
func (g *AgentGraph) FindCycles() [][]string {
	adj := make(map[string][]string)
	for _, edge := range g.Edges {
		adj[edge.FromID] = append(adj[edge.FromID], edge.ToID)
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var cycles [][]string
	var path []string

	var dfs func(u string)
	dfs = func(u string) {
		visited[u] = true
		recStack[u] = true
		path = append(path, u)

		for _, v := range adj[u] {
			if !visited[v] {
				dfs(v)
			} else if recStack[v] {
				// Cycle detected: record the cycle path
				idx := -1
				for i, node := range path {
					if node == v {
						idx = i
						break
					}
				}
				if idx != -1 {
					cycle := make([]string, len(path[idx:]))
					copy(cycle, path[idx:])
					cycle = append(cycle, v)
					cycles = append(cycles, cycle)
				}
			}
		}

		path = path[:len(path)-1]
		recStack[u] = false
	}

	for _, node := range g.Nodes {
		if !visited[node.ID] {
			dfs(node.ID)
		}
	}

	return cycles
}

// GraphDiff captures semantic differences between two versions of an AgentGraph.
type GraphDiff struct {
	OldVersion        string   `json:"old_version"`
	NewVersion        string   `json:"new_version"`
	AddedNodes        []string `json:"added_nodes"`
	RemovedNodes      []string `json:"removed_nodes"`
	AddedTools        []string `json:"added_tools"`
	RemovedTools      []string `json:"removed_tools"`
	AddedDelegations  []string `json:"added_delegations"`
	RemovedDelegations []string `json:"removed_delegations"`
	AddedModels       []string `json:"added_models"`
	RemovedModels     []string `json:"removed_models"`
	HasBreakingChanges bool   `json:"has_breaking_changes"`
}

// DiffGraphs compares two versions of an AgentGraph.
func DiffGraphs(oldG, newG *AgentGraph) *GraphDiff {
	diff := &GraphDiff{
		OldVersion:   oldG.Version,
		NewVersion:   newG.Version,
		AddedNodes:   make([]string, 0),
		RemovedNodes: make([]string, 0),
		AddedTools:   make([]string, 0),
		RemovedTools: make([]string, 0),
	}

	oldNodes := make(map[string]bool)
	for _, n := range oldG.Nodes {
		oldNodes[n.ID] = true
	}
	newNodes := make(map[string]bool)
	for _, n := range newG.Nodes {
		newNodes[n.ID] = true
		if !oldNodes[n.ID] {
			diff.AddedNodes = append(diff.AddedNodes, n.ID)
		}
	}
	for id := range oldNodes {
		if !newNodes[id] {
			diff.RemovedNodes = append(diff.RemovedNodes, id)
			diff.HasBreakingChanges = true
		}
	}

	oldTools := stringSliceToMap(oldG.Tools)
	newTools := stringSliceToMap(newG.Tools)
	for t := range newTools {
		if !oldTools[t] {
			diff.AddedTools = append(diff.AddedTools, t)
		}
	}
	for t := range oldTools {
		if !newTools[t] {
			diff.RemovedTools = append(diff.RemovedTools, t)
			diff.HasBreakingChanges = true
		}
	}

	oldDelegates := stringSliceToMap(oldG.Delegations)
	newDelegates := stringSliceToMap(newG.Delegations)
	for d := range newDelegates {
		if !oldDelegates[d] {
			diff.AddedDelegations = append(diff.AddedDelegations, d)
		}
	}
	for d := range oldDelegates {
		if !newDelegates[d] {
			diff.RemovedDelegations = append(diff.RemovedDelegations, d)
		}
	}

	return diff
}

func stringSliceToMap(slice []string) map[string]bool {
	m := make(map[string]bool, len(slice))
	for _, s := range slice {
		m[s] = true
	}
	return m
}
