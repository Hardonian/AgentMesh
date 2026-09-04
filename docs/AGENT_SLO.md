# Agent Service Level Objectives (SLOs)

AgentMesh provides continuous evaluation of Service Level Objectives (SLOs) and error budgets across agents and capabilities.

## SLO Targets
- `TargetSuccessRate`: Required success percentage (e.g. 99.0%).
- `MaxP95LatencyMs`: Maximum acceptable P95 latency (e.g. 4000ms).
- `MaxCostUSD`: Maximum allowable average cost per task.
- `TargetToolSuccess`: Downstream MCP tool call reliability target (e.g. 99.5%).

## SLO Statuses
- `HEALTHY`: Performance satisfies all targets with ample error budget.
- `AT_RISK`: Performance is within targets, but remaining error budget is below 20%.
- `BREACHED`: Observed performance violates one or more configured targets.
- `UNKNOWN`: Insufficient sample volume (< 5 tasks).

## Error Budget Tracking
Error budget is tracked as:
$$\text{RemainingBudget} = 1.0 - \frac{\text{ActualErrorRate}}{\text{AllowedErrorRate}}$$

When an agent's error budget is exhausted, its composite routing score is penalized, naturally shifting traffic to healthier candidates while allowing the breached agent to recover safely.
