# Offline Routing Replay & Regret Analysis

AgentMesh enables operators to evaluate candidate routing models against historical traffic corpora before promoting them to production.

## CLI Workflow
```bash
agentmesh route replay [historyFile]
```

## Comparative Metrics
- **Decision Agreement Rate**: Percentage of historical tasks where the candidate model selected the same agent.
- **Estimated Cost Delta**: Total projected USD expenditure change across the task corpus.
- **Estimated Latency Delta**: Projected average task execution speedup or slowdown.
- **Regret Metrics**:
  - `COST_REGRET`: Cost difference when an eligible candidate was cheaper than the selected agent.
  - `LATENCY_REGRET`: Latency difference when an eligible candidate was faster.
  - `QUALITY_REGRET`: Quality difference based on benchmark scorecards.

## Counterfactual Boundaries
When evaluating counterfactual choices (an agent that was eligible but not executed), AgentMesh marks outcomes as:
- `OBSERVED`: Empirical production outcome on that task.
- `ESTIMATED`: Projected from the candidate's statistical reliability profile.
- `UNKNOWN`: Insufficient evidence to model counterfactual behavior.
