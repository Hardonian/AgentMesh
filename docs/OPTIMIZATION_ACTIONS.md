# Optimization Actions & Specifications

## AgentOptimizationAction Schema

AgentMesh models all proposed mutations as typed `AgentOptimizationAction` structures:

| Field | Type | Description |
| :--- | :--- | :--- |
| `action_id` | `string` | Unique action identifier |
| `organization_id` | `string` | Tenant isolation scope |
| `capability_id` | `string` | Target capability |
| `action_type` | `enum` | Type of mutation (`CHANGE_ROUTE_WEIGHT`, `CHANGE_MODEL_TARGET`, etc.) |
| `current_state` | `json` | Verified pre-change operational state |
| `proposed_state` | `json` | Target state |
| `risk_class` | `enum` | `LOW`, `MEDIUM`, `HIGH`, `CRITICAL` |
| `blast_radius` | `json` | Traffic %, affected agents, cost caps |
| `rollback_plan` | `json` | Deterministic rollback steps and last-known-good target |
| `approval_requirement` | `json` | Cryptographic hash binding and approver identity |

## Action Hashing

Approvals bind cryptographically to `ComputeActionHash()`:
$$\text{ActionHash} = \text{SHA256}(\text{Org} \parallel \text{Capability} \parallel \text{ActionType} \parallel \text{CurrentState} \parallel \text{ProposedState} \parallel \text{PolicyVersion})$$
If any parameter changes after approval is granted, the hash changes and execution is blocked (`ErrStaleApproval`).
