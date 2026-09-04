package outcome

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// NodeType identifies operational entities in the outcome graph.
type NodeType string

const (
	NodeAgent             NodeType = "Agent"
	NodeAgentVersion      NodeType = "AgentVersion"
	NodeCapability        NodeType = "Capability"
	NodeTool              NodeType = "Tool"
	NodeToolVersion       NodeType = "ToolVersion"
	NodeModel             NodeType = "Model"
	NodeModelVersion      NodeType = "ModelVersion"
	NodeAgentGraph        NodeType = "AgentGraph"
	NodePolicyVersion     NodeType = "PolicyVersion"
	NodeRouteDecision     NodeType = "RouteDecision"
	NodeInvocation        NodeType = "Invocation"
	NodeEvaluation        NodeType = "Evaluation"
	NodeCanary            NodeType = "Canary"
	NodeDelegation        NodeType = "Delegation"
	NodeProductionOutcome NodeType = "ProductionOutcome"
	NodeFailure           NodeType = "Failure"
	NodeDeployment        NodeType = "Deployment"
	NodeRegion            NodeType = "Region"
)

// EdgeType identifies verified relationships in the operational graph.
type EdgeType string

const (
	EdgeAgentSupportsCapability       EdgeType = "AGENT_SUPPORTS_CAPABILITY"
	EdgeAgentInvokedTool              EdgeType = "AGENT_INVOKED_TOOL"
	EdgeAgentDelegatedToAgent         EdgeType = "AGENT_DELEGATED_TO_AGENT"
	EdgeVersionEvaluatedForCapability EdgeType = "VERSION_EVALUATED_FOR_CAPABILITY"
	EdgeRouteSelectedAgent            EdgeType = "ROUTE_SELECTED_AGENT"
	EdgeRouteRejectedAgent            EdgeType = "ROUTE_REJECTED_AGENT"
	EdgeInvocationSucceeded           EdgeType = "INVOCATION_SUCCEEDED"
	EdgeInvocationFailed              EdgeType = "INVOCATION_FAILED"
	EdgeCanaryPromoted                EdgeType = "CANARY_PROMOTED"
	EdgeCanaryRolledBack              EdgeType = "CANARY_ROLLED_BACK"
	EdgePolicyDenied                  EdgeType = "POLICY_DENIED"
	EdgeToolSchemaChanged             EdgeType = "TOOL_SCHEMA_CHANGED"
	EdgeModelFallbackOccurred         EdgeType = "MODEL_FALLBACK_OCCURRED"
)

// GraphNode represents an operational entity.
type GraphNode struct {
	ID         string            `json:"id"`
	Type       NodeType          `json:"type"`
	TenantID   string            `json:"tenantId"`
	Properties map[string]any    `json:"properties,omitempty"`
	CreatedAt  time.Time         `json:"createdAt"`
}

