# RB-11: Canary Abort & Rapid Rollback

## 1. Metadata

- **Severity**: P2 (Deployment Anomaly) / P1 (if canary causes customer-impacting failures)
- **Target Component**: Progressive Delivery Engine, Canary Controller (`internal/canary`, `internal/routing`)
- **Relevant Alerts**: `CanarySlaViolation`, `CanaryErrorRateSpike`, `CanaryP95LatencyBreached`

---

## 2. Symptoms

- Canary deployment (e.g. 10% traffic to Agent V2 or new Policy V3) triggers automated alerts.
- Error rate on canary traffic exceeds threshold (e.g. > 1% errors).
- P95 latency on canary version increases by > 20% compared to baseline.

---

## 3. Immediate Triage (First 2 Minutes)

1. Check real-time canary metrics vs baseline:

   ```bash
   agentmesh canary status --route=<route_id>
   ```

2. If metrics show SLA breach and auto-rollback has not triggered, abort manually:

   ```bash
   agentmesh canary abort --route=<route_id> --reason="Error rate exceeded 2% threshold"
   ```

---

## 4. How Canary Abort Works

- The Canary Controller sets canary traffic weight to 0% instantaneously.
- 100% of incoming production requests are routed immediately back to the stable baseline agent.
- In-flight requests on the canary version complete gracefully without disconnection.
- The canary version is marked `ABORTED` in the database, preventing accidental re-promotion.

---

## 5. Post-Abort Investigation

1. Collect canary execution logs and failure traces:

   ```bash
   agentmesh canary logs --route=<route_id> --version=<canary_version> --only-errors
   ```

2. Compare contract differences between baseline and canary:

   ```bash
   agentmesh contracts diff --base=<baseline_version> --candidate=<canary_version>
   ```

3. Identify whether issue was prompt hallucination, tool failure, or timeout.

---

## 6. Verification & Safe Clean-up

- Confirm 100% of traffic is served by the stable baseline.
- Verify error rates and latency return to normal baseline within 60 seconds.
- Tag the failed version in CI/CD pipeline to block future deployments until fix is merged.
