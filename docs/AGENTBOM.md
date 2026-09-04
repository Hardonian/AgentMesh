# AgentBOM: Software Bill of Materials for AI Agents

## Overview

`AgentBOM` provides an enterprise-grade Software Bill of Materials for autonomous AI agents, cataloging their runtime dependencies, foundation models, MCP tools, delegation topologies, and data classifications.

## Structure

- **Metadata**: Agent name, semantic version, organization, timestamp.
- **Agent Identity**: Runtime (`go`, `python`), framework (`google-adk`, `langgraph`, `custom`).
- **Models**: Foundation model dependencies, providers (`google-vertex`, `gemini`, `anthropic`), context window sizes, purpose.
- **Protocols**: Supported communication standards (`a2a`, `mcp`).
- **Tools**: Declared tool dependencies, risk ratings (`LOW`, `MEDIUM`, `HIGH`, `CRITICAL`), data classifications (`PUBLIC`, `INTERNAL`, `CONFIDENTIAL`, `RESTRICTED`).
- **MCPServers**: Attached Model Context Protocol servers and transports (`stdio`, `sse`, `http`).
- **Delegates**: Allowable downstream peer agents.
- **Permissions**: Required scopes and approval requirements.

## CLI Usage

```bash
# Generate AgentBOM from an AgentContract
agentmesh bom generate agent.contract.yaml > agent.bom.json
```
