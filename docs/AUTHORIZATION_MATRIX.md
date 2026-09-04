# Role-Based Access Control (RBAC) & Authorization Matrix — AgentMesh

## 1. Overview

AgentMesh enforces strict, fail-closed Role-Based Access Control (RBAC) across all Control Plane HTTP endpoints, MCP intelligence tools, and data-plane operations.

Authentication is established via cryptographic API keys (`mesh_...`), mTLS peer identities, or JWT bearer tokens. Every authenticated entity is assigned one or more roles within a specific `tenant_id`.

## 2. Roles & Hierarchy

| Role | Hierarchy Level | Description |
| --- | --- | --- |
| **OWNER** | 6 (Highest) | Full administrative control of tenant, tenant settings, billing, key management, and member administration. |
| **ADMIN** | 5 | Configuration management, policy authoring, emergency freeze/unfreeze, agent lifecycle, and role assignment. |
| **OPERATOR** | 4 | Deployment orchestration, canary promotion/abort, routing configuration, and operational action execution. |
| **APPROVER** | 3 | Human-In-The-Loop (HITL) authorization for destructive/write actions and high-risk tool execution. |
| **DEVELOPER** | 2 | Agent contract authoring, policy simulation, test execution, benchmark evaluation, and read access to telemetry. |
| **VIEWER** | 1 (Lowest) | Read-only inspection of agents, passports, public metrics, and routing status. Cannot mutate any state. |

---

## 3. Control Plane HTTP Endpoint Authorization Matrix

| Endpoint | Method | Required Role | OWNER | ADMIN | OPERATOR | APPROVER | DEVELOPER | VIEWER |
| --- | --- | --- | :---: | :---: | :---: | :---: | :---: | :---: |
| `/healthz`, `/readyz` | GET | *Public / Any* | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/api/v1/agents` | GET | `VIEWER` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/api/v1/agents` | POST | `DEVELOPER` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `/api/v1/agents/:id` | GET | `VIEWER` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/api/v1/agents/:id` | PUT | `DEVELOPER` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `/api/v1/agents/:id` | DELETE | `ADMIN` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `/api/v1/passports/:id` | GET | `VIEWER` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/api/v1/passports/:id/public` | GET | *Unauthenticated* | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/api/v1/passports` | POST | `DEVELOPER` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `/api/v1/contracts/validate` | POST | `DEVELOPER` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `/api/v1/contracts/diff` | POST | `DEVELOPER` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `/api/v1/policies` | GET | `VIEWER` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/api/v1/policies` | POST | `ADMIN` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `/api/v1/policies/simulate` | POST | `DEVELOPER` | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| `/api/v1/policies/canary` | POST | `OPERATOR` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `/api/v1/routes` | GET | `VIEWER` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/api/v1/routes/simulate` | POST | `DEVELOPER` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| `/api/v1/canary/promote` | POST | `OPERATOR` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `/api/v1/canary/abort` | POST | `OPERATOR` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `/api/v1/control/actions` | GET | `VIEWER` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/api/v1/control/actions/:id/approve` | POST | `APPROVER` | ✅ | ✅ | ❌ | ✅ | ❌ | ❌ |
| `/api/v1/control/actions/:id/reject` | POST | `APPROVER` | ✅ | ✅ | ❌ | ✅ | ❌ | ❌ |
| `/api/v1/control/actions/:id/execute` | POST | `OPERATOR` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `/api/v1/control/freeze` | POST | `OPERATOR` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `/api/v1/credentials` | GET | `ADMIN` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `/api/v1/credentials` | POST | `ADMIN` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `/api/v1/credentials/:id` | DELETE | `ADMIN` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `/api/v1/audit/logs` | GET | `VIEWER` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/api/v1/telemetry/spans` | GET | `VIEWER` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `/api/v1/tenants/rotate-keys` | POST | `OWNER` | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

---

## 4. MCP Tools Authorization Matrix

| MCP Tool Name | Tool Risk Class | Minimum Role | Read-Only Safe | Requires HITL Token | Description |
| --- | --- | --- | :---: | :---: | --- |
| `inspect_agent` | `READ` | `VIEWER` | ✅ | ❌ | Inspects registered agent contract, identity, and lifecycle state. |
| `get_agent_passport` | `READ` | `VIEWER` | ✅ | ❌ | Retrieves Agent Passport V2 with reliability, evidence, and provenance. |
| `inspect_graph` | `READ` | `VIEWER` | ✅ | ❌ | Retrieves canonical AgentGraph topology, tools, and risk findings. |
| `explain_policy` | `READ` | `DEVELOPER` | ✅ | ❌ | Evaluates deterministic policy and explains ALLOW/DENY decision. |
| `simulate_route` | `READ` | `DEVELOPER` | ✅ | ❌ | Simulates candidate agent ranking and returns route confidence score. |
| `get_evaluation` | `READ` | `DEVELOPER` | ✅ | ❌ | Retrieves benchmark evaluation scorecard and regression baselines. |
| `compare_versions` | `READ` | `DEVELOPER` | ✅ | ❌ | Analyzes contract drift and flags security-sensitive differences. |
| `get_tool_risk` | `READ` | `VIEWER` | ✅ | ❌ | Returns risk classification and schema drift status for a tool. |
| `request_approval` | `WRITE` | `DEVELOPER` | ❌ | ❌ | Creates a pending human approval request for a sensitive action. |
| `validate_approval` | `READ` | `OPERATOR` | ✅ | ❌ | Validates a single-use approval token against bound parameters. |
| `execute_tool_action` | `DESTRUCTIVE` | `OPERATOR` | ❌ | ✅ | Executes high-risk tool action; rejected unless valid HITL token provided. |

---

## 5. Security Invariants & Enforcement Mechanism

1. **Fail-Closed Default**: If an incoming request lacks valid authentication or fails role evaluation, it is immediately denied with HTTP 401 Unauthorized or HTTP 403 Forbidden.
2. **Tenant Scoping**: Role evaluation is strictly scoped within the caller's tenant. An `OWNER` of Tenant A has zero access or authority over Tenant B.
3. **No Role Stacking for Approvals**: For segregation of duties, the requester of an action cannot approve their own action (`reviewerID != requesterID`).
4. **Separation of Approval & Execution**: An `APPROVER` can grant an approval token, but only an `OPERATOR`, `ADMIN`, or `OWNER` can dispatch the action to the execution pipeline.
