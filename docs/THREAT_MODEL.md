# STRIDE Threat Model & Security Architecture Specification — AgentMesh V1.0

## 1. Executive Summary & Zero-Trust Governance

AgentMesh acts as the zero-trust control plane and data-plane security proxy for autonomous AI agents, multi-agent swarms communicating over A2A (Agent-to-Agent), and tools exposed via MCP (Model Context Protocol).

The core security invariants of AgentMesh are non-negotiable:

1. **Deterministic Enforcement Over Prompt-Based Filtering**: Model hallucinations, prompt injections, and adversarial jailbreaks cannot bypass deterministic Go-compiled authorization gates.
2. **Fail-Closed Multi-Tenant Isolation**: Complete physical and logical segregation of tenant data, policies, credentials, and telemetry (`ErrEmptyTenant` invariant).
3. **Cryptographic Provenance**: Every policy decision, delegation context, config bundle, and Human-in-the-Loop (HITL) approval token is cryptographically signed and tamper-evident.
4. **Least-Privilege Sandboxing & Blast Radius Containment**: Tool access, delegation depth, and resource consumption are bounded with strict default-deny policies.

---

## 2. Comprehensive System Architecture & Trust Boundaries

AgentMesh operates across 10 distinct architectural trust boundaries:

```text
                                [ External / Adversarial Callers ]
                                                │
                          [TB-01: Network Ingress & SSRF Protection]
                                                ▼
     ┌───────────────────────────────────────────────────────────────────────────────────────┐
     │                             AgentMesh Data-Plane Proxy                                │
     │                                                                                       │
     │  [TB-04: Control Plane Sync] ◄─── (Ed25519 Signed Bundles / Anti-Downgrade Cache)     │
     │  [TB-08: Model Egress]       ───► (Deterministic Schema Validator / Semantic Gateway) │
     │  [TB-09: Telemetry Redactor] ───► (Regex Secret Scrubber -> OTel Collector)           │
     └──────────────┬──────────────────────────────────────────┬─────────────────────────────┘
                    │                                          │
       [TB-02: A2A Swarm Wire]                     [TB-03: MCP Tool Gate]
                    ▼                                          ▼
     ┌──────────────────────────────┐           ┌─────────────────────────────┐
     │      Downstream Agents       │           │      MCP Tool Servers       │
     │ (Max Depth 5, Cycle Check)   │           │ (Risk Class: Destructive)   │
     └──────────────┬───────────────┘           └──────────────┬──────────────┘
                    │                                          │
                    │                           [TB-05: HITL Approvals]
                    │                                          ▼
                    │                          ┌─────────────────────────────┐
                    │                          │   Human Reviewer (Token)    │
                    │                          └─────────────────────────────┘
                    ▼                                          ▼
     ┌───────────────────────────────────────────────────────────────────────────────────────┐
     │                            AgentMesh Control Plane Fleet                              │
     │                                                                                       │
     │  [TB-06: Database Layer]   ───► (PostgreSQL Fail-Closed RLS / MemoryStore)            │
     │  [TB-07: Secrets Vault]    ───► (Google Cloud KMS / HashiCorp Vault Envelope Encrypt) │
     │  [TB-10: K8s Operator]     ───► (Admission Webhook / Mutating Webhook Sidecar Inject) │
     └───────────────────────────────────────────────────────────────────────────────────────┘
```

### Trust Boundary Definitions

