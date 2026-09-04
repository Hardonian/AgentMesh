# AGENTS.md — AgentMesh Operational Intelligence Guide for AI Agents

Welcome to AgentMesh. This document provides machine-discoverable instructions for AI coding assistants, sub-agents, and autonomous tools operating within this repository.

---

## What is AgentMesh?

AgentMesh is the open control plane and data-plane proxy for production AI agents communicating over **A2A (Agent-to-Agent)** and **MCP (Model Context Protocol)**. It provides identity, semantic policy, capability-aware routing, graph risk analysis, and progressive delivery.

---

## Key Machine Interfaces

### 1. AgentMesh MCP Intelligence Server

AgentMesh exposes its intelligence via a native MCP server (`agentmesh mcp serve` or via stdio):

| Tool Name | Parameters | Description |
| --- | --- | --- |
| `inspect_agent` | `{"agentId": string}` | Retrieves an agent's registered contract, identity, and lifecycle state. |
| `get_agent_passport` | `{"agentId": string}` | Retrieves Agent Passport V2 with operational evidence, reliability, and provenance. |
| `inspect_graph` | `{"agentId": string}` | Retrieves canonical AgentGraph topology, tools, delegations, and 9-dimension risk findings. |
| `explain_policy` | `{"agentId": string, "tool": string, "action": string}` | Evaluates deterministic policy and explains why an action is ALLOWED or DENIED. |
| `simulate_route` | `{"capability": string, "strategy": string}` | Simulates multi-stage candidate ranking and returns confidence score. |
| `get_evaluation` | `{"agentId": string}` | Retrieves benchmark evaluation scorecard and regression baselines. |
| `compare_versions` | `{"agentId": string, "versionA": string, "versionB": string}` | Analyzes change impact and flags security-sensitive differences. |
| `get_tool_risk` | `{"toolName": string}` | Returns risk classification (`READ`, `WRITE`, `DESTRUCTIVE`, etc.) and schema drift status. |

---

## Contributor Agent Rules & Non-Negotiable Invariants

When contributing code or modifying this repository, AI agents must adhere to the following rules:

1. **Deterministic Policy is Authoritative**:
   - Never replace compiled Go policy evaluation with prompt-based or LLM-based authorization gates.
   - All authorization decisions (`ALLOW`, `DENY`, `REQUIRE_APPROVAL`) must be deterministic, transparent, and reproducible.
2. **Fail-Closed Multi-Tenant Isolation**:
   - Never query or mutate database tables without an authenticated `tenant_id`.
   - All store listing methods must return `ErrEmptyTenant` if `tenant_id == ""`.
3. **No Unbounded Memory Allocations**:
   - Every deserialization interface (YAML, JSON) must enforce a pre-read length check (max 10MB) before parsing (`MaxContractPayloadBytes`).
4. **Zero Secrets in Telemetry**:
   - All logs, metric labels, and traces must pass through the `internal/telemetry` secret scrubber regex suite before output.
5. **Windows Memory & Compilation Constraint**:
   - Always run Go tests with `-p 1` (e.g. `go test -p 1 ./...`) to prevent Windows paging file exhaustion.
6. **No Data Races**:
   - All code must execute cleanly with zero warnings under `go test -p 1 -race ./...` and `go vet ./...`.

---

## Standard Verification Commands

```bash
# Build binary
go build -o bin/agentmesh ./cmd/agentmesh

# Run deterministic demo
./bin/agentmesh demo run

# Run full Definition-of-Done and Red Team verification suites
go test -p 1 -v ./tests -run "TestP0RedTeamScenarios|TestPhase5DefinitionOfDone35Certifications"

# Run linter and compiler static checks
go vet ./...
```
