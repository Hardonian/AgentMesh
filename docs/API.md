# Control Plane REST API Reference

All APIs are versioned under `/api/v1` and support multi-tenant isolation via the `X-Tenant-ID` header.

## Endpoints

### Agents

- `POST /api/v1/agents`: Register or update an AgentContract.
- `GET /api/v1/agents`: List registered agents.
- `GET /api/v1/agents/{id}`: Inspect agent details, contract, and Agent Passport evidence.
- `DELETE /api/v1/agents/{id}`: Deregister an agent.

### Policies

- `POST /api/v1/policy/evaluate`: Evaluate an authorization request (`ALLOW`, `DENY`, `REQUIRE_APPROVAL`).
- `GET /api/v1/policies`: List policies.
- `POST /api/v1/policies`: Create or update a declarative policy.

### Routing

- `POST /api/v1/routing/route`: Determine the optimal agent for a task requirement.
- `POST /api/v1/routing/explain`: Detailed routing candidate explanation and score breakdown.

### Approvals

- `GET /api/v1/approvals`: List pending approval requests.
- `POST /api/v1/approvals/{id}/resolve`: Approve or reject a sensitive tool execution.

### Progressive Delivery (Canaries)

- `GET /api/v1/canaries`: List active canary rollouts.
- `POST /api/v1/canaries`: Launch a new canary rollout.
- `POST /api/v1/canaries/{agentId}/promote`: Increase traffic weight or promote to active.
- `POST /api/v1/canaries/{agentId}/rollback`: Emergency rollback to baseline version.

### Telemetry & Audit

- `GET /api/v1/traces`: List recent execution traces.
- `GET /api/v1/traces/{id}`: Fetch waterfall trace.
- `GET /api/v1/audit`: Query append-only audit trail.

### Configuration

- `GET /api/v1/config/bundle`: Fetch Ed25519 cryptographically signed configuration bundle for proxies.
