# AgentMesh v1.0 Scope, Guarantees & Non-Goals

## 1. Explicit Architectural Guarantees in v1.0

1. **Deterministic Containment**:
   - Zero LLM code in the authorization path.
   - Policies evaluate compiled Go rules (`ALLOW`, `DENY`, `REQUIRE_APPROVAL`).
   - Prompt injections or model hallucinations cannot bypass deterministic tool gates.
2. **Multi-Tenant Physical & Logical Isolation**:
   - Every database query strictly enforces `tenant_id` and PostgreSQL Row-Level Security.
   - Listing and mutating methods fail closed on empty or mismatched tenant ID (`ErrEmptyTenant`).
3. **Control Plane Outage Independence**:
   - Data plane edge proxies continue routing, enforcing policy, and logging traces using local Last Known Good (LKG) signed config bundles if the control plane or database fails.
4. **Finite Delegation Invariants**:
   - Max delegation depth is capped at 5 hops (`MaxDelegationDepth`).
   - Ordered stack cycle detection terminates recursive agent loops (`ErrCycleDetected`).
5. **Cryptographic Provenance**:
   - Config bundles are signed with Ed25519 publisher keys.
   - HITL approval tokens bind cryptographically to `sha256(canonical_json(params))` and are consumed single-use.
6. **Zero-Leak Telemetry**:
   - Automated secret scrubbers redact Bearer tokens, OpenAI keys, Anthropic keys, AWS secrets, and passwords before traces or metrics are emitted.

---

## 2. Supported Protocols & Interfaces

- **A2A Protocol**: HTTP/JSON-RPC 2.0 and SSE with cryptographic handshake verification and delegation context propagation.
- **Model Context Protocol (MCP)**: stdio and SSE transport protocols; JSON schema drift detection; tool risk classification (`READ`, `WRITE`, `DESTRUCTIVE`).
- **OpenTelemetry**: Native OTLP gRPC/HTTP exporter for distributed traces and metrics.

---

## 3. Explicit Non-Goals & Out-of-Scope Items for v1.0

1. **Not an Agent Development Framework**:
   - AgentMesh is **not** an agent authoring SDK like LangChain, CrewAI, or AutoGen. It does not provide prompt engineering abstractions or agent loops. Instead, it governs and routes agents built with those frameworks.
2. **Not a General-Purpose API Gateway**:
   - AgentMesh is specialized for agentic communication (A2A, MCP, agent identity, delegation stacks). It is not intended to replace NGINX or Envoy for generic static asset hosting or public website routing.
3. **No Unbounded Autonomous Self-Mutation in Production**:
   - By default, automated route mutation and policy generation operate in advisory/shadow mode. Production mutations require human approval tokens or explicit operator commands.
4. **No Quantum-Resistant Cryptography**:
   - v1.0 relies on Ed25519 elliptic curve signatures and SHA-256 digests. Post-quantum hybrid signatures (ML-DSA) are scheduled for v2.0.
