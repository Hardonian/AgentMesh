# Canary Engine V3

## Multi-Target Progressive Delivery

Canary V3 provides unified progressive delivery across five dimensions:

- `AGENT_VERSION`: Roll out new agent container image or ADK graph revision.
- `MODEL_TARGET`: Transition between Gemini model targets (e.g., Flash $\to$ Pro).
- `ROUTE_POLICY`: Test new capability routing objectives or constraints.
- `TOOL_PROVIDER`: Validate tool server substitutions.
- `ROUTER_ALGORITHM`: Safely canary new learned routing models.

## Progressive Stages

Default rollout progression:

- **Stage 1 (1%)**: Sanity verification, 10 requests minimum.
- **Stage 2 (5%)**: Initial canary sample, 25 requests minimum.
- **Stage 3 (10%)**: Statistical trend analysis, 50 requests minimum.
- **Stage 4 (25%)**: Load verification, 100 requests minimum.
- **Stage 5 (50%)**: Pre-promotion balance, 200 requests minimum.
- **Stage 6 (100%)**: Full active promotion, marked as `LastKnownGoodRoute`.

## Automated Rollback Triggers

Rollback is immediately triggered if:

- Policy violation count exceeds 0.
- Error rate spikes above allowed threshold.
- P95 latency exceeds allowed budget.
- Fallback rate surges unexpectedly.