| Boundary ID | Boundary Name | Ingress Point | Egress Point | Trust Transition |
| --- | --- | --- | --- | --- |
| **TB-01** | Network Ingress & SSRF Protection | Public Internet / Untrusted WAN | Proxy HTTP / gRPC Ingress | Untrusted External -> DMZ Proxy |
| **TB-02** | A2A Swarm Wire Protocol | Peer Agent Client | Local Proxy / Downstream Agent | External Agent -> Controlled Swarm |
| **TB-03** | MCP Tool Execution Gate | Agent Host Process | Remote MCP Tool Host | Generative Agent Code -> Privileged Tool |
| **TB-04** | Control Plane to Data Plane Fleet | Control Cluster API | Distributed Proxy Caches | Central Orchestration -> Edge Enforcer |
| **TB-05** | Human-In-The-Loop (HITL) Review | Webhook / UI Reviewer | Approval Resolution Service | Human Operator -> Authorization Engine |
| **TB-06** | Multi-Tenant Data Store | Internal Service DAOs | PostgreSQL / DB Pool | Service Layer -> Persistent Storage |
| **TB-07** | Secret Management & Credential Injection | Vault / KMS Storage | Tool Execution Environment | Static Storage -> Ephemeral Process |
| **TB-08** | Model / LLM Egress | Proxy Outbound Client | External Model API (Vertex/Gemini) | Mesh Perimeter -> Cloud AI Service |
| **TB-09** | Telemetry & Observability Pipeline | Internal Trace Collector | OTel / BigQuery Exporter | Protected Memory -> Observability Sink |
| **TB-10** | Kubernetes Operator & Sidecar Webhook | K8s API Server | Pod Injection Spec | Cluster Control -> Pod Runtime |

---

## 3. Threat Classification Framework & Industry Alignment

Every threat scenario evaluated by AgentMesh is systematically correlated against three gold-standard industry security taxonomies:

1. **STRIDE**: Spoofing, Tampering, Repudiation, Information Disclosure, Denial of Service, Elevation of Privilege.
2. **OWASP Top 10 for LLMs (2025/2026)**:
   - LLM01: Prompt Injection
   - LLM02: Sensitive Information Disclosure
   - LLM03: Supply Chain Vulnerabilities
   - LLM04: Data and Model Poisoning
   - LLM05: Improper Output Handling
   - LLM06: Excessive Agency
   - LLM07: System Prompt Leakage
   - LLM08: Vector and Embedding Weaknesses
   - LLM09: Misinformation
   - LLM10: Unbounded Consumption
3. **MITRE ATLAS (Adversarial Threat Landscape for AI Systems)** & **NIST AI RMF 1.0** (Govern, Map, Measure, Manage).

---

## 4. 50+ STRIDE Threat Scenarios, Attacks & Invariants

### 4.1. Spoofing (Identity & Authenticity)

| ID | STRIDE / OWASP / MITRE | Threat Description | Attack Vector | AgentMesh Hardened Mitigation & Invariant |
| --- | --- | --- | --- | --- |
| **S-01** | Spoofing / LLM06 / AML.T0040 | Agent Identity Impersonation | Adversary generates request containing victim's `agent_id` to execute restricted tools. | **Cryptographic Context Binding**: Request `agent_id` must match authenticated identity token or SPIFFE mTLS SAN. Identity is resolved from verified credentials (`mesh_...`) and locked into `r.Context()`. Unauthenticated requests fail closed. |
| **S-02** | Spoofing / LLM02 / AML.T0043 | Multi-Tenant Tenant Forgery | Malicious tenant sends header `X-Tenant-ID: victim-corp` to hijack route outcomes. | **Fail-Closed Context Resolution**: `X-Tenant-ID` is authenticated against database credential key hash. If header does not match authenticated token or is empty, proxy returns `ErrEmptyTenant` (HTTP 401). |
| **S-03** | Spoofing / LLM03 / AML.T0010 | Rogue Configuration Injection | Attacker on internal network broadcasts rogue config bundles to proxies. | **Ed25519 Cryptographic Verification**: Every bundle must carry Ed25519 signature verified by `KeyRing.Verify()`. Unsigned or invalidly signed updates are dropped immediately (`ErrSignatureInvalid`). |
| **S-04** | Spoofing / LLM06 / AML.T0040 | A2A Remote Peer Handshake Forgery | Malicious agent mimics legitimate peer during A2A protocol negotiation. | **Cryptographic Handshake Challenge**: A2A handshake executes mutual nonce signature verification. SSRF validator ensures peer IP is not loopback, RFC1918, or cloud metadata. |
| **S-05** | Spoofing / LLM06 / AML.T0043 | Human Approver Impersonation | Attacker submits fake approval payload pretending to be authorized security engineer. | **HMAC-SHA256 Token Binding**: Approvals require cryptographically generated one-time tokens bound to reviewer credentials, timestamp, and action hash. |
| **S-06** | Spoofing / LLM03 / AML.T0018 | Rogue MCP Tool Registration | Rogue agent registers malicious tool name matching built-in tool (e.g. `bigquery.read`). | **Tool Passport Namespace Reservation**: System and core tools are cryptographically namespaced and cannot be shadowed by dynamic registrations without `admin:tools` role. |
| **S-07** | Spoofing / LLM01 / AML.T0043 | MCP JSON-RPC ID Collision Spoofing | Concurrent attacker injects JSON-RPC responses with overlapping IDs to hijack results. | **Cryptographic Request-Response Correlation**: MCP gateway verifies request ID, tenant context, and session UUID before returning result to caller. |
| **S-08** | Spoofing / LLM03 / AML.T0010 | Sidecar Admission Webhook Spoofing | Attacker calls mutation webhook directly to forge sidecar container definitions. | **mTLS Webhook Authentication**: Webhook requires client cert signed by Kubernetes API CA. Requests without valid client cert are rejected with HTTP 403. |

