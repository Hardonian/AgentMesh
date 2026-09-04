# Autonomy Readiness Dimensions

Before elevating an organization from `ADVISORY` to `GUARDED_AUTOMATION` or `FULL_POLICY_AUTOMATION`, AgentMesh evaluates seven readiness criteria:

1. **Telemetry Health**: Telemetry buffer healthy, zero dropped spans, active proxy heartbeat.
2. **Rollback Drill Verified**: Successful automated rollback executed in staging within the last 30 days.
3. **SLO Defined**: Active `AgentSLO` tracking success rate, latency, and cost with $\ge 99\%$ compliance.
4. **Pre-Canary Evaluations Current**: Candidate agent evaluated against `GoldenTaskSet`.
5. **Policy Explicitly Formulated**: Strict allow/deny action lists configured.
6. **Historical Action Success Rate**: $\ge 95\%$ of previous planned actions succeeded without rollback.
7. **Healthy Fallback Proven**: Verified fallback agent active and capable of absorbing full traffic.
