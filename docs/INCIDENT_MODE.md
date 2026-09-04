# Incident Mode & Emergency Governance

When a capability breaches its error budget or enters an incident state:

1. **Non-Critical Optimization Frozen**: All cost-reduction mutations and traffic shifts are halted.
2. **Safe Exploration Disabled**: Canary rollouts and experimental routes are suspended.
3. **Reliability-First Routing**: Traffic is pinned to the verified `LastKnownGoodRoute`.
4. **Emergency Failover Permitted**: If primary agent is unresponsive, automatic fallback executes under policy.
5. **Emergency Tool Block**: Operators can issue fast signed policy updates to block failing downstream MCP tools.
