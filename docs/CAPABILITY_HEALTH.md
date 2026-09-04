# Capability Health & Availability

Enterprises operate around capabilities (e.g. `financial_analysis`, `customer_support`) rather than individual agent instances. AgentMesh abstracts agent health into high-level capability availability.

## Status Classifications

- `HEALTHY`: All eligible agents supporting this capability are healthy and compliant with SLOs.
- `DEGRADED`: One or more candidate agents are breached or in incident mode, but at least one healthy candidate remains available to serve traffic.
- `UNAVAILABLE`: Zero healthy agents are available to serve this capability.

## CLI & MCP Inspection

```bash
agentmesh capability health [capability-id]
```

The response summarizes:

- Total registered agents
- Healthy vs breached agents
- Aggregate P95 latency and average cost
- Current capability operational status
