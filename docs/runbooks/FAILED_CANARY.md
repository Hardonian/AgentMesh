# Runbook: Failed Canary Rollout

## Severity

Medium to High (depending on traffic weight)

## Symptoms

- Automated rollback triggered during a canary stage.
- Canary status reports `StateRolledBack` with non-empty `RollbackTrigger`.
- Latency, error rate, or policy violation thresholds exceeded.

## Automated Response

1. Canary Engine halts traffic progression immediately.
2. Proxy weight for candidate is shifted to 0%.
3. Baseline route receives 100% traffic.

## Operator Action

1. Inspect trigger reason: `agentmesh canary status [canary-id]`
2. Review structured error traces: `agentmesh route debug [task-id]`
3. Fix underlying agent container or prompt, and re-run offline evaluation before restarting canary.
