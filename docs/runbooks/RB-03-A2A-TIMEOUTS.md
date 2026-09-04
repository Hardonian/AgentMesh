# RB-03: Agent-to-Agent (A2A) Timeouts & Deadlock Resolution

## 1. Metadata

- **Severity**: P1 / P2
- **Target Component**: A2A Communication Engine, Delegation Manager (`internal/a2a`)
- **Relevant Alerts**: `A2ATimeoutSpike`, `A2ADelegationCycleDetected`, `A2ADeadlockDetected`

---

## 2. Symptoms

- Multi-agent collaboration tasks stalling indefinitely or returning HTTP 504 Gateway Timeout.
- Error logs exhibiting `ErrDelegationTimeout`, `ErrCycleDetected`, or `ErrMaxDepthExceeded`.
- Memory or active goroutines climbing on proxy instances handling A2A handshakes.

---

## 3. Immediate Triage (First 5 Minutes)

1. List currently running and hanging A2A tasks:

   ```bash
   agentmesh a2a tasks list --status=running --older-than=60s
   ```

2. Inspect delegation stack trace of stalled tasks:

   ```bash
   agentmesh a2a tasks inspect <task_id> --show-stack
   ```

3. Check for cyclic delegation loops:

   ```bash
   agentmesh a2a graph detect-cycles --tenant=<tenant_id>
   ```

---

## 4. Root Cause Analysis

- **Case A: Unhandled Circular Dependency (A -> B -> A)**
  - Cyclic delegation logic in agent prompts. AgentMesh cycle detector aborts the request, but caller retries infinitely.
  - *Fix*: Break cycle in agent prompt/spec; enforce max delegation depth = 3.
- **Case B: Deadlock on Downstream Tool or External Lock**
  - Two agents waiting on shared resources or reciprocal approvals.
  - *Fix*: Terminate stalled task and release locks.
- **Case C: Slow Sub-Agent Network Connectivity**
  - Callee agent instance unreachable or dropped packets across VPC peering.
  - *Fix*: Fail over callee traffic to redundant standby agent instance.

---

## 5. Remediation Steps

1. **Force terminate stalled A2A tasks**:

   ```bash
   agentmesh a2a tasks abort --older-than=120s --reason="A2A timeout remediation"
   ```

2. **Temporarily reduce max delegation depth to isolate deep chains**:

   ```bash
   agentmesh a2a set-limits --tenant=<tenant_id> --max-depth=3
   ```

3. **If a callee agent is totally unresponsive, mark degraded**:

   ```bash
   agentmesh health set-override --agent=<callee_agent_id> --status=degraded
   ```

---

## 6. Verification & Recovery Confirmation

- Verify active A2A tasks return to normal lifecycle durations (< 15s for standard workflows).
- Confirm zero occurrences of `ErrCycleDetected` in log stream for 15 minutes.
- Verify memory consumption on proxy nodes stabilizes.
