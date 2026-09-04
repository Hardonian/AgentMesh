# Routing Intelligence V3

AgentMesh routes requests to the optimal agent based on empirical operational evidence, policy, and system health.

## BaselineRouterV1: Deterministic 9-Step Pipeline

1. **Policy Eligibility (Step 1)**: Deterministic policy evaluation. Explicit `DENY` rules immediately disqualify candidates.
2. **Capability Evidence Tier**:
   - `PRODUCTION_OBSERVED_CAPABILITY` (Score 1.0)
   - `EVALUATED_CAPABILITY` (Score 0.8)
   - `DECLARED_CAPABILITY` (Score 0.5)
3. **Current Health**: Excludes `UNHEALTHY` agents or those under an active `INCIDENT`.
4. **Tool Requirement Match**: Verifies all required tools are authorized and available.
5. **Region & Data Residency**: Rejects cross-region transfers for `RESTRICTED` or `SOVEREIGN` data.
6. **SLO Compliance & Error Budget**: Penalizes agents with `BREACHED` or `AT_RISK` SLOs.
7. **Empirical Reliability**: Ingests rolling 1m/5m/1h/24h statistical success rates.
8. **Cost & Latency Efficiency**: Evaluates P95 latency and average cost per successful task.
9. **Deterministic Tie-Break**: Predictable, stable tie-breaking across equal candidates.

## Multi-Objective Modes

- `RELIABILITY`: 50% reliability, 20% SLO, 15% quality, 10% latency, 5% cost.
- `QUALITY`: 50% quality, 20% reliability, 15% SLO, 10% latency, 5% cost.
- `LATENCY`: 50% latency, 20% reliability, 15% SLO, 10% quality, 5% cost.
- `COST`: 50% cost, 20% reliability, 15% SLO, 10% quality, 5% latency.
- `POLICY_FIRST`: 40% policy score, 30% SLO, 15% reliability, 15% quality.
- `BALANCED`: 30% reliability, 25% quality, 20% latency, 15% cost, 10% SLO.

## Routing Hysteresis

Prevents rapid route flapping by enforcing a minimum improvement delta (e.g. 5% score lift) before displacing a healthy incumbent agent.

## Failover Routing

If the preferred agent experiences a circuit breaker trip, timeout, or failure, AgentMesh automatically selects the next highest-scoring eligible candidate and records the failover outcome.
