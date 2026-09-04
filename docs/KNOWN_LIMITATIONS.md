# Known Limitations & Operating Boundaries — AgentMesh

## 1. Explicit Architectural & Performance Boundaries

| Boundary | Enforced Limit | Rationale & Security Invariant |
| --- | --- | --- |
| **Max Request Body Size** | **10 MB** (`10 * 1024 * 1024` bytes) | Enforced via `http.MaxBytesReader` to prevent memory exhaustion and DoS via large JSON/YAML payloads. |
| **Max Contract Payload** | **10 MB** (`MaxContractPayloadBytes`) | Enforced in `pkg/contracts/contract.go` to protect against YAML deserialization bombs and nested entity expansion. |
| **Max Delegation Depth** | **5 Hops** (`MaxDelegationDepth`) | Enforced in `internal/a2a` to prevent runaway recursive sub-agent chains and stack overflow. |
| **Delegation Cycle Detection** | **Strict Ordered Stack Loop Check** | Immediate abort (`ErrCycleDetected`) if any agent appears twice within an active delegation trace. |
| **Default Tool Timeout** | **30 Seconds** (configurable up to 120s) | Prevents tool calls from holding proxy worker threads or connection pools indefinitely. |
| **HITL Approval Token TTL** | **15 Minutes** (default) | Bounded lifetime to prevent stale authorization; requires re-approval upon expiration. |
| **DB Connection Pool Max** | **50 Open Connections** (default) | Configured in `internal/database` with `SetMaxOpenConns(50)`, `SetMaxIdleConns(10)`, `SetConnMaxLifetime(5m)` to prevent database connection starvation. |

---

## 2. Storage & Database Engines

- **Primary Supported Database**: PostgreSQL 16+ (requires Row-Level Security support).
- **Embedded / Testing Mode**: In-memory store and SQLite (for single-node unit tests and local CLI evaluation only).
- *Limitation*: Distributed clustering in multi-region environments requires PostgreSQL managed replication or Cloud Spanner adapter (planned for future release).

---

## 3. Cryptography & Key Management

- **Supported Signature Algorithms**: Ed25519 (RFC 8032) for bundle signing and A2A peer verification.
- **Hash Functions**: SHA-256 for parameter binding, contract digests, and task fingerprints.
- *Limitation*: Quantum-resistant hybrid signatures (e.g. Dilithium / ML-DSA) are not yet natively supported in Phase 5.

---

## 4. Model Context Protocol (MCP) Governance

- Supports stdio and SSE (Server-Sent Events) MCP transports.
- Strict structural JSON schema diffing and risk classification (`READ`, `WRITE`, `DESTRUCTIVE`).
- *Limitation*: Arbitrary binary protocol extensions outside standard MCP JSON-RPC 2.0 specifications are rejected.
