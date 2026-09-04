# RB-02: Policy Enforcement Failures & Corrupted Cache

## 1. Metadata

- **Severity**: P1 (Critical — Potential Fail-Closed Blockage)
- **Target Component**: Deterministic Policy Engine, Cache Layer (`internal/policy`)
- **Relevant Alerts**: `PolicyEvaluationPanic`, `CorruptPolicyBundleDetected`, `PolicyDenySpike`

---

## 2. Symptoms

- Legitimate agent tool calls are unexpectedly blocked with HTTP 403 / `ErrPolicyDenied`.
- Log entries reporting checksum mismatch or deserialization error in policy cache.
- Proxies failing to reconcile new policy bundles from the Control Plane.

---

## 3. Immediate Triage (First 5 Minutes)

1. Verify policy engine health and recent bundle sync:

   ```bash
   agentmesh policy status --tenant=<tenant_id>
   ```

2. Test a known safe tool invocation against deterministic policy:

   ```bash
   agentmesh policy simulate --agent=<agent_id> --tool=<tool_name> --action=read
   ```

3. Check proxy logs for signature or digest validation errors:

   ```bash
   kubectl logs -l app=agentmesh-proxy -c proxy --tail=100 | grep -i "policy"
   ```

---

## 4. Root Cause Analysis

- **Case A: Malformed or Invalid Policy Bundle Deployed**
  - A recent policy update introduced a malformed rule, syntax error, or conflicting deny rule.
  - *Fix*: Roll back to Last Known Good (LKG) policy bundle immediately.
- **Case B: Corrupt In-Memory Cache or Deserialization Crash**
  - Node memory corruption or incomplete gRPC sync resulted in an unparsable bundle.
  - *Mitigation*: AgentMesh proxies fail closed (`ErrCorruptPolicy`). They do NOT fall back to permissive defaults.
- **Case C: Unannounced Upstream Contract Drift**
  - Agent updated its contract without registering new tools with the policy engine.
  - *Fix*: Author and apply contract migration or update policy rules.

---

## 5. Remediation Steps

1. **Roll back policy to previous stable version**:

   ```bash
   agentmesh policy rollback --tenant=<tenant_id> --target-version=<previous_version>
   ```

2. **Flush corrupt proxy memory cache and reload from persistent store**:

   ```bash
   agentmesh proxy fleet reload-cache --tenant=<tenant_id>
   ```

3. **If policy engine remains in degraded state, lock to LKG config**:

   ```bash
   agentmesh config force-lkg --tenant=<tenant_id>
   ```

---

## 6. Verification & Recovery Confirmation

- Execute test policy evaluation across all active agents:

  ```bash
  agentmesh policy test-all --tenant=<tenant_id>
  ```

- Confirm `agentmesh_policy_evaluations_total{decision="allow"}` returns to normal operational baseline.
- Ensure error logs report zero `ErrCorruptPolicy` or deserialization errors.