### 4.2. Tampering (Integrity & Non-Malleability)

| ID | STRIDE / OWASP / MITRE | Threat Description | Attack Vector | AgentMesh Hardened Mitigation & Invariant |
| --- | --- | --- | --- | --- |
| **T-01** | Tampering / LLM05 / AML.T0031 | HITL Parameter Mutation (Post-Approval Tampering) | Human approves `DELETE WHERE id = 123`; attacker intercepts and changes to `DELETE WHERE 1=1`. | **Parameter Digest Pinning**: Approval token cryptographically binds `SHA-256(canonical_json(parameters))`. Execution recalculates hash; any mutation fails with `ErrApprovalTampered`. |
| **T-02** | Tampering / LLM03 / AML.T0010 | Config Policy In-Transit Tampering | Man-in-the-middle alters policy rules in transit to allow destructive tools. | **SHA-256 Content Digest Verification**: Config bundle payload digest is checked against signed header before unmarshaling. Mismatches abort instantly. |
| **T-03** | Tampering / LLM03 / AML.T0010 | Config Downgrade / Replay Attack | Attacker captures valid signed config from 6 months ago and replays it to restore permissive rules. | **Strict Monotonic Timestamp & Sequence**: Proxies reject any bundle whose `IssuedAt <= current.IssuedAt` or `Version <= current.Version` (`ErrConfigDowngrade`). |
| **T-04** | Tampering / LLM10 / AML.T0029 | Contract Deserialization Zip Bomb | Attacker uploads deeply nested or circular YAML/JSON payload to trigger OOM. | **10MB Pre-Read Bounds Check**: Deserialization strictly enforces `http.MaxBytesReader` (10MB limit) prior to JSON/YAML unmarshaling (`MaxContractPayloadBytes`). |
| **T-05** | Tampering / LLM04 / AML.T0031 | Routing Outcome Graph Poisoning | Attacker submits fabricated success telemetry for malicious agent to manipulate routing. | **Cryptographic Trace Verification**: Routing outcomes must correlate with signed execution traces logged by data plane proxy before influencing scoring. |
| **T-06** | Tampering / LLM05 / AML.T0015 | Tool Input Schema Drift Mutation | MCP tool server updates schema unexpectedly, adding permissive parameters. | **Schema Drift Differential Engine**: `DetectSchemaDrift()` calculates SHA-256 structural schema diff. Destructive or unknown changes automatically quarantine the tool. |
| **T-07** | Tampering / LLM05 / AML.T0031 | Audit Trail Hash Chain Severing | Attacker with read access attempts to rewrite historic audit records. | **Cryptographic Audit Hash Chaining**: Every audit entry includes `SHA-256(prev_entry_hash + current_event)`. Tampering severs the chain and triggers an alert. |
| **T-08** | Tampering / LLM06 / AML.T0040 | Delegation Stack Header Tampering | Malicious agent strips predecessor caller IDs from A2A delegation stack. | **Cryptographic Stack Attestation**: Delegation stacks are signed or verified with parent delegation tokens; tampering breaks stack validation (`ErrDelegationStackCorrupt`). |

