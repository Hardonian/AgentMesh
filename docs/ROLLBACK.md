# Deterministic Rollback & Last Known Good Routes

## Last Known Good Route Qualification

A route only qualifies as a `LastKnownGoodRoute` when:

1. It has completed the full canary progression to 100% traffic.
2. It has been observed in production for a minimum stabilization window ($\ge 1\text{ hour}$).
3. Success rate meets or exceeds the capability SLO target ($\ge 99\%$).
4. Zero security policy denials or unauthorized tool attempts were recorded.

## Single-Action Rollback

Every `AgentOptimizationAction` generates a deterministic `RollbackPlan`. When triggered:

1. The reconciler halts forward traffic progression.
2. The proxy mutator restores the prior signed configuration.
3. The operational graph records a `ROLLED_BACK` outcome with the causal metric trigger.
