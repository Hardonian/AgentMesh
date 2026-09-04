# RB-07: Database Connection Pool Limits & Failover

## 1. Metadata

- **Severity**: P1 (Critical Infrastructure)
- **Target Component**: Database Layer, Connection Pool Manager (`internal/database`)
- **Relevant Alerts**: `DatabaseConnectionPoolExhausted`, `DatabaseFailoverTriggered`, `DatabaseHighLatency`

---

## 2. Symptoms

- Control plane handlers returning HTTP 500 or timing out waiting for database connections.
- Metrics show `db_pool_open_connections` pegged at `MaxOpenConns` (e.g. 50).
- PostgreSQL primary node undergoing automated cloud failover (e.g. Google Cloud SQL or AWS Aurora).

---

## 3. Immediate Triage (First 5 Minutes)

1. Check current connection pool stats:

   ```bash
   agentmesh metrics query --metric=go_sql_stats_open_connections
   ```

2. Verify database health and reachability:

   ```bash
   agentmesh db ping
   ```

3. Check PostgreSQL lock contention and active transactions:

   ```sql
   SELECT pid, age(clock_timestamp(), query_start), usename, query, state 
   FROM pg_stat_activity 
   WHERE state != 'idle' ORDER BY query_start ASC LIMIT 10;
   ```

---

## 4. Root Cause Analysis

- **Case A: Slow Transaction / Connection Leak**
  - Unclosed rows or transactions holding database connections open.
  - *Fix*: Ensure all queries adhere to `defer rows.Close()` and context timeouts (`SetConnMaxLifetime: 5m`).
- **Case B: Sudden Spike in Control Plane Ingress**
  - Burst of concurrent requests exceeding configured `MaxOpenConns`.
  - *Fix*: Scale connection pool limits or activate read replica routing.
- **Case C: Primary Database Node Outage**
  - Primary VM down; cloud provider promoting replica to new primary.
  - *Fix*: Allow connection pool to cycle connections to new primary IP.

---

## 5. Remediation Steps

1. **Kill hanging/blocking PostgreSQL transactions**:

   ```sql
   SELECT pg_terminate_backend(pid) FROM pg_stat_activity 
   WHERE state != 'idle' AND query_start < now() - interval '30 seconds';
   ```

2. **If connection pool is undersized for load, update limits**:

   ```bash
   kubectl set env deployment/agentmesh-control-plane DB_MAX_OPEN_CONNS=100 DB_MAX_IDLE_CONNS=25
   ```

3. **If failover did not update DNS, restart control plane pods to re-resolve**:

   ```bash
   kubectl rollout restart deployment/agentmesh-control-plane
   ```

---

## 6. Verification & Recovery Confirmation

- Check `/readyz` endpoint returns HTTP 200 OK.
- Verify `db_pool_wait_duration_ms` drops to < 5ms.
- Ensure all CRUD operations against agents and policies complete successfully.
