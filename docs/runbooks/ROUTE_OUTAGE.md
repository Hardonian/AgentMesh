# Runbook: Route Outage & Unhealthy Agent Cascade

## Symptoms

- Rapid consecutive timeouts across multiple agents supporting a capability.
- Capability health status drops from `HEALTHY` to `DEGRADED` or `UNAVAILABLE`.

## Recovery Steps

1. Pin proven healthy fallback agent:

   ```bash
   agentmesh route explain [capability-id]
   ```

2. Enable circuit breaker to shed load on failing endpoints.
3. Verify downstream MCP tools are reachable.
