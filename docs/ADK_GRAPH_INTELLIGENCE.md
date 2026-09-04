# Google ADK Graph Intelligence

AgentMesh incorporates native awareness for the Google Agent Development Kit (ADK). It enables static extraction, validation, and risk analysis of Go-based ADK agent topologies without executing untrusted user code or external build scripts.

---

## Static AST Inspection

ADK projects are inspected via `agentmesh adk graph inspect PATH` or programmatically using `adk.NewStaticProjectInspector()`.

The inspector parses Go Abstract Syntax Trees (ASTs) to discover:

- Declared agent structures and metadata
- Graph workflow step nodes and directional edges
- Tool invocations and required permissions
- Sub-agent delegation boundaries
- Human-in-the-loop (HITL) approval nodes
- Model calls (Gemini 1.5 Pro, Flash)

### Security Invariant: Zero Arbitrary Execution

To prevent malicious project inspection attacks, the static inspector:

- **Never invokes `go run`, `go generate`, or arbitrary build scripts**.
- Bounds the maximum number of AST nodes and files scanned to prevent recursion attacks.
- Rejects circular delegation constructs before registration.

---

## The 9 Graph Risk Dimensions

Every extracted graph undergoes structured risk analysis across 9 dimensions:

1. **Cycles**: Detects circular loops using Tarjan/DFS traversal.
2. **Delegation Depth**: Flags chains exceeding contract limits (default max depth: 3).
3. **Privileged Tools**: Flags infrastructure or payment execution tools.
4. **Write-Capable Tools**: Flags data modification or external email dispatch.
5. **External Network Access**: Identifies unrestrained HTTP/outbound network nodes.
6. **Approval Coverage**: Verifies whether write operations are gated by human approval nodes.
7. **Single-Agent Privilege Concentration**: Warns when a single sub-agent holds multiple high-risk capabilities.
8. **Cross-Data-Class Flow**: Flags movement from RESTRICTED data sources to PUBLIC sinks.
9. **Unresolved Identity**: Flags undeclared external delegation targets.

Findings are classified into `LOW`, `MEDIUM`, `HIGH`, and `CRITICAL` with actionable remediation steps.

---

## Delegation Taint Propagation

When Agent A delegates to Agent B:

- The principal authorization context propagates down the delegation chain.
- Invariant: **Delegation can attenuate privileges, never expand them**.
- If Agent A lacks `payment.execute` permission, Agent B cannot invoke `payment.execute` on Agent A's behalf, even if Agent B's standalone contract allows it.
