# Production Operations & SRE Guide — AgentMesh

## 1. Architecture & Deployment Topologies

AgentMesh is deployed as a cloud-native, distributed system composed of two primary tiers:

```text
[ Ingress / Cloud Load Balancer (HTTPS / TLS 1.3 / mTLS) ]
                       |
         +-------------+-------------+
         v                           v
  +--------------+            +--------------+
  |  Data Plane  |            | Control Plane|
  |  Proxy Fleet |            |   Server     |
  | (Edge/Stateless)          | (Stateful)   |
  +--------------+            +--------------+
         |                           |
         | <--- Ed25519 Bundles ---- |
         v                           v
[ Target Agents / MCP Tools ]   [ Cloud SQL / PostgreSQL 16+ ]
                                [ Redis Cluster (Cache/LKG)  ]
```

### 1.1. Data Plane Proxy Tier

- **Characteristics**: Horizontally scalable, stateless, latency-critical (< 10ms proxy overhead).
- **Placement**: Multi-region Kubernetes (GKE / EKS) or serverless container runtimes (Cloud Run).
- **Scaling Rule**: Target 60% CPU utilization. Scale up at > 75% or when connection count > 2000 per pod.

### 1.2. Control Plane Tier

- **Characteristics**: Highly available active-active cluster, transactional consistency for contracts, policies, and HITL approvals.
- **Placement**: Regional multi-zone Kubernetes deployment behind internal load balancer.
- **Database**: PostgreSQL 16+ with Row-Level Security (RLS) enabled, managed replication, and automated failover.

---

## 2. High Availability & Fault Tolerance

1. **Failure of Control Plane**:
   - Data plane proxies continue operating autonomously using their local in-memory configuration cache and Last Known Good (LKG) snapshot.
   - Zero disruption to active agent routing, tool policy evaluation, or A2A handshakes.
2. **Database Partition / Degradation**:
   - Control plane reads fall back to local read replicas.
   - Cache layer serves cached passports and contracts.
3. **External Model Provider Outage**:
   - Capability-aware router detects provider failure within 3 heartbeats (< 15s) and routes traffic to registered fallback model providers.

---

## 3. Capacity Planning & Resource Limits

| Resource | Default / Minimum | Recommended Production | Max Tested Limit |
| --- | --- | --- | --- |
| Proxy Pod CPU | 1 vCPU | 4 vCPU | 16 vCPU |
| Proxy Pod Memory | 512 MB | 2 GB | 8 GB |
| Control Plane CPU | 2 vCPU | 8 vCPU | 32 vCPU |
| Control Plane Memory | 1 GB | 4 GB | 16 GB |
| DB Connection Pool | 10 conns | 50 conns (max 100) | 250 conns |
| Max Request Body | 10 MB | 10 MB | 10 MB (`http.MaxBytesReader`) |
| Max Contract Size | 10 MB | 10 MB | 10 MB (`MaxContractPayloadBytes`) |
| Max Delegation Depth | 5 hops | 3–5 hops | 5 hops |

---

## 4. Backup, Disaster Recovery & Failover

1. **Database Automated Backups**:
   - Daily automated full backups with point-in-time recovery (PITR) enabled for 30 days.
   - RPO (Recovery Point Objective): < 5 minutes.
   - RTO (Recovery Time Objective): < 15 minutes.
2. **Configuration Bundle Archival**:
   - Every published configuration bundle is archived to immutable object storage (GCS / S3) with Ed25519 signature.
3. **Disaster Recovery Restoration Procedure**:
   - In case of catastrophic region failure, spin up proxy and control plane pods in secondary region, restore DB from PITR snapshot, and broadcast latest signed bundle.

---

## 5. Security & Maintenance Windows

- **Key Rotation**: Dual-key overlap key rotation executed every 90 days following `RB-06-KEY-ROTATION.md`. Zero downtime.
- **Security Patching**: Base container images (`distroless/static-debian12`) rebuilt and scanned weekly. Zero CVEs required for production deployment.
- **Health Probes**:
  - Liveness: `GET /healthz` (checks process liveness; failure restarts pod).
  - Readiness: `GET /readyz` (checks DB connection and bundle cache; failure removes pod from load balancer pool).
