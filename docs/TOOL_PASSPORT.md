# Tool Passport Specification

The **Tool Passport** (`internal/mcp/types.go`) tracks empirical reliability, latency, risk classification, and schema lineage for every MCP tool governed by AgentMesh.

---

## Specification Schema

```json
{
  "toolId": "bigquery.read",
  "toolName": "BigQuery Read Tool",
  "provider": "google-cloud-managed-mcp",
  "server": "mcp-bigquery-europe-west1",
  "riskClass": "READ",
  "schemaFingerprint": "d9e810a42f5342a19b48c772e0a293817f0932c64e81fa214309a90184b2cd5e",
  "healthStatus": "HEALTHY",
  "sampleCount": 14200,
  "successRate": 0.999,
  "p95LatencyMs": 310,
  "lastEvaluated": "2026-09-03T18:30:00Z"
}
```

---

## Key Fields & Lifecycle
- **`riskClass`**: Governs whether invocations require administrative pre-approval, MFA, or read-only confinement.
- **`schemaFingerprint`**: SHA-256 digest of normalized input schema. Any change invalidates cached agent tool bindings.
- **`healthStatus`**: `HEALTHY`, `DEGRADED`, or `UNHEALTHY` driven by circuit breaker probes.
- **`sampleCount` & `successRate`**: Empirical reliability scores informing capability router candidate ranking.
