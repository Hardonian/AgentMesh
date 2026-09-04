# AgentMesh System Architecture

## 1. High-Level Design

AgentMesh separates control concerns from data plane execution:

```text
          AGENTS (A2A / MCP / Google ADK)
                        │
                        ▼
            AGENTMESH DATA PLANE (agentmesh-proxy)
       ┌──────────────────────────────────────────────┐
       │ - Identity Validation & Tenant Scopes        │
       │ - Deterministic Policy Evaluation (ALLOW/DENY│
       │ - Delegation Graph Anti-Escalation           │
       │ - Circuit Breakers & Idempotent Retries      │
       │ - Bounded Deadlines & Rate Limits            │
       │ - Token/Cost Hard Budget Enforcement         │
       │ - Secret Scrubbing & OpenTelemetry Exporters │
       └──────────────────────┬───────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
 Downstream Agents      MCP Tool Servers      Model Providers
  (A2A Protocol)      (BigQuery/Cloud Run)  (Gemini/Vertex AI)
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              ▼
           AGENTMESH CONTROL PLANE (agentmesh-controller)
       ┌──────────────────────────────────────────────┐
       │ - Agent Registry & Lifecycle                 │
       │ - AgentContract & AgentBOM Store             │
       │ - Agent Passport Operational Metrics Engine  │
       │ - Cryptographic Ed25519 Config Signing       │
       │ - Progressive Delivery (Canary / Shadow)     │
       │ - HITL Approvals Queue & Token Verification  │
       │ - Append-Only Cryptographic Audit Log        │
       └──────────────────────────────────────────────┘
```

## 2. Decoupled Data Plane & Control Plane

A critical architectural invariant is that **the control plane does not sit synchronously in every production request path**.

1. **Signed Config Distribution**: The control plane compiles active agent contracts, policies, and route tables into an Ed25519 cryptographically signed configuration bundle.
2. **Local Caching & Offline Survivability**: The `agentmesh-proxy` verifies the signature against its trusted key ring and stores the active bundle in memory.
3. **Outage Resilience**: If the control plane, database, or network experiences an outage, the proxy continues routing and enforcing policy using its last-known-good signed configuration.

## 3. Subsystems

### Identity

Stable cryptographic identities (`ag_...`, `tool_...`, `cred_...`) and SHA-256 hashed API keys. Scoped permissions prevent unauthorized agent invocation or configuration changes.

### Policy Engine

Pure declarative, deterministic evaluation. Returns `ALLOW`, `DENY`, or `REQUIRE_APPROVAL`. Never relies on an LLM for canonical authorization.

### Capability-Based Routing

Matches task requirements to registered agents using multi-stage filtering (health, policy eligibility, cost, latency SLO) and deterministic ranking strategies.

### A2A Firewall & MCPGuard

- A2A Firewall protects agent-to-agent interactions, terminating cycles and preventing privilege escalation through delegation.
- MCPGuard sits as a reverse proxy gateway in front of MCP servers, intercepting dangerous operations and enforcing data classification controls.
