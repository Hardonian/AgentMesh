# Graph-Level Policy Analysis & Confused Deputy Protection

Traditional agent authorization operates on isolated, single-hop requests. However, in enterprise multi-agent networks, complex security vulnerabilities emerge across **multi-hop delegation paths**.

AgentMesh implements static **Graph Policy Analysis** (`internal/policy/graph_analysis.go`) to evaluate holistic workflow topologies before runtime execution.

---

## Static Path Inspection

Graph Policy Analysis walks all reachable paths from the graph `Entrypoint` to detect:
1. **Forbidden Delegation Paths**: When Agent A delegates to Agent B, which subsequently delegates to Agent C or Tool D forbidden by Agent A's contract.
2. **Indirect Privilege Escalation**: Sub-agents invoking sensitive capabilities on behalf of a caller that lacks those permissions.
3. **Approval Bypass**: Mutating or high-risk tools reachable through a branch that circumvents declared human approval nodes.
4. **Tool Denial Propagation**: Immediate denial if any reachable node depends on a tool explicitly denied by organizational policy.

### Concrete Example: Indirect Exfiltration
```
CEO Agent ──(delegates)──> Finance Agent ──(delegates)──> Research Agent ──(invokes)──> Gmail Send
```
If corporate policy specifies that `Finance Agent` cannot dispatch external communications, Graph Policy Analysis flags this path as an `ERROR: FORBIDDEN_DELEGATION` and rejects the graph during CI or registration, even if `Research Agent` independently possesses the `gmail.send` tool.

---

## Confused Deputy Adversarial Tests
AgentMesh includes explicit test fixtures verifying confused deputy prevention:
- A restricted agent attempting to execute payments through an unrestricted intermediary is rejected:
  `restricted-caller -> privileged-deputy -> payment.execute => DENIED (Confused deputy privilege escalation prevented)`
- Delegation taint propagation ensures that effective capabilities equal the intersection of all principals along the delegation chain.
