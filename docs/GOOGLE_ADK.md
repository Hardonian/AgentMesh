# Google Agent Developer Kit (ADK) Go Integration

## Overview
AgentMesh provides first-class support for Go agents built using the Google Agent Developer Kit (ADK).

## Graph Awareness
AgentMesh inspects ADK workflow graph topologies:
- Identifies workflow nodes (Agents, Tools, Decisions, Delegations, Approvals).
- Detects attached MCP tools and external service dependencies.
- Synthesizes canonical `AgentContract` specifications directly from the Go code topology.

## Example Usage
```go
import (
    "github.com/agentmesh/agentmesh/internal/adk"
    "github.com/agentmesh/agentmesh/pkg/sdk"
)

// 1. Build an ADK workflow graph
graph := adk.NewGraph("market-agent", "Market Insights", "1.0.0")
graph.AddNode("n1", "Fetch Data", adk.NodeTypeTool, "bigquery.read", "SQL query")
graph.AddNode("n2", "Summarize", adk.NodeTypeDelegation, "summarizer-agent", "Delegated summary")

// 2. Synthesize contract
contract := graph.SynthesizeContract()

// 3. Register with AgentMesh control plane
client := sdk.NewClient("http://127.0.0.1:8080", "mesh_api_key")
resp, err := client.RegisterAgent(ctx, contract)
```
