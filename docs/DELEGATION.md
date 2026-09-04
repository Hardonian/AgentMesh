# Multi-Agent Delegation & Anti-Escalation

## The Problem
In multi-agent architectures, Agent A delegates subtasks to Agent B, which may delegate further to Agent C.
Without control-plane enforcement, two critical vulnerabilities emerge:
1. **Delegation Loops / Denial of Service**: Cycles like `A -> B -> A` cause infinite recursion and cost explosions.
2. **Privilege Escalation (Confused Deputy)**: Agent A (forbidden from making payments) delegates a task to Agent B (which has payment privileges), indirectly achieving unauthorized action.

## AgentMesh Defense Invariants
1. **Cycle Detection**: Every delegation request carries an ordered `DelegationStack`. If any target agent already exists in the stack, execution terminates immediately with `ErrCycleDetected`.
2. **Bounded Depth**: Contracts enforce `maxDepth` (e.g. 3 hops). Any call exceeding this bound fails.
3. **Anti-Escalation Verification**: When any agent in the chain attempts a tool call, AgentMesh verifies that **both the executing agent and all upstream callers in the stack** are permitted to perform that action under active policy.
