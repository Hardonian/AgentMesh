# Distributed Reliability Network

The AgentMesh Reliability Network computes statistical profiles and operational health for every registered agent, version, and capability.

## Statistical Rolling Windows
Rather than single static scores, AgentMesh tracks performance across 4 rolling windows:
- **1-minute window (`w1m`)**: Immediate failure spikes and rate-limiting.
- **5-minute window (`w5m`)**: Short-term operational trends and incident detection.
- **1-hour window (`w1h`)**: Medium-term reliability baseline.
- **24-hour window (`w24h`)**: Long-term SLO compliance.

## Reliability Profile Metrics
- `OverallSuccessRate`: Empirical task success percentage.
- `P50LatencyMs`, `P95LatencyMs`, `P99LatencyMs`: Latency distribution.
- `AverageCostUSD` / `MaxObservedCostUSD`: Economic expenditure.
- `ToolCallSuccessRate`: Success of downstream MCP tool interactions.
- `TimeoutRate`: Rate of deadline expirations.
- `Confidence`: Categorized as `HIGH_EVIDENCE`, `MEDIUM_EVIDENCE`, `LOW_EVIDENCE`, or `COLD_START`.

## Incident Routing Mode
When the 5-minute rolling window observes severe failure spikes (>50% errors with >= 5 samples), AgentMesh activates `INCIDENT` mode:
1. Disables the failing candidate from routing consideration.
2. Pins traffic to verified healthy fallback agents.
3. Halts safe exploration.
4. Emits a signed `agent.degraded` webhook.
