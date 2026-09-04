# Production Outcomes & Verified Economic Benefits

## Projected vs. Verified Improvement

AgentMesh maintains a strict boundary between projected and verified metrics:

- **Projected Savings**: Estimates computed during optimization scheduling based on historical benchmarks.
- **Verified Savings**: Empirical deltas calculated from equal-duration observation windows before and after a change, normalized across task complexity and request volume.

$$\text{CostDeltaUSD} = \text{CostPerTask}_{\text{after}} - \text{CostPerTask}_{\text{before}}$$
$$\text{LatencyDeltaMs} = \text{P95Latency}_{\text{after}} - \text{P95Latency}_{\text{before}}$$

## Operational Outcome Graph Integration

Production outcomes are recorded as first-class nodes in the Operational Outcome Graph, connecting:
`Action` $\to$ `CanaryRun` $\to$ `Promotion` $\to$ `ProductionOutcome` $\to$ `ImprovementStatus` (`IMPROVED`, `REGRESSED`, `NEUTRAL`).
