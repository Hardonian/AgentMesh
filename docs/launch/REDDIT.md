# Reddit Launch Brief (r/golang, r/LocalLLaMA, r/MachineLearning)

**Suggested Title**: We built an open-source Go control plane & proxy for A2A and MCP AI agents (AgentMesh v1.0)

---

## Body

Hey everyone,

Over the past year, we noticed that while there are dozens of Python frameworks for writing agent prompts and loops, the production networking and governance layer for agents is almost completely missing.

When an agent talks to another agent using the A2A protocol or calls tools over Model Context Protocol (MCP), you need serious infrastructure:

- You can't rely on the LLM to decide whether it's allowed to drop a database or make a wire transfer.
- You can't let recursive delegation loops run up thousands of dollars in token bills.
- You need distributed tracing that tracks caller -> intermediary agent -> MCP tool -> LLM completion, without logging cleartext API keys.

We wrote **AgentMesh** in Go because we needed single static binaries, high concurrency, low GC pause times, and sub-15ms proxy overhead.

### What it does

- **Deterministic Policy**: Compiles declarative YAML rules into in-memory Go evaluators. Supports `ALLOW`, `DENY`, and `REQUIRE_APPROVAL` (with one-time cryptographic parameter-bound approval tokens).
- **A2A Handshake & Delegation Stack**: Tracks caller stacks, prevents circular delegation loops (`A -> B -> A`), and stops confused deputy privilege escalation.
- **MCP Tool Governance**: Manages stdio and SSE MCP servers, checks for breaking JSON schema drift, and classifies tool risk (`READ`, `WRITE`, `DESTRUCTIVE`).
- **Capability-Aware Routing**: Scores available agents by SLA health, P95 latency, and token cost.
- **Fail-Closed Isolation**: PostgreSQL 16+ Row-Level Security across all multi-tenant tables.
- **Progressive Delivery**: Canary weighted splits (1% to 100%) with automated rollback if error rate or P95 latency regresses.

Everything runs locally in a self-contained binary with zero cloud credentials:

```bash
git clone https://github.com/agentmesh/agentmesh.git
cd agentmesh
go build -o bin/agentmesh ./cmd/agentmesh
./bin/agentmesh demo run
```

Repository: <https://github.com/agentmesh/agentmesh>  
License: Apache-2.0

Looking forward to your thoughts, feedback, and critiques on our Go architecture and data-plane design!
