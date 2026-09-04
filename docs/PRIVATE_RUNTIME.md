# Private Enterprise Runtime & Offline Survivability

AgentMesh guarantees data-plane survivability and security even during control plane or SaaS outages.

## Offline Survivability
Each private proxy caches the latest signed configuration bundle locally (`CachedSignedConfig`):
1. **Uninterrupted Data Plane**: Invocations, policy evaluations, and routing decisions continue executing locally using the cached configuration.
2. **Audit & Expiry Window**: The cache retains a maximum staleness duration (default 24 hours).
3. **Fail-Closed Security**: If the control plane remains unreachable beyond the configured staleness window, the proxy enforces a fail-closed policy to prevent executing out-of-date security rules.

## Telemetry Backpressure
If the control plane telemetry ingestion endpoint becomes unreachable, the proxy buffers telemetry events in a bounded in-memory ring buffer, dropping low-priority debug metrics while strictly preserving security audit logs.
