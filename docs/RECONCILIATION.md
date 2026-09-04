# Reconciliation Engine & Desired-State Architecture

AgentMesh reconciles declarative desired state against observed runtime state:

1. **Routing State**: `AgentRoutingSpec` (desired weights, fallbacks, SLOs) vs. active proxy routes.
2. **Deployment State**: `AgentDeploymentSpec` (desired models, replicas, tools) vs. GKE/Cloud Run running services.

## Safe Progressive Execution

Reconciliation plans generate safe, staged execution steps rather than issuing abrupt switches:
1. `validate_policy_eligibility`
2. `shadow_traffic_evaluation`
3. `canary_initial_stage_5pct`
4. `canary_interim_stage_25pct`
5. `promote_final_weight_100pct`
6. `observe_and_verify_stability`

Every step maintains an explicit compensation/rollback action.
