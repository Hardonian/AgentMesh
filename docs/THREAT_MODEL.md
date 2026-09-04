# STRIDE Threat Model & Security Architecture — AgentMesh

## 1. Executive Summary & Security Objectives

AgentMesh acts as the zero-trust control plane and data-plane security proxy for autonomous AI agents, multi-agent swarms communicating over A2A (Agent-to-Agent), and tools exposed via MCP (Model Context Protocol).

The primary security objectives of AgentMesh are:

1. **Deterministic Enforcement**: Model hallucinations, prompt injections, or adversarial inputs cannot bypass deterministic security invariants.
2. **Multi-Tenant Isolation**: Complete physical and logical segregation of tenant data, policies, credentials, and telemetry (Fail-Closed RLS).
3. **Cryptographic Provenance**: Every policy decision, delegation context, config bundle, and HITL approval token is cryptographically signed and tamper-evident.
4. **Least-Privilege Containment**: Tool access, delegation depth, and resource consumption are bounded with strict default-deny policies.

---

## 2. Trust Boundaries

```text
[ External / Adversarial World ]
             |
             | [Trust Boundary 1: External Network Ingress & SSRF Protection]
             v
   +--------------------+
   | Data Plane Proxy   | <==== [Trust Boundary 4: Control-to-Data Plane (Ed25519 Signed)]
   +--------------------+
     |                |
     | [TB 2: A2A]    | [TB 3: MCP Tool Gate]
     v                v
+----------+    +------------+
| Target   |    | MCP Server | <==== [Trust Boundary 5: Human-In-The-Loop Approvals]
| Agent    |    | / Tools    |
+----------+    +------------+
             |
             v
 [Trust Boundary 6: Multi-Tenant Data Store (Row-Level Security & Encrypted Creds)]
```

### Boundary 1: External Network Ingress

- Untrusted callers sending HTTP/A2A requests to AgentMesh endpoints.
- SSRF risk on outbound webhooks, callback endpoints, and remote A2A proxy registrations.

### Boundary 2: Agent-to-Agent (A2A) Delegation

- Agents requesting downstream execution from other agents.
- Confused deputy attacks where a low-privilege agent leverages a high-privilege agent to access forbidden tools.
- Uncontrolled recursive delegation loops causing compute exhaustion.

### Boundary 3: Agent-to-MCP Tool Execution

- Agents invoking tools with arbitrary generated JSON parameters.
- Tool schema drift, unexpected parameter mutations, and destructive execution (e.g. database drop, funds transfer).

### Boundary 4: Control Plane to Data Plane Proxy Fleet

- Propagation of configuration bundles, routing decisions, canary weights, and deterministic policies.
- Untrusted network transport between control cluster and edge proxies.

### Boundary 5: Human-In-The-Loop (HITL) Approvals

- Manual review and cryptographic authorization of sensitive/destructive actions.
- Post-approval parameter tampering and token replay attacks.

### Boundary 6: Multi-Tenant Data Storage

- PostgreSQL and distributed cache layers shared across tenants.
- Cross-tenant data leakage via SQL query injection or missing tenant scope.

---

## 3. STRIDE Threat Analysis & Mitigations

### 3.1. Spoofing (Identity & Authenticity)

| Threat ID | Threat Description | Attack Vector | AgentMesh Mitigation & Invariant |
| --- | --- | --- | --- |
| **S-01** | Agent Identity Impersonation | Caller presents arbitrary `AgentID` in request headers or JSON payload to access tools assigned to another agent. | **Identity Attestation**: Identity is verified via cryptographically signed API keys (`mesh_...`), SPIFFE/mTLS certificates, or Ed25519 client signatures. Request body `agent_id` is validated against authenticated context. |
| **S-02** | Tenant Context Spoofing | Multi-tenant caller attempts to supply forged `X-Tenant-ID` header. | **Context Binding**: Tenant identity is resolved strictly from verified database API credentials (`store.GetCredentialByKey`) and bound to request context. Unauthenticated or empty tenant requests fail closed (`ErrEmptyTenant`, `ErrUnauthorized`). |
| **S-03** | Config Publisher Forgery | Attacker on internal network broadcasts rogue configuration updates to proxies. | **Ed25519 Signing**: All configuration bundles must be signed by authorized Control Plane keypairs. Proxies verify signature with trusted public key before applying (`ErrSignatureInvalid`). |
| **S-04** | A2A Handshake Forgery | Malicious node claims to be a trusted remote peer agent. | **A2A Handshake Verification**: Strict cryptographic challenge-response and endpoint verification. SSRF validator ensures peer IP is not loopback, private RFC1918, or cloud metadata. |

