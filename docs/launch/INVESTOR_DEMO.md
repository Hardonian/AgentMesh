# 5-Minute Technical Investor Demo Script — AgentMesh

## Objective

Demonstrate the long-term enterprise value, architectural defensibility, and operational data moat of AgentMesh in under five minutes.

---

## 00:00 – 01:00: The Enterprise Problem
>
> "Every enterprise is building AI agents. But when an agent can write SQL, initiate wire transfers, or delegate to external partner agents, traditional networking infrastructure is blind.
> An API gateway sees raw HTTP; it doesn't know what an agent identity is, what tools it's permitted to use, or whether a prompt injection is trying to escalate privileges.
> AgentMesh is the dedicated control plane and proxy for the agentic era."

---

## 01:00 – 02:00: The Product in Action (CLI Demo)

Execute:

```bash
agentmesh demo run
```

**Key Highlights to Point Out**:

1. **Deterministic Policy Gate**: Show `bigquery.read` -> `ALLOW`, but `payments.execute` -> `REQUIRE_APPROVAL`. Emphasize that authorization is deterministic code, not an LLM guess.
2. **Cryptographic Parameter Binding**: Show that human approval tokens bind to the SHA-256 hash of parameters. If an agent changes the account number or amount post-approval, the request fails mathematically.
3. **Capability Routing**: Show how AgentMesh selects the most reliable agent based on historical P95 latency and token cost.

---

## 02:00 – 03:30: The Web Control Plane & Progressive Delivery

Open Web Control Plane (`http://localhost:3000/dashboard`):

1. **Agent Operational Graph**: Show the live topology of agents, tools, and delegation paths. Point out cycle detection and taint tracking.
2. **Canary Deployments**: Show a 10% traffic split to a new agent version. Demonstrate that if the canary latency regresses by >20%, AgentMesh rolls back traffic to baseline automatically in under 60 seconds.
3. **Emergency Freeze Kill Switch**: Show the single-click mesh-wide freeze that halts all autonomous mutations during an incident.

---

## 03:30 – 05:00: The Data Moat & Enterprise Business Model

1. **The Operational Outcome Graph (The Moat)**:
   - Every time an agent routes, executes a tool, or finishes a task, AgentMesh records the cryptographic outcome fingerprint: cost, latency, reliability, tool failure rate, and provider stability.
   - This proprietary graph creates an compounding data asset that trains our routing intelligence to continuously lower cost and improve agent accuracy.
2. **Open-Core Commercial Model**:
   - **Open Source (Apache-2.0)**: Edge proxy, CLI, SDK, A2A/MCP gateways, AgentContract, and local policy engine.
   - **Enterprise SaaS / Self-Hosted**: Advanced multi-region control plane, automated continuous policy generation, enterprise RBAC/SSO, long-term compliance audit storage, and automated SLA regression detection.
