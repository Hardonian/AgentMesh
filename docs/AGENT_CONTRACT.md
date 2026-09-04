# AgentContract Specification

## Overview

`AgentContract` is AgentMesh's foundational primitive defining an agent's identity, allowed capabilities, tool boundaries, delegation limits, budgets, and SLOs.

## Specification Schema

```yaml
apiVersion: agentmesh.dev/v1
kind: AgentContract

metadata:
  name: <string>               # Unique agent identifier
  organization: <string>       # Tenant or organization
  version: <string>            # Semantic version (e.g., 1.0.0)
  description: <string>        # Human-readable purpose
  labels: map[string]string    # Deployment metadata

identity:
  protocols: []string          # Supported protocols: ["a2a", "mcp"]
  serviceAccount: <string>     # Optional GCP / Kubernetes service account

capabilities: []string         # Advertised skills (e.g. "quote_analysis")

tools:
  allow: []string              # Permitted tool patterns (e.g. "bigquery.*")
  deny: []string               # Explicitly barred tools (e.g. "payment.execute")
  dataClassification:
    <toolName>: <string>       # Tag: PUBLIC, INTERNAL, CONFIDENTIAL, RESTRICTED

delegation:
  allow: []string              # Allowed peer agents for delegation
  deny: []string               # Forbidden peer agents
  maxDepth: <int>              # Maximum delegation depth (default: 5)

budgets:
  max_cost_per_task: <float>   # USD ceiling per task invocation
  max_tokens_per_task: <int>   # Token ceiling per task invocation
  max_tool_calls_per_task: <int>
  max_daily_cost: <float>      # Daily tenant expenditure limit

slo:
  p95_latency_ms: <int>        # Target P95 latency in milliseconds
  success_rate: <float>        # Target success rate (0.0 to 1.0)

approval:
  required_for: []string       # Tools requiring human approval
  timeout_seconds: <int>       # Approval expiry TTL
```

## Deterministic Hashing

Contracts are canonicalized and hashed via SHA-256 (`contract.Hash()`). Any modification to an agent's contract changes its hash, requiring re-registration and triggering regression evaluations.
