# AgentMesh Visual Studio Code & Google Cloud Shell Extension

Developer tool for Google ADK, A2A, and MCP agent systems.

## Features

- **ADK Graph Inspection**: Statically inspects local Go ADK projects and renders interactive `AgentGraph` topologies directly in VS Code.
- **Confused Deputy Protection**: Simulates semantic policies across multi-hop delegation chains to immediately alert developers of privilege escalation risks.
- **Automated LLM Red-Teaming**: Executes pre-canary adversarial probes (`agentmesh.evalRedTeam`) directly from the IDE.
- **Environment Diagnostics**: Live checks via `agentmesh doctor` for GCP ADC, Kubernetes contexts, and local development status.

## Installation

### Local VS Code

```bash
cd tools/vscode-extension
npm install
npm run package # or press F5 in VS Code to launch the Extension Development Host
```

### Google Cloud Shell

Install directly into Cloud Shell Editor using the `.vsix` bundle or open the project folder in Cloud Shell.

## Commands

- `AgentMesh: Inspect ADK AgentGraph` (`agentmesh.inspectGraph`)
- `AgentMesh: Simulate Policy & Check Confused Deputy` (`agentmesh.simulatePolicy`)
- `AgentMesh: Run Operational Diagnostics` (`agentmesh.diagnoseAgent`)
- `AgentMesh: Run Adversarial Red-Team Evaluator` (`agentmesh.evalRedTeam`)