### 4.3. Repudiation (Accountability & Auditability)

| ID | STRIDE / OWASP / MITRE | Threat Description | Attack Vector | AgentMesh Hardened Mitigation & Invariant |
| --- | --- | --- | --- | --- |
| **R-01** | Repudiation / LLM06 / AML.T0040 | Destructive Tool Action Deniability | Compromised agent drops production database table and claims command was not issued. | **Immutable Audit Log & Passport Evidence**: Every tool call records caller agent ID, tenant ID, parameter SHA-256 digest, evaluation rule, and timestamp. |
| **R-02** | Repudiation / LLM06 / AML.T0043 | Human Approver Denial | Reviewer approves funds transfer and later claims they never clicked approve. | **Signed Reviewer Identity**: Approvals log reviewer ID, authentication method, IP address, timestamp, and cryptographic one-time token. |
| **R-03** | Repudiation / LLM03 / AML.T0010 | Canary Deployment Mutation Deniability | Unauthorized user promotes canary to 100% traffic causing outage, then denies action. | **RBAC-Audited Promotions**: Canary promotions record initiator identity, previous weight, new weight, and policy rationale to immutable audit logs. |
| **R-04** | Repudiation / LLM08 / AML.T0040 | Autonomous Policy Optimization Rollback Deniability | Autonomous optimizer changes routing weights; team denies understanding reason. | **Explainable Action Workflows**: Every optimization action creates a durable `AgentOptimizationAction` with state before/after, blast radius, and decision rationale. |

### 4.4. Information Disclosure (Confidentiality & Privacy)

| ID | STRIDE / OWASP / MITRE | Threat Description | Attack Vector | AgentMesh Hardened Mitigation & Invariant |
| --- | --- | --- | --- | --- |
| **I-01** | Info Disclosure / LLM02 / AML.T0035 | API Key & Secret Leakage in Telemetry | Agent generates prompts containing OpenAI/Stripe/GCP keys which are emitted into OTel traces. | **Automated Secret Scrubber**: Regex redaction suite (`internal/telemetry`) scrubs all logs, spans, and metrics, replacing tokens with `[REDACTED_*]` before output. |
| **I-02** | Info Disclosure / LLM02 / AML.T0043 | Cross-Tenant Data Leakage | Tenant A queries control plane and retrieves Tenant B's agent contracts or policies. | **Fail-Closed Row-Level Security**: Every SQL query is parameterized with authenticated `tenant_id`. Missing tenant IDs reject immediately (`ErrEmptyTenant`). |
| **I-03** | Info Disclosure / LLM07 / AML.T0037 | Internal Topology Exposure in Passports | Public inspection of Agent Passport reveals internal microservices, VPC endpoints, or DB names. | **Public Passport Sanitization**: `SanitizeForPublic()` strips internal endpoints, private tool schemas, cost contracts, and tenant IDs when `is_public=false` or public query flag is set. |
| **I-04** | Info Disclosure / LLM02 / AML.T0038 | Timing Attacks on Token Verification | Attacker measures network latency differences to infer valid authorization tokens byte-by-byte. | **Constant-Time Comparison**: All API keys, hashes, and HITL tokens are compared using `crypto/subtle.ConstantTimeCompare`. |
| **I-05** | Info Disclosure / LLM01 / AML.T0043 | SSRF on Outbound Webhooks & Peer URLs | Attacker registers webhook URL `http://169.254.169.254/latest/meta-data/` to steal cloud credentials. | **Strict SSRF Validator**: `ValidateSafeRemoteURL()` resolves IP addresses and blocks loopback (127.0.0.0/8), RFC1918 (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16), and cloud metadata IPs (169.254.169.254). |
| **I-06** | Info Disclosure / LLM02 / AML.T0035 | Database Memory Dump Disclosure | Core dump or crash exposes unencrypted credentials stored in memory. | **Short-Lived Memory & Hashed Storage**: API keys are hashed with SHA-256 immediately upon ingest. Raw plaintext secrets are never stored in database tables. |
| **I-07** | Info Disclosure / LLM02 / AML.T0035 | CORS Misconfiguration Leakage | Attacker site executes cross-origin AJAX calls to steal active agent sessions. | **Strict CORS Allowlist**: CORS middleware restricts origins to explicit allowlisted endpoints (`http://localhost:3000`, `http://127.0.0.1:8080`), rejecting wildcard `*`. |
| **I-08** | Info Disclosure / LLM02 / AML.T0035 | Clickjacking & MIME-Sniffing Attacks | Malicious frame embeds AgentMesh dashboard to trick operators into approving actions. | **Security Headers On All Responses**: Server injects `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Content-Security-Policy: default-src 'self'`, and `Strict-Transport-Security`. |

