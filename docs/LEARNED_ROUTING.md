# Learned Routing Engine & Model Registry

AgentMesh incorporates learned routing models to continuously optimize candidate selection as real operational outcomes compound.

## The Entry Gate (`DISABLED_INSUFFICIENT_DATA`)

Learned routing models are strictly gated. A tenant router remains in `DISABLED_INSUFFICIENT_DATA` until:

- At least 50 verified `RoutingOutcome` records are logged.
- At least 2 distinct agents have participated in production tasks.
- At least 1 capability has observed performance evidence.

When thresholds are not met, the system automatically falls back to `BaselineRouterV1` deterministic routing.

## Pure Go Online Inference

AgentMesh serves learned model scoring natively in Go, avoiding Python sidecars or foreign runtimes in the hot routing path. Features include historical success rate, latency distribution, cost per task, and task complexity class.

## Uncertainty & Cold Start

Candidates without sufficient historical data are flagged as `COLD_START` or `LOW_EVIDENCE`. Safe exploration (1–5% of non-critical traffic) is allowed only for non-destructive tools.

## Model Lifecycle & Rollback

Models advance through:

1. `TRAINING`: Offline training on tenant-isolated outcomes.
2. `CANDIDATE`: Evaluated in offline route replay.
3. `SHADOW`: Evaluates live production tasks without live route impact.
4. `ACTIVE`: Promoted via human approval.
5. `RETIRED`: Retained for historical audit.

A single rollback action instantly restores the last known good active router model.