### 3.2. Tampering (Integrity & Non-Malleability)

| Threat ID | Threat Description | Attack Vector | AgentMesh Mitigation & Invariant |
| --- | --- | --- | --- |
| **T-01** | HITL Parameter Tampering | Attacker intercepts an approved action and mutates execution parameters (e.g., changing recipient account or SQL statement). | **Cryptographic Parameter Binding**: Approval tokens bind directly to `SHA-256(canonical_json(params))`. On execution, `ValidateApproval` recalculates hash. Any parameter deviation fails with `ErrApprovalTampered`. |
| **T-02** | Config Bundle Tampering | Man-in-the-middle modifies policy rules in transit to disable tool restrictions. | **Digest Verification**: Bundles carry Ed25519 signatures and SHA-256 digests. Proxies verify content digest before unmarshaling. |
| **T-03** | Config Downgrade / Replay | Attacker replays an older valid signed config bundle that contained permissive legacy rules. | **Monotonic Versioning**: Every bundle contains monotonically increasing `version` and `issued_at`. Proxies reject any bundle where `version <= current_version` or `issued_at < current_issued_at`. |
| **T-04** | Contract Deserialization Bomb | Attacker uploads malicious, deeply-nested or cyclic YAML/JSON payload to crash parser. | **Size-Bounded Deserialization**: Strict 10MB pre-read limit (`MaxContractPayloadBytes`). Parsers reject oversized inputs immediately before heap allocation. |

### 3.3. Repudiation (Accountability & Auditability)

| Threat ID | Threat Description | Attack Vector | AgentMesh Mitigation & Invariant |
| --- | --- | --- | --- |
| **R-01** | Unauthorized Action Deniability | Operator or automated agent executes a destructive tool and denies origin. | **Agent Passport V2 & Audit Trail**: Every tool call and policy decision records caller identity, cryptographic fingerprint, timestamp, and evaluation reason. |
| **R-02** | Non-Repudiation in HITL | Approver claims they did not approve a sensitive transaction. | **Signed Approver Identity**: Approval record stores immutable reviewer ID, resolution timestamp, and cryptographically generated one-time token bound to reviewer credentials. |

### 3.4. Information Disclosure (Confidentiality & Privacy)

| Threat ID | Threat Description | Attack Vector | AgentMesh Mitigation & Invariant |
| --- | --- | --- | --- |
| **I-01** | Credential Leakage in Telemetry | Agent prompt or tool parameters containing API keys/passwords are emitted to logs and distributed traces. | **Automated Secret Scrubbing**: All logs, traces, and metrics pass through `internal/telemetry` secret scrubber, matching and redacting Bearer tokens, OpenAI keys, Anthropic keys, AWS secrets, GCP keys, passwords, and private certificates with `[REDACTED_*]`. |
| **I-02** | Cross-Tenant Data Leakage | Tenant A queries API and retrieves routing history, passports, or agents belonging to Tenant B. | **Row-Level Security (RLS)**: Database queries strictly enforce `tenant_id` WHERE clauses and Postgres RLS. All store listing methods fail closed on empty tenant ID (`ErrEmptyTenant`). |
| **I-03** | Internal Topology Exposure in Passports | Public inspection of Agent Passport reveals internal microservices, private VPC endpoints, or unreleased models. | **Public Passport Sanitization**: `SanitizeForPublic()` strips internal routing topology, cluster nodes, raw latency breakdowns, and private metadata when `is_public=false` or when serving unauthenticated clients. |
| **I-04** | Memory Dump Disclosure | Attacker inspects memory or core dumps to extract auth tokens. | **Constant-Time Operations**: Auth tokens and hashes use `crypto/subtle.ConstantTimeCompare` to eliminate timing side-channels. |

