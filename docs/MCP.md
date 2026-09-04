# Model Context Protocol (MCP) Gateway & MCPGuard

## Overview
AgentMesh functions as a reverse proxy gateway sitting between MCP clients (agents) and MCP servers:

```
MCP Client (Agent) ──► AgentMesh MCP Gateway ──► Upstream MCP Tool Server
```

## Gateway Responsibilities
1. **Tool Discovery (`tools/list`)**: Filters advertised tools based on the caller's active policy. Tools the agent is barred from executing are hidden from discovery.
2. **Policy Enforcement (`tools/call`)**: Validates execution requests against deterministic policy before reaching the upstream tool server.
3. **MCPGuard Subsystem**:
   - Risk Classifications: `LOW`, `MEDIUM`, `HIGH`, `CRITICAL`.
   - Data Classifications: `PUBLIC`, `INTERNAL`, `CONFIDENTIAL`, `RESTRICTED`.
   - Argument Validation: Restricts execution to approved parameters (e.g. BigQuery project/dataset allowlists).
4. **Reliability & Rate Limiting**:
   - Circuit breakers protect failing upstream servers.
   - Token-bucket rate limiting prevents runaway tool loops.
5. **Human-in-the-Loop Interception**: Sensitive operations return `-32002 (MCPApprovalRequired)` and queue an approval request.
