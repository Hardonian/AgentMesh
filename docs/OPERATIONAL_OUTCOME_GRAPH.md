# Operational Outcome Graph

The AgentMesh Operational Outcome Graph transforms raw distributed agent executions into an auditable knowledge graph of operational intelligence.

## Entity Model (Nodes)

The graph connects:

- **Agent & AgentVersion**: Executing identities
- **Capability**: Declared, evaluated, and observed task domains
- **Tool & ToolVersion**: MCP tool invocations and schema states
- **Model & ModelVersion**: Gemini / Vertex AI model revisions
- **PolicyVersion**: Deterministic rules enforced during the execution
- **RouteDecision & Invocation**: Primary vs failover paths
- **Evaluation & Canary**: Pre-flight baselines and traffic splits
- **Delegation**: A2A delegation chains
- **ProductionOutcome**: Verified success, failure type, latency, and cost
- **Region & Deployment**: Geographic runtime location (e.g. `us-central1`, GKE, Cloud Run)

## Relational Graph Querying

Rather than introducing heavy Neo4j dependencies, AgentMesh uses PostgreSQL relational tables (`operational_outcome_nodes` and `operational_outcome_edges`) with indexed adjacency lookups in Go (`internal/outcome/graph.go`).

## Automated Root-Cause Inference

Given a failed invocation, AgentMesh traverses the outbound execution edges to identify the `LikelySource`:

1. `POLICY_DENIAL`: Deterministic rule blocked the call.
2. `TOOL_TIMEOUT`: MCP tool exceeded deadline.
3. `TOOL_SCHEMA`: Input/output failed JSON schema validation.
4. `DELEGATION_ERROR`: Downstream A2A peer agent failed.
5. `MODEL_ERROR`: Vertex AI provider returned quota exhaustion or error.
6. `AGENT_CODE`: Unhandled runtime panic in agent logic.

## Bottleneck Analysis

Aggregates serial call latency across the delegation chain and surfaces dominant latency contributors along with deterministic concurrency recommendations.
