# Post-Launch Monitoring, Telemetry & Observability Guide — AgentMesh

## 1. Service Level Objectives (SLOs) & Indicators (SLIs)

| Category | Service Level Indicator (SLI) | Target SLO | Measurement Window | Action on Breach |
| --- | --- | --- | --- | --- |
| **Availability** | % of successful HTTP responses (non-5xx) | **99.95%** | Rolling 30 days | Alert P1, initiate failover |
| **Proxy Latency** | P95 proxy routing overhead | **< 15ms** | Rolling 5 minutes | Alert P2, check CPU/memory |
| **Policy Latency** | P99 deterministic policy evaluation time | **< 2ms** | Rolling 5 minutes | Alert P2, inspect rule count |
| **A2A Reliability** | % of completed multi-agent handshakes | **99.9%** | Rolling 1 hour | Alert P1, check cycle limits |
| **Security** | Zero unredacted secrets in logs/traces | **100% (Zero Leak)** | Continuous | Alert P0, trigger `RB-09` |
| **Isolation** | Zero cross-tenant data leakage | **100% (Zero Leak)** | Continuous | Alert P0, engage kill switch |

---

## 2. Core Prometheus Metrics

### 2.1. Routing & Proxy Metrics

- `agentmesh_proxy_requests_total{tenant, status, route}`: Total inbound proxy requests.
- `agentmesh_proxy_request_duration_seconds{tenant, route}`: End-to-end request duration histogram.
- `agentmesh_proxy_overhead_duration_seconds`: Isolated proxy processing overhead (excluding tool/model time).
- `agentmesh_route_failures_total{tenant, reason}`: Route failures (e.g. timeout, no healthy candidate).

### 2.2. Policy & Security Metrics

- `agentmesh_policy_evaluations_total{tenant, decision="allow|deny"}`: Deterministic policy outcomes.
- `agentmesh_policy_evaluation_duration_seconds`: Policy engine execution time.
- `agentmesh_approval_requests_total{tenant, status}`: HITL approval counts (pending, approved, rejected, expired).
- `agentmesh_security_events_total{type="ssrf_blocked|tampering|privilege_escalation"}`: Security violation counter.

### 2.3. System & Database Health

- `agentmesh_db_connections_open`: Active database connections in pool.
- `agentmesh_db_connections_in_use`: Connections currently performing SQL queries.
- `agentmesh_config_bundle_version{tenant}`: Current active configuration bundle version.

---

## 3. Recommended Prometheus Alerting Rules

```yaml
groups:
  - name: agentmesh-production-alerts
    rules:
      - alert: AgentMeshHighErrorRate
        expr: sum(rate(agentmesh_proxy_requests_total{status=~"5.."}[5m])) / sum(rate(agentmesh_proxy_requests_total[5m])) > 0.01
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "AgentMesh proxy error rate > 1%"
          runbook_url: "docs/runbooks/RB-01-HIGH-LATENCY.md"

      - alert: AgentMeshP95LatencyBreach
        expr: histogram_quantile(0.95, sum(rate(agentmesh_proxy_overhead_duration_seconds_bucket[5m])) by (le)) > 0.05
        for: 3m
        labels:
          severity: warning
        annotations:
          summary: "AgentMesh proxy P95 overhead > 50ms"
          runbook_url: "docs/runbooks/RB-01-HIGH-LATENCY.md"

      - alert: AgentMeshSSRFAttemptDetected
        expr: increase(agentmesh_security_events_total{type="ssrf_blocked"}[5m]) > 0
        labels:
          severity: critical
        annotations:
          summary: "SSRF attempt blocked on AgentMesh endpoint"
          runbook_url: "docs/runbooks/RB-08-SSRF-INCIDENT.md"

      - alert: AgentMeshAutomationFrozen
        expr: agentmesh_automation_frozen_status == 1
        labels:
          severity: warning
        annotations:
          summary: "Tenant or Mesh automation has been frozen by kill switch"
          runbook_url: "docs/runbooks/RB-10-EMERGENCY-AUTOMATION-FREEZE.md"
```

---

## 4. Synthetic Probing & Health Checks

- Continuous synthetic blackbox probing every 30 seconds across all regions:
  - `GET /healthz` (liveness probe)
  - `GET /readyz` (readiness probe)
  - `POST /api/v1/policies/simulate` with standard test agent payload.
- Any 2 consecutive synthetic probe failures trigger immediate pager duty escalation to the on-call SRE.