### 4.5. Denial of Service (Availability & Resilience)

| ID | STRIDE / OWASP / MITRE | Threat Description | Attack Vector | AgentMesh Hardened Mitigation & Invariant |
| --- | --- | --- | --- | --- |
| **D-01** | DoS / LLM10 / AML.T0029 | Recursive Delegation Infinite Loop | Agent A calls Agent B which calls Agent A, creating unbounded call loop and goroutine exhaustion. | **Stack Depth & Cycle Detection**: Stack depth is strictly bounded to 5 (`MaxDelegationDepth`). Repeat occurrences of any agent in stack immediately fail with `ErrCycleDetected`. |
| **D-02** | DoS / LLM10 / AML.T0029 | Unbounded Request Body Flooding | Attacker streams multi-gigabyte payload to exhaust server memory. | **Pre-Read Payload Cap**: All HTTP endpoints wrap bodies in `http.MaxBytesReader(10MB)`. Requests exceeding limit return HTTP 413. |
| **D-03** | DoS / LLM10 / AML.T0029 | Unbounded Tool Call Hanging | Slow or hanging third-party MCP tool holds connection threads indefinitely. | **Context Timeout Enforcement**: Tool executions have strict context deadlines (default 30s). Expired calls abort with `ErrToolTimeout`. |
| **D-04** | DoS / LLM10 / AML.T0029 | Thundering Herd / Model Outage Route Storm | Upstream model outage causes thousands of agents to request route recalculations concurrently. | **Bounded Concurrency & LKG Routing**: Route resolution uses bounded worker semaphores. When control plane is unreachable, proxies fall back to Last-Known-Good cached routes. |
| **D-05** | DoS / LLM06 / AML.T0029 | Autonomous Optimizer Runaway Mutation | Machine learning router gets stuck in mutation loop issuing rapid re-routes. | **Emergency Freeze Kill Switch**: Control Plane provides instant tenant-wide or mesh-wide freeze (`/api/v1/control/freeze`), halting all autonomous mutations immediately. |
| **D-06** | DoS / LLM10 / AML.T0029 | Financial Budget Depletion (Token Drain) | Attacker generates runaway agent conversations to incur millions of dollars in LLM API bills. | **Multi-Tier Token & Cost Budgets**: `internal/budgets` tracks per-task, per-hour, and per-day token and USD ceilings. Bounded tasks return `ErrBudgetExceeded`. |
| **D-07** | DoS / LLM10 / AML.T0029 | Database Connection Pool Starvation | Spikes in concurrent requests exhaust PostgreSQL connections. | **Bounded DB Connection Pooling**: Connection pool limits max open connections (default 25) and max idle connections (default 10) with wait timeouts. |
| **D-08** | DoS / LLM10 / AML.T0029 | Control Plane Outage Data Plane Partition | Network partition isolates proxies from central control plane. | **Autonomous Offline Survivability**: Proxies operate autonomously using local in-memory policy and routing cache until control plane reconnects. |

