# Canonical AgentGraph Specification

The **AgentGraph** (`pkg/graph`) is the canonical, normalized representation of an AI agent's operational topology. It captures all workflow steps, tool invocations, sub-agent delegations, model dependencies, and human-in-the-loop approval boundaries.

---

## Schema Definition

```go
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
```

### Node Types

- `AGENT`: Direct invocation of another registered agent.
- `WORKFLOW_STEP`: Local logical execution step, router, or branch point.
- `TOOL`: Outbound tool invocation over MCP or direct API.
- `MODEL`: LLM invocation (Gemini 1.5 Pro, Flash, etc.).
- `HITL_APPROVAL`: Human-in-the-loop decision gate.

---

## Invariants & Validation Rules

1. **Single Entrypoint**: Every graph must declare exactly one valid `Entrypoint` referencing an existing node.
2. **Referential Integrity**: All edge `FromID` and `ToID` values must resolve to defined nodes within the same graph.
3. **Deterministic Canonical Hashing**: Nodes, edges, tools, and delegations are canonicalized and sorted deterministically before computing the SHA-256 digest (`Graph.Hash()`). Timestamp is zeroed during hashing to guarantee identical digests for identical topologies.
4. **Cycle Detection**: Unbounded cycles are detected via DFS traversal and rejected.

---

## Graph Diffing

`graph.DiffGraphs(g1, g2)` produces a structured `GraphDiff` detailing:

- Added/Removed nodes
- Added/Removed tools
- Added/Removed delegations
- Added/Removed models
- `HasBreakingChanges`: Set to true when tools or delegations are modified, triggering automated policy re-evaluation and canary gates.
