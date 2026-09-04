# RB-01: High Latency & Tool Execution Delay

## 1. Metadata

- **Severity**: P2 (Warning) / P1 (if P95 > SLA threshold)
- **Target Component**: Data Plane Proxy, Routing Engine, MCP Tool Bridge
- **Relevant Alerts**: `HighLatencyWarning`, `P95LatencySlaBreach`, `ToolExecutionSlow`

---

## 2. Symptoms

- Inbound client requests experiencing latency spikes > 2000ms.
- Downstream A2A requests queueing up or timing out.
- Dashboard indicates elevated P95/P99 latency across one or more agent routes.

---

## 3. Immediate Triage (First 5 Minutes)

1. Check current P95/P99 latency by route:

   ```bash
   agentmesh metrics query --metric=agentmesh_route_duration_seconds_p95 --window=10m
   ```

2. Determine if the bottleneck is upstream model, downstream tool, or mesh proxy:

   ```bash
   agentmesh trace analyze --slowest=10 --tenant=<tenant_id>
   ```

3. Inspect whether a specific agent candidate is degraded:

   ```bash
   agentmesh routes inspect <route_id>
   ```

---

## 4. Root Cause Analysis

- **Case A: Downstream Model Latency Spike**
  - External model provider (e.g. Gemini, OpenAI) experiencing degraded inference speed.
  - *Fix*: Divert traffic to fallback provider or lighter model using route override.
- **Case B: MCP Tool Hang / Locking**
  - Tool execution database or third-party API is slow or deadlocked.
  - *Fix*: Verify tool timeout config (`DefaultToolTimeoutSeconds: 30s`). Check database connection pool contention.
- **Case C: CPU / Memory Throttling on Proxy Nodes**
  - Proxy pod CPU utilization > 85%, causing request queueing.
  - *Fix*: Scale proxy deployment horizontally (`kubectl scale deployment/agentmesh-proxy --replicas=10`).

---

## 5. Remediation Steps

1. **Divert traffic away from slow candidate**:

   ```bash
   agentmesh routes update <route_id> --demote-candidate=<slow_agent_id> --priority=99
   ```

2. **If canary version is causing latency, trigger instant abort**:

   ```bash
   agentmesh canary abort --route=<route_id> --reason="High P95 latency breach"
   ```

3. **If widespread across an MCP server, throttle tool concurrency**:

   ```bash
   agentmesh mcp update-server <server_id> --max-concurrency=10
   ```

---

## 6. Verification & Recovery Confirmation

- Verify P95 latency returns below 500ms for 5 consecutive minutes:

  ```bash
  agentmesh metrics query --metric=agentmesh_route_duration_seconds_p95 --window=5m
  ```

- Confirm zero dropped connections or HTTP 504 gateway timeouts.
- Close incident and record findings in postmortem.