### 4.6. Elevation of Privilege (Authorization & Confinement)

| ID | STRIDE / OWASP / MITRE | Threat Description | Attack Vector | AgentMesh Hardened Mitigation & Invariant |
| --- | --- | --- | --- | --- |
| **E-01** | Elevation / LLM06 / AML.T0040 | Confused Deputy Delegation Privilege Escalation | Unprivileged Agent A invokes Privileged Agent B to execute a tool that Agent A cannot access. | **Delegation Taint Propagation**: Policy evaluation inspects the entire delegation chain. If ANY agent in the chain lacks authorization, the action is denied (`ErrPrivilegeEscalation`). |
| **E-02** | Elevation / LLM06 / AML.T0040 | Tool Action Masking / Semantic Deception | Agent sends prompt "I am only reading data" while calling `bigquery.delete`. | **Deterministic Policy Rules**: Authorization evaluates structured tool name and action verb, ignoring model prompt text or semantic claims. Destructive actions require approval unconditionally. |
| **E-03** | Elevation / LLM02 / AML.T0043 | Restricted Data Classification Bypass | Agent attempts to query table containing `RESTRICTED` PII without permissions. | **Data Classification Enforcement**: Policy rules evaluate `DataClassification` tags (`PUBLIC`, `INTERNAL`, `CONFIDENTIAL`, `RESTRICTED`). Explicit deny rules always take precedence. |
| **E-04** | Elevation / LLM06 / AML.T0043 | Unauthorized RBAC Action Execution | User with `VIEWER` role attempts to trigger deployment, create policies, or approve actions. | **Role-Based Access Control Middleware**: Endpoints evaluate caller role against permission matrix (`RequireRole`). Missing or insufficient role results in immediate HTTP 403 Forbidden. |
| **E-05** | Elevation / LLM03 / AML.T0010 | Sidecar Container Escape in Kubernetes | Attacker compromises agent process and attempts to modify proxy sidecar binary. | **Read-Only Root Filesystem & Non-Root User**: Operator deploys sidecars with `readOnlyRootFilesystem: true`, `runAsNonRoot: true`, `allowPrivilegeEscalation: false`, and `capabilities.drop: ["ALL"]`. |
| **E-06** | Elevation / LLM06 / AML.T0040 | Expired Approval Token Re-execution | Attacker attempts to reuse an old approved token to execute a second destructive transaction. | **Single-Use Approval Tokens**: Approval service marks tokens as `RESOLVED` immediately upon execution. Repeat attempts fail with `ErrApprovalAlreadyUsed`. |
| **E-07** | Elevation / LLM06 / AML.T0040 | Approval Token TTL Window Violation | Attacker presents an approval token that was issued 48 hours ago. | **Strict Token Expiration (TTL)**: Tokens carry maximum lifespan (default 15 minutes). Requests beyond TTL fail with `ErrApprovalExpired`. |
| **E-08** | Elevation / LLM03 / AML.T0010 | Unverified Agent Contract Registration | Untrusted agent registers invalid contract with arbitrary unvetted capabilities. | **Strict Schema Validation**: Control plane validates contracts against JSON schema, verifying semantic versioning, valid protocols, and capability strings. |

---

## 5. Defense-in-Depth Verification & Residual Risk

AgentMesh verifies all 50+ threat scenarios via continuous CI/CD pipelines:

- **Red Team Scenarios**: `tests/phase5_redteam_test.go` exercises 15 hostile attack vectors (SSRF, tenant cross-talk, token tampering, downgrade attacks, race storms).
- **Definition of Done Certifications**: `tests/phase5_dod35_test.go` verifies 35 architectural non-negotiables.
- **Race Condition Testing**: Compiled and verified with `go test -p 1 -race ./...`.
- **Static Analysis**: Enforced via `go vet ./...` with zero warnings.
