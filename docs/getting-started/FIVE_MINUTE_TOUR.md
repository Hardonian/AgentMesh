# Five-Minute Tour: Local AgentMesh Hands-On

Welcome to AgentMesh. This guide walks you through the core workflow of AgentMesh in less than five minutes on your local machine.

**Prerequisites**: Go 1.22+ installed. **No cloud credentials or external databases required.**

---

## Step 1: Initialize Your Workspace & Run Doctor

Verify your development environment:

```bash
agentmesh doctor
```

You will see a diagnostic report showing that your local Go environment and AgentMesh CLI are ready.

---

## Step 2: Initialize an AgentContract

Generate a sample `AgentContract` for a financial research agent:

```bash
agentmesh init
```

This generates `agentmesh.yaml`:

```yaml
apiVersion: agentmesh.dev/v1
kind: AgentContract
metadata:
  name: my-first-agent
  version: "1.0.0"
capabilities:
  - financial-research
tools:
  allow:
    - bigquery.read
  deny:
    - raw_sql.execute
delegation:
  maxDepth: 3
budget:
  maxCostPerTaskMicroUSD: 50000
```

Validate the contract against AgentMesh schema invariants:

```bash
agentmesh contract validate agentmesh.yaml
```

Output:

```text
✓ Contract 'my-first-agent:1.0.0' is valid.
```

---

## Step 3: Author and Simulate a Deterministic Policy

Create a sample policy file `policy.yaml`:

```yaml
rules:
  - id: POL-01
    effect: allow
    agent: my-first-agent
    tool: bigquery.read

  - id: POL-02
    effect: approval
    agent: my-first-agent
    tool: payments.transfer
    reason: "Funds transfers require human operator approval token"
```

Simulate tool calls against the policy:

```bash
agentmesh policy simulate --agent=my-first-agent --tool=bigquery.read
# Output: Decision: ALLOW (Rule: POL-01)

agentmesh policy simulate --agent=my-first-agent --tool=payments.transfer
# Output: Decision: REQUIRE_APPROVAL (Rule: POL-02)
```

---

## Step 4: Simulate Capability-Aware Routing

Evaluate which agent is selected for a capability:

```bash
agentmesh route simulate --capability=financial-research
```

AgentMesh evaluates candidate agents, filters out any agent lacking required tool permissions, checks P95 SLA reliability, and selects the most cost-effective candidate.

---

## Step 5: Run the Complete End-to-End Demonstration

To see multi-agent delegation, A2A protocol handshakes, MCP tool dispatch, and OpenTelemetry trace generation in one command:

```bash
agentmesh demo run
```

Congratulations! You have completed the five-minute tour.

- To explore production deployment with Google Cloud, see [Google Quickstart](../google/QUICKSTART.md).
- To read the architectural security analysis, see [Threat Model](../THREAT_MODEL.md).
