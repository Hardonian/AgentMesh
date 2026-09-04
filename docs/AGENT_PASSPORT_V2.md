# Agent Passport V2 Specification

The **Agent Passport V2** (`pkg/passport/passport.go`) upgrades the Phase 1 passport to a multi-dimensional operational certificate detailing an agent's identity, protocols, graph topology, compatibility, empirical reliability, economics, safety, evaluations, deployment state, and section provenance.

---

## The 10 Core Sections

```
┌────────────────────────────────────────────────────────────┐
│                    Agent Passport V2                       │
├────────────────────────────┬───────────────────────────────┤
│ 1. Identity                │ Agent ID, Version, Runtime    │
│ 2. Protocol                │ A2A & MCP Compliance          │
│ 3. Graph                   │ Canonical Graph Topology Hash │
│ 4. Compatibility           │ A2A & MCP Profiles            │
│ 5. Reliability             │ Task Success, Latency, MTBF   │
│ 6. Economics               │ Cost / Task, Token Spend      │
│ 7. Safety                  │ Policy Coverage, Approvals    │
│ 8. Evaluations             │ Baseline Benchmark Scores     │
│ 9. Deployment              │ Active Revisions, Canary Stg  │
│ 10. Provenance             │ DECLARED / MEASURED / OBSERVED│
└────────────────────────────┴───────────────────────────────┘
```

---

## Section Provenance
Every field exposes its provenance tier:
- `DECLARED`: Self-reported by developer in AgentContract.
- `MEASURED`: Verified in automated CI evaluation suites.
- `OBSERVED`: Aggregated from real production proxy telemetry.
- `INFERRED`: Derived via heuristic analysis.

---

## Public vs Private Passport Sanitization
By default, all Agent Passports are **PRIVATE**.

When an organization publishes an agent to an external directory, AgentMesh invokes `passport.SanitizeForPublic()`:
- **Redacts**: Private endpoint URLs, internal tenant IDs, restricted tools, sensitive policy rules, and internal graph node identifiers.
- **Retains**: Verifiable name, version, protocol conformance status, capability badges, and public evaluation benchmarks.

---

## Verifiable Badges
Generate status badges via CLI:
```bash
agentmesh badge procurement-agent
```
Produces an ASCII or SVG badge certifying:
`AgentMesh Passport: PASS | A2A: COMPATIBLE | Policy: GOVERNED`
