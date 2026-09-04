# Threat Model & Security Boundaries

## 1. Threat Scenarios & Mitigations

| Threat Scenario | Attack Vector | AgentMesh Defense & Mitigation |
|---|---|---|
| **Agent Identity Spoofing** | Attacker crafts an HTTP payload claiming to be a privileged agent. | AgentMesh enforces cryptographic API keys (`mesh_...`), mTLS readiness, and server-side tenant resolution. Identity is never trusted from request body alone. |
| **Delegation Escalation** | Low-privilege Agent A calls high-privilege Agent B to bypass tool restrictions. | Context propagation enforces that origin caller and all intermediaries must hold permission for the requested tool (`ErrPrivilegeEscalation`). |
| **Prompt Injection into Tools** | Attacker manipulates model output to issue unauthorized tool calls. | Deterministic policy engine checks every tool call. AgentMesh acts as a deterministic containment layer, blocking denied tools regardless of model hallucinations. |
| **Delegation Recursion Attack** | Malicious agent loops `A -> B -> A` to exhaust memory and compute. | Ordered stack cycle detection halts cyclic calls immediately (`ErrCycleDetected`). |
| **Parameter Tampering (HITL)** | Agent changes payment recipient or amount after human approval was granted. | Approval tokens cryptographically bind to the SHA-256 hash of parameters. Any parameter change invalidates the token (`ErrApprovalTampered`). |
| **Cost & Token Exhaustion** | Runaway agent calls expensive models or tools in tight loops. | Hard budget tracker halts execution when token, cost, or tool call ceilings are breached. |
| **Telemetry Leaks** | Secrets, keys, and private tokens emitted into shared trace aggregators. | Centralized secret scrubber automatically redacts bearer tokens, Google API keys, OpenAI keys, and passwords. |
| **Tampered Control Plane Config** | Attacker intercepts config sync or injects fake policies into proxies. | Ed25519 cryptographic signatures verify publisher key ID and freshness. Expired or tampered bundles are rejected. |
