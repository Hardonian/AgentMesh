# AGENTS.md — AgentMesh Operational Intelligence Guide for AI Agents

Welcome to AgentMesh. This document provides machine-discoverable instructions for AI coding assistants, sub-agents, and tools operating within this repository.

---

## What is AgentMesh?
AgentMesh is the open control plane and intelligence layer for production AI agents communicating over **A2A (Agent-to-Agent)** and **MCP (Model Context Protocol)**. It provides identity, semantic policy, capability-aware routing, graph risk analysis, and progressive delivery.

---

## Key Machine Interfaces

### 1. AgentMesh MCP Intelligence Server
AgentMesh exposes its intelligence via a native MCP server (`agentmesh mcp serve` or via stdio):

| Tool Name | Parameters | Description |
|---|---|---|
| `inspect_agent` | `{"agentId": string}` | Retrieves an agent's registered contract, identity, and lifecycle state. |
| `get_agent_passport` | `{"agentId": string}` | Retrieves Agent Passport V2 with operational evidence, reliability, and provenance. |
| `inspect_graph` | `{"agentId": string}` | Retrieves canonical AgentGraph topology, tools, delegations, and 9-dimension risk findings. |
| `explain_policy` | `{"agentId": string, "tool": string, "action": string}` | Evaluates deterministic policy and explains why an action is ALLOWED or DENIED. |
| `simulate_route` | `{"capability": string, "strategy": string}` | Simulates multi-stage candidate ranking and returns confidence score. |
| `get_evaluation` | `{"agentId": string}` | Retrieves benchmark evaluation scorecard and regression baselines. |
| `compare_versions` | `{"agentId": string, "versionA": string, "versionB": string}` | Analyzes change impact and flags security-sensitive differences. |
| `get_tool_risk` | `{"toolName": string}` | Returns risk classification (`READ`, `WRITE`, `DESTRUCTIVE`, etc.) and schema drift status. |

---

## Troubleshooting Common Developer Questions

### "Why is my agent unable to access BigQuery or Gmail?"
1. Use `explain_policy` or `agentmesh policy simulate [policy-file]`.
2. Check if the tool is explicitly listed under `tools.deny` in the agent's contract.
3. Check if delegation taint propagation applies: if an upstream caller lacks permission for the tool, the delegated sub-agent is blocked to prevent confused deputy privilege escalation.
4. Verify tool risk class: `WRITE` or `DESTRUCTIVE` operations require explicit policy rules or approved HITL tokens.

### "How do I verify an ADK project before deploying?"
Run:
```bash
agentmesh adk graph inspect ./my-adk-agent
agentmesh adk graph validate ./my-adk-agent
```
Ensure that no cyclic dependencies exist, delegation depth does not exceed 3, and all write-capable tools are gated by approval nodes.
