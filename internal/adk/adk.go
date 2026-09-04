package adk

import (
	"fmt"
	"time"

	"github.com/agentmesh/agentmesh/pkg/contracts"
)

// WorkflowNodeType represents a functional block in a Google ADK workflow graph.
type WorkflowNodeType string

const (
	NodeTypeAgent         WorkflowNodeType = "AGENT"
	NodeTypeTool          WorkflowNodeType = "TOOL"
	NodeTypeDecision      WorkflowNodeType = "DECISION"
	NodeTypeDelegation    WorkflowNodeType = "DELEGATION"
	NodeTypeHumanApproval WorkflowNodeType = "HUMAN_APPROVAL"
)

// WorkflowNode is a graph node in an ADK multi-agent workflow.
type WorkflowNode struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Type        WorkflowNodeType `json:"type"`
	Target      string           `json:"target,omitempty"` // Agent ID or Tool Name
	Description string           `json:"description,omitempty"`
}

// WorkflowEdge connects two workflow nodes.
type WorkflowEdge struct {
	FromID    string `json:"fromId"`
	ToID      string `json:"toId"`
	Condition string `json:"condition,omitempty"`
}

// Graph represents the topological execution graph of a Google ADK Go agent application.
type Graph struct {
	AgentID     string         `json:"agentId"`
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Nodes       []WorkflowNode `json:"nodes"`
	Edges       []WorkflowEdge `json:"edges"`
	InspectedAt time.Time      `json:"inspectedAt"`
}

// NewGraph creates an ADK workflow graph.
func NewGraph(agentID, name, version string) *Graph {
	return &Graph{
		AgentID:     agentID,
		Name:        name,
		Version:     version,
		InspectedAt: time.Now().UTC(),
	}
}

// AddNode registers a step in the ADK workflow.
func (g *Graph) AddNode(id, name string, nodeType WorkflowNodeType, target, description string) {
	g.Nodes = append(g.Nodes, WorkflowNode{
		ID:          id,
		Name:        name,
		Type:        nodeType,
		Target:      target,
		Description: description,
	})
}

// AddEdge connects two nodes in the workflow.
func (g *Graph) AddEdge(fromID, toID, condition string) {
	g.Edges = append(g.Edges, WorkflowEdge{
		FromID:    fromID,
		ToID:      toID,
		Condition: condition,
	})
}

// SynthesizeContract extracts tools and delegations from the ADK graph into an AgentContract.
func (g *Graph) SynthesizeContract() *contracts.AgentContract {
	c := &contracts.AgentContract{
		APIVersion: contracts.ExpectedAPIVersion,
		Kind:       contracts.ExpectedKind,
		Metadata: contracts.Metadata{
			Name:        g.AgentID,
			Version:     g.Version,
			Description: fmt.Sprintf("Auto-synthesized contract for Google ADK workflow %s", g.Name),
		},
		Identity: contracts.IdentityConfig{
			Protocols: []string{"a2a", "mcp"},
		},
	}

	toolSet := make(map[string]bool)
	delegateSet := make(map[string]bool)

	for _, node := range g.Nodes {
		switch node.Type {
		case NodeTypeTool:
			if node.Target != "" {
				toolSet[node.Target] = true
			}
		case NodeTypeDelegation:
			if node.Target != "" {
				delegateSet[node.Target] = true
			}
		case NodeTypeHumanApproval:
			c.Approval.RequiredFor = append(c.Approval.RequiredFor, node.Target)
		}
	}

	for t := range toolSet {
		c.Tools.Allow = append(c.Tools.Allow, t)
	}
	for d := range delegateSet {
		c.Delegation.Allow = append(c.Delegation.Allow, d)
	}

	return c
}
