# AgentMesh A2A Compatibility Lab

The **AgentMesh A2A Compatibility Lab** (`internal/a2a/lab.go`) provides an objective test harness for verifying agent compliance against the Agent-to-Agent protocol specification (A2A Protocol v0.3.0).

---

## 10 Core Interoperability Test Suites
The Compatibility Lab executes 10 test categories:
1. **Agent Discovery**: Verifies that `GET /a2a/agent-card` returns a compliant metadata card.
2. **Schema Conformance**: Validates Agent Card fields (name, version, capabilities, endpoints, formats).
3. **Task Invocation**: Dispatches a standard task and verifies synchronous or asynchronous task response schemas.
4. **Malformed Request Rejection**: Tests that invalid JSON or missing required fields return proper HTTP 400 errors.
5. **Context & Tenant Propagation**: Verifies that headers (`X-AgentMesh-Tenant-ID`, `X-Trace-Context`) pass through task execution boundaries.
6. **Cancellation Lifecycle**: Issues `POST /a2a/tasks/{id}/cancel` and checks that downstream task execution transitions to `CANCELLED`.
7. **Task State Machine**: Verifies valid transitions: `PENDING` -> `RUNNING` -> `COMPLETED` / `FAILED`.
8. **Artifact Handshake**: Validates artifact generation and transfer metadata.
9. **Streaming Backpressure**: Tests chunked streaming without unbounded memory buffering.
10. **Timeout & Failure Handling**: Tests deadline propagation and structured error responses.

---

## Compatibility Profiles
Test runs produce an `A2ACompatibilityProfile`:
- Status: `COMPATIBLE` (all mandatory tests passed), `PARTIALLY_COMPATIBLE` (non-critical features like streaming failed), or `INCOMPATIBLE`.
- Numerical score ($0.0$ to $1.0$).
- Granular result per test case with duration and error traces.

Run tests via CLI:
```bash
agentmesh a2a test http://localhost:8080
```