// GraphEdge represents a directed operational link.
type GraphEdge struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenantId"`
	Type       EdgeType       `json:"type"`
	FromID     string         `json:"fromId"`
	ToID       string         `json:"toId"`
	Properties map[string]any `json:"properties,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
}

// LikelySource classifies root-cause probabilities.
type LikelySource string

const (
	SourceAgentCode       LikelySource = "AGENT_CODE"
	SourceToolTimeout     LikelySource = "TOOL_TIMEOUT"
	SourceToolSchema      LikelySource = "TOOL_SCHEMA"
	SourceModelError      LikelySource = "MODEL_ERROR"
	SourcePolicyDenial    LikelySource = "POLICY_DENIAL"
	SourceDelegationError LikelySource = "DELEGATION_ERROR"
	SourceApprovalTimeout LikelySource = "APPROVAL_TIMEOUT"
	SourceUnknown         LikelySource = "UNKNOWN"
)

// RootCauseAnalysis reports automated inference on failure origin.
type RootCauseAnalysis struct {
	OutcomeID     string       `json:"outcomeId"`
	LikelySource  LikelySource `json:"likelySource"`
	Confidence    float64      `json:"confidence"`
	Explanation   string       `json:"explanation"`
	FailingEntity string       `json:"failingEntity"`
	TracePath     []string     `json:"tracePath"`
}

// BottleneckReport identifies dominant contributors to latency and cost.
type BottleneckReport struct {
	DominantLatencySource string             `json:"dominantLatencySource"`
	DominantCostSource    string             `json:"dominantCostSource"`
	ComponentBreakdown    map[string]float64 `json:"componentBreakdown"` // e.g. "tool:bigquery": 1200ms
	TotalDurationMs       int64              `json:"totalDurationMs"`
	Recommendations       []string           `json:"recommendations"`
}

// OperationalOutcomeGraph manages in-memory/relational graph queries.
type OperationalOutcomeGraph struct {
	mu       sync.RWMutex
	nodes    map[string]*GraphNode           // tenant:id -> Node
	edges    map[string]*GraphEdge           // tenant:id -> Edge
	adjOut   map[string]map[string]*GraphEdge // tenant:fromId -> toId -> Edge
	adjIn    map[string]map[string]*GraphEdge // tenant:toId -> fromId -> Edge
}

// NewOperationalOutcomeGraph creates an operational outcome graph.
func NewOperationalOutcomeGraph() *OperationalOutcomeGraph {
	return &OperationalOutcomeGraph{
		nodes:  make(map[string]*GraphNode),
		edges:  make(map[string]*GraphEdge),
		adjOut: make(map[string]map[string]*GraphEdge),
		adjIn:  make(map[string]map[string]*GraphEdge),
	}
}

func nodeKey(tenantID, id string) string {
	return tenantID + ":" + id
}

// AddNode adds an entity to the operational outcome graph.
func (g *OperationalOutcomeGraph) AddNode(node *GraphNode) error {
	if node == nil || node.ID == "" || node.TenantID == "" {
		return errors.New("invalid node")
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	k := nodeKey(node.TenantID, node.ID)
	if node.CreatedAt.IsZero() {
		node.CreatedAt = time.Now().UTC()
	}
	g.nodes[k] = node
	return nil
}

// AddEdge adds a directed operational relationship.
func (g *OperationalOutcomeGraph) AddEdge(edge *GraphEdge) error {
	if edge == nil || edge.ID == "" || edge.FromID == "" || edge.ToID == "" || edge.TenantID == "" {
		return errors.New("invalid edge")
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	k := nodeKey(edge.TenantID, edge.ID)
	if edge.CreatedAt.IsZero() {
		edge.CreatedAt = time.Now().UTC()
	}
	g.edges[k] = edge

	fromKey := nodeKey(edge.TenantID, edge.FromID)
	toKey := nodeKey(edge.TenantID, edge.ToID)

	if g.adjOut[fromKey] == nil {
		g.adjOut[fromKey] = make(map[string]*GraphEdge)
	}
	g.adjOut[fromKey][edge.ToID] = edge

	if g.adjIn[toKey] == nil {
		g.adjIn[toKey] = make(map[string]*GraphEdge)
	}
	g.adjIn[toKey][edge.FromID] = edge

	return nil
}

// GetNode returns an operational node.
func (g *OperationalOutcomeGraph) GetNode(tenantID, id string) (*GraphNode, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	n, ok := g.nodes[nodeKey(tenantID, id)]
	return n, ok
}

// GetOutEdges returns directed outgoing edges from a node.
func (g *OperationalOutcomeGraph) GetOutEdges(tenantID, fromID string) []*GraphEdge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	m, ok := g.adjOut[nodeKey(tenantID, fromID)]
	if !ok {
		return nil
	}
	list := make([]*GraphEdge, 0, len(m))
	for _, e := range m {
		list = append(list, e)
	}
	return list
}

// AnalyzeRootCause infers the probable failure point along an execution chain.
func (g *OperationalOutcomeGraph) AnalyzeRootCause(tenantID, invocationID string) *RootCauseAnalysis {
	g.mu.RLock()
	defer g.mu.RUnlock()

	res := &RootCauseAnalysis{
		OutcomeID:    invocationID,
		LikelySource: SourceUnknown,
		Confidence:   0.50,
		TracePath:    []string{invocationID},
	}

	invNode, ok := g.nodes[nodeKey(tenantID, invocationID)]
	if !ok {
		res.Explanation = "Invocation record not found in operational graph"
		return res
	}

	edges := g.adjOut[nodeKey(tenantID, invocationID)]
	for toID, edge := range edges {
		res.TracePath = append(res.TracePath, toID)
		switch edge.Type {
		case EdgePolicyDenied:
			res.LikelySource = SourcePolicyDenial
			res.Confidence = 0.95
			res.FailingEntity = toID
			res.Explanation = fmt.Sprintf("Request was blocked by security policy %s", toID)
			return res
		case EdgeInvocationFailed:
			// Check if target is a tool or delegation
			targetNode, targetOk := g.nodes[nodeKey(tenantID, toID)]
			if targetOk {
				switch targetNode.Type {
				case NodeTool, NodeToolVersion:
					res.LikelySource = SourceToolTimeout
					res.Confidence = 0.85
					res.FailingEntity = toID
					res.Explanation = fmt.Sprintf("MCP Tool invocation failed on %s", toID)
					return res
				case NodeDelegation, NodeAgent:
					res.LikelySource = SourceDelegationError
					res.Confidence = 0.80
					res.FailingEntity = toID
					res.Explanation = fmt.Sprintf("Delegated sub-agent execution failed on %s", toID)
					return res
				case NodeModel:
					res.LikelySource = SourceModelError
					res.Confidence = 0.90
					res.FailingEntity = toID
					res.Explanation = fmt.Sprintf("Downstream model error on %s", toID)
					return res
				}
			}
		}
	}

	// Fallback to properties if attached
	if failType, ok := invNode.Properties["failure_type"].(string); ok && failType != "" {
		switch failType {
		case "TIMEOUT":
			res.LikelySource = SourceToolTimeout
			res.Confidence = 0.75
			res.Explanation = "Execution exceeded configured deadline"
		case "AGENT_ERROR":
			res.LikelySource = SourceAgentCode
			res.Confidence = 0.75
			res.Explanation = "Agent runtime encountered unhandled internal error"
		case "POLICY_DENIED":
			res.LikelySource = SourcePolicyDenial
			res.Confidence = 0.95
			res.Explanation = "Execution prohibited by deterministic policy"
		}
	}

	return res
}

// AnalyzeBottlenecks calculates latency and cost bottlenecks along an operational execution chain.
func (g *OperationalOutcomeGraph) AnalyzeBottlenecks(tenantID, invocationID string) *BottleneckReport {
	g.mu.RLock()
	defer g.mu.RUnlock()

	report := &BottleneckReport{
		ComponentBreakdown: make(map[string]float64),
		Recommendations:    make([]string, 0),
	}

	invNode, ok := g.nodes[nodeKey(tenantID, invocationID)]
	if !ok {
		return report
	}

	var maxLatency float64
	var maxLatencyComponent string

	if dur, ok := invNode.Properties["duration_ms"].(float64); ok {
		report.TotalDurationMs = int64(dur)
	}

	outEdges := g.adjOut[nodeKey(tenantID, invocationID)]
	for toID, edge := range outEdges {
		if lat, ok := edge.Properties["latency_ms"].(float64); ok {
			report.ComponentBreakdown[toID] = lat
			if lat > maxLatency {
				maxLatency = lat
				maxLatencyComponent = toID
			}
		}
	}

	report.DominantLatencySource = maxLatencyComponent
	if maxLatency > 3000 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("Component %s accounts for %.0fms; consider caching or parallelization", maxLatencyComponent, maxLatency))
	}
	if len(outEdges) > 4 {
		report.Recommendations = append(report.Recommendations,
			"High serial delegation depth detected; evaluate parallel branch execution")
	}

	sort.Strings(report.Recommendations)
	return report
}