### 3.5. Denial of Service (Availability & Resilience)

| Threat ID | Threat Description | Attack Vector | AgentMesh Mitigation & Invariant |
| --- | --- | --- | --- |
| **D-01** | Recursive Delegation Loop | Agents recursively delegate calls (`Agent A -> Agent B -> Agent A`) exhausting memory and goroutines. | **Stack Depth & Cycle Detection**: Every A2A delegation propagates an ordered call stack. Stack depth is capped at 5 (`MaxDelegationDepth`), and repeat appearances of any agent trigger immediate `ErrCycleDetected`. |
| **D-02** | Unbounded Request Body / OOM | Attacker sends gigabytes of streamed data to control plane or proxy endpoints. | **Body Limit Middleware**: HTTP middleware wraps requests with `http.MaxBytesReader(w, r.Body, 10MB)`. Oversized payloads are aborted with HTTP 413. |
| **D-03** | Tool Call Hang / Resource Locking | Tool execution hangs indefinitely, holding proxy connections and worker threads. | **Tool Call Timeouts**: Strict context timeouts (default 30s) enforced on all tool invocations. Timed-out calls return `ErrToolTimeout`. |
| **D-04** | Route Storm / Thundering Herd | Multiple agents flood routing engine simultaneously during upstream model outage. | **Rate-Limiting & Concurrency Control**: Proxy enforces connection pooling, semaphore-bounded routing concurrency, and last-known-good (LKG) cached routes during outages. |
| **D-05** | Rogue Automation Runaway | Automated agent enters an infinite loop issuing thousands of mutating tool actions. | **Emergency Freeze Kill Switch**: Control Plane provides instant tenant-wide or mesh-wide automation freeze, halting all autonomous action execution. |

### 3.6. Elevation of Privilege (Authorization & Confinement)

| Threat ID | Threat Description | Attack Vector | AgentMesh Mitigation & Invariant |
| --- | --- | --- | --- |
| **E-01** | Delegation Privilege Escalation (Confused Deputy) | Agent A (unauthorized for tool T) calls Agent B (authorized for tool T) to execute tool T on its behalf. | **Delegation Taint Propagation**: Downstream tool policy evaluation checks permissions of ALL intermediaries in the delegation chain. If any caller lacks authorization, the action is denied (`ErrPrivilegeEscalation`). |
| **E-02** | Tool Classification Bypass | Attacker submits destructive operation disguised as a READ-only query. | **Deterministic Tool Risk Classification**: Tools are strictly categorized (`READ`, `WRITE`, `DESTRUCTIVE`, `CRITICAL`). Policies deterministically enforce approval requirements based on tool classification, regardless of agent prompt intent. |
| **E-03** | RBAC Role Bypass | User with `VIEWER` role attempts to trigger deployment, create policies, or approve actions. | **Fail-Closed RBAC Middleware**: Every control plane endpoint evaluates caller role against strict permission matrix (`RequireRole`). Missing or insufficient role results in immediate HTTP 403 Forbidden. |

---

## 4. Residual Risk Assessment & Ongoing Mitigations

1. **Compromised Root Signing Key**: If the Ed25519 root private key is compromised, an attacker could sign valid config bundles.
   - *Mitigation*: Runbook `RB-06-KEY-ROTATION.md` provides immediate key revocation and rotation procedures. Production keys are stored in KMS/HSM (e.g. Google Cloud KMS, AWS KMS).
2. **Subtle Timing Variations in Network Transport**: Network jitter could theoretically leak minimal timing metadata.
   - *Mitigation*: Constant-time token comparisons (`subtle.ConstantTimeCompare`) in application layer; TLS 1.3 padding across WAN boundaries.
3. **Third-Party Model Provider Compromise**: If an external LLM provider is compromised, generated tool calls may become adversarial.
   - *Mitigation*: AgentMesh assumes the LLM is fully untrusted. All tool calls, parameter schemas, and permissions are verified by deterministic code before execution.
