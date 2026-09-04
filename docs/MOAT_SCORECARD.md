# AgentMesh Operational Moat Scorecard

This scorecard tracks the accumulation of defensible operational evidence across the AgentMesh platform.

## Moat Telemetry Dimensions

| Telemetry Dimension | Tracking Objective | Status / Target |
| --- | --- | --- |
| **Routing Outcomes Recorded** | Cumulative volume of observed real-world executions | Production Indexing Active (`routing_outcomes_v3`) |
| **Capability Coverage** | Percentage of active organizational capabilities with empirical benchmarks | Minimum 1 baseline suite per capability |
| **Evaluation Density** | Ratio of continuous evaluation runs to declared agent contracts | Evaluated on every version/model change |
| **Statistical Windows** | Multi-timescale rolling performance tracking (1m, 5m, 1h, 24h) | Calculated per agent/capability pair |
| **Delegation Coverage** | Graph-mapped A2A delegation chains with observed success rates | Active in `operational_outcome_edges` |
| **Tool Reliability Network** | Empirical failure rate, P95 latency, and schema drift per MCP tool | Governed via Tool Passports |
| **Decision Agreement Rate** | Agreement between candidate routers and historical production choices | Measured via `agentmesh route replay` |
| **Verified Routing Improvements** | Measured post-promotion lift in cost reduction and latency speedup | Tracked in Router Passports |
| **Private Fleet Survivability** | Data-plane operational uptime during control plane outages | 24-hour signed configuration cache with fail-closed guarantee |
