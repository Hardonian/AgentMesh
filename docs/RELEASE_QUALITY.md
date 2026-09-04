# Release Quality & Verification Standards — AgentMesh v1.0

## 1. Quality Assurance Philosophy

AgentMesh is infrastructure software designed to operate in the critical network path of enterprise AI agent systems. Its failure modes, latency characteristics, and security invariants must be completely predictable.

Every release of AgentMesh is validated across five independent verification dimensions before distribution.

---

## 2. Five-Dimension Verification Matrix

### Dimension 1: Concurrency & Data Race Free Execution

- **Tool**: Go Race Detector (`go test -race`)
- **Execution**: All packages (`pkg/...`, `internal/...`, `tests/...`) are tested under maximum concurrency with `-race`.
- **Standard**: Zero data races permitted. Any data race detected in CI immediately fails the build.

### Dimension 2: Deterministic Adversarial QA (Red Teaming)

- **Suite**: 15 P0 Red Team Scenarios in `tests/phase5_redteam_test.go`
- **Scope**:
  - SSRF injection against cloud metadata (`169.254.169.254`) and loopback
  - Cross-tenant leakage via forged or empty tenant IDs
  - HITL approval token parameter tampering and replay
  - Contract deserialization memory exhaustion (DoS)
  - Delegation taint escalation (Confused Deputy attack)
  - Config bundle signature forgery and downgrade attacks
  - Clock skew and expired auth tokens
- **Standard**: 100% pass rate. All attack vectors fail closed.

### Dimension 3: Definition-of-Done (DoD) Functional Certification

- **Suite**: 35 Automated DoD Certification Tests in `tests/phase5_certification_test.go`
- **Scope**: Core Proxy invariants, Control Plane RBAC, A2A protocol state machines, MCP schema drift detection, capability-aware routing algorithms, security headers, and emergency kill switches.
- **Standard**: 35 / 35 passing tests required for release authorization.

### Dimension 4: Fuzz Testing (Mutational Coverage)

- **Engines**: 8 native Go fuzz suites:
  - `pkg/agentbom.FuzzAgentBOMUnmarshal`
  - `internal/policy.FuzzPolicyEvaluate`
  - `pkg/spec.FuzzSpecUnmarshal`
  - `internal/a2a.FuzzA2AMessageDecode`
  - `internal/mcp.FuzzMCPFrameDecode`
  - `internal/crypto.FuzzCryptoVerify`
  - `pkg/task.FuzzTaskFingerprintCompute`
  - `internal/reconcile.FuzzReconcileParse`
- **Throughput**: > 500,000 executions per engine with 0 crashes, 0 panics, and 0 unhandled memory allocations.

### Dimension 5: Clean-Room Usability & Zero-Dependency Quickstart

- **Standard**: A developer downloading AgentMesh must be able to run `agentmesh doctor` and `agentmesh demo run` on an air-gapped machine without external network calls or cloud credentials.
