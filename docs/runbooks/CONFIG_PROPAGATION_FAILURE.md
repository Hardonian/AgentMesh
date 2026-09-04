# Runbook: Configuration Propagation Failure

## Symptoms
- Proxies report `PARTIAL` or `DRIFTED` config synchronization status.
- Fleet convergence percentage below threshold (< 90%).

## Recovery Steps
1. Inspect proxy fleet heartbeat statuses:
   ```bash
   agentmesh proxy fleet
   ```
2. Check network connectivity between control plane and proxy cluster.
3. Validate signing key rotation status to ensure proxies possess current public keys.
