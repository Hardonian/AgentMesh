# Show HN: AgentMesh – An open Go control plane for A2A and MCP agents

**Title**: Show HN: AgentMesh – An open Go control plane for A2A and MCP agents  
**URL**: <https://github.com/agentmesh/agentmesh>  

---

## Body

Hi HN,

We've been building and operating autonomous AI agents in production, and we ran into a recurring infrastructure wall:

Once agents start delegating tasks to other agents (via A2A), invoking external tool servers (via Model Context Protocol / MCP), and querying multimodal models, traditional infrastructure breaks down:

- Who is allowed to call what tool?
- How do you prevent a low-privilege agent from tricking a high-privilege agent into running destructive queries (confused deputy problem)?
- What happens when a model provider is throttled or a downstream agent hangs?
- How do you safely canary an agent update when prompt changes can subtly degrade tool accuracy?

Generic API gateways (Envoy, Kong) don't understand agent identities, delegation chains, or MCP schemas. Agent application frameworks (LangGraph, CrewAI) help you *build* an agent loop, but they aren't an operations control plane for multi-agent fleets.

So we built **AgentMesh**.

AgentMesh is an open-source, Go-native control plane and data-plane proxy that sits between agents, tools, and model providers:

1. **Deterministic Policy Gate**: Evaluates typed rules (`ALLOW`, `DENY`, `REQUIRE_APPROVAL`) in compiled Go with sub-millisecond latency. Zero LLM hallucinations in the authorization path.
2. **Cryptographic Parameter Binding**: Human-in-the-loop (HITL) approval tokens bind directly to `sha256(canonical_json(params))` and expire single-use, stopping parameter tampering or replay attacks.
3. **Capability-Aware Routing**: Evaluates agent health, historical P95 latency against SLOs, and token budgets to route tasks to the best eligible candidate with automated fallback.
4. **Delegation Invariants**: Enforces strict call stack depth limits (max 5 hops) and ordered cycle detection to stop runaway recursive agent loops.
5. **Built for Failure**: The data plane edge proxy uses an in-memory cache of the Last Known Good (LKG) signed Ed25519 config. If the control plane goes down, the proxy continues routing and enforcing policy safely.
6. **Zero-Leak Telemetry**: Native OpenTelemetry trace propagation with automated regex scrubbing for API keys, bearer tokens, and passwords.

You can run the full deterministic multi-agent demo locally in under 60 seconds with zero cloud setup:

```bash
git clone https://github.com/agentmesh/agentmesh.git
cd agentmesh
go build -o bin/agentmesh ./cmd/agentmesh
./bin/agentmesh demo run
```

The repo is licensed under Apache-2.0. We also have full STRIDE threat models, 11 SRE operational runbooks, and automated adversarial red team test suites in the repository.

We would love to hear feedback from engineers running agents or MCP servers in production.

GitHub: <https://github.com/agentmesh/agentmesh>
