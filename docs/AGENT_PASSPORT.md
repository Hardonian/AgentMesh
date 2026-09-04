# Agent Passport Specification

## Concept
The **Agent Passport** bridges declared configurations and empirical operational evidence:
> **Core Principle: Never present declared capability as proven performance.**

## Evidence Statuses
Every metric and capability in a Passport carries one of four statuses:
1. `DECLARED`: Declared in the agent's contract or manifest, not yet validated in production.
2. `INFERRED`: Preliminary evidence based on fewer than 5 task samples.
3. `MEASURED`: Empirically proven through production telemetry across a statistically significant sample count.
4. `UNKNOWN`: Unverified or missing evidence.

## Fields
- **Identity**: Agent ID, organization, version, runtime, framework.
- **Declared Claims**: Target SLO, capabilities, allowed tools, contract hash.
- **Measured Evidence**:
  - `sampleCount`: Total completed task executions.
  - `successRate`: Ratio of successful runs to total runs.
  - `p95LatencyMs`: Empirically observed P95 latency.
  - `averageCostUSD`: Mean USD expenditure per task.
  - `toolCallSuccessRate`: Success rate of downstream MCP tool invocations.
  - `policyCompliance`: Compliance status (`COMPLIANT` vs `VIOLATIONS_DETECTED`).
  - `confidenceScore`: Statistical confidence scaled by sample count (0.0 to 1.0).
- **Audit**: Issued timestamp, expiration date, issuing control plane identifier.
