# Agent-to-Agent (A2A) Protocol & A2A Firewall

## Protocol Overview
AgentMesh implements first-class support for the open Agent-to-Agent (A2A) protocol:
- **Agent Cards (`/a2a/agent-card`)**: Machine-readable capabilities, schemas, models, and authentication parameters.
- **Task Invocation (`POST /a2a/tasks`)**: Standardized task payloads with context propagation (`traceId`, `delegationStack`, `tenantId`, `budgetRemaining`).
- **Streaming**: Real-time event streaming for long-running workflows.
- **Cancellation**: Immediate abort signaling across distributed agent workers.

## A2A Firewall
The flagship A2A Firewall governs multi-agent interactions:
- Traditional Firewall: `IP -> Port`
- AgentMesh A2A Firewall: `Caller Agent -> Target Agent -> Capability -> Tools -> Resources`

### Anti-Escalation & Cycle Termination
- **Privilege Anti-Escalation**: If Agent A is forbidden from executing `payment.execute`, delegating the task to Agent B (even if B is nominally permitted to execute payments) will be blocked by the A2A Firewall because the origin caller's security context propagates along the stack.
- **Cycle Detection**: If an invocation graph forms a cycle (`A -> B -> A`), the firewall detects the repeated agent identifier and terminates the invocation with `ErrCycleDetected`.
