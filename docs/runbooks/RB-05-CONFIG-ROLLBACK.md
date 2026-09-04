# RB-05: Configuration Rollback & LKG Restoration

## 1. Metadata

- **Severity**: P1 (Critical Operation)
- **Target Component**: Control Plane Configuration Engine, Bundle Distributor (`internal/config`)
- **Relevant Alerts**: `ConfigPropagationFailure`, `ConfigBundleRejected`, `InvalidConfigSignature`

---

## 2. Symptoms

- Proxies rejecting newly published configuration bundles.
- Control Plane pushed breaking routing rules, invalid policies, or incorrect tool definitions.
- Proxies operating on stale or mismatched configuration states across the fleet.

---

## 3. Immediate Triage (First 5 Minutes)

1. Verify the current active bundle version across all proxies:

   ```bash
   agentmesh proxy fleet versions --tenant=<tenant_id>
   ```

2. Identify why proxies rejected the bundle:

   ```bash
   agentmesh config status --show-errors
   ```

3. Locate the Last Known Good (LKG) bundle in the local cache or persistent database:

   ```bash
   agentmesh config list-history --tenant=<tenant_id> --limit=5
   ```

---

## 4. Root Cause Analysis

- **Monotonic Version Invariant**: AgentMesh proxies reject any bundle with `version <= current_version` or `issued_at < current_issued_at`.
- Therefore, a rollback **cannot** simply be a replay of the raw old bundle bytes (which would be rejected as a replay attack).
- A rollback **must** generate a *NEW* signed bundle with a monotonically *higher* version number and a current timestamp, containing the exact payload of the LKG state.

---

## 5. Remediation Steps

1. **Execute Monotonic LKG Rollback via CLI**:

   ```bash
   agentmesh config rollback --tenant=<tenant_id> --to-version=<target_lkg_version>
   ```

   *Note*: The Control Plane reads the target LKG state, increments the global version counter (`new_version = current_version + 1`), signs with the active Ed25519 key, and broadcasts.

2. **If Control Plane is down, force proxies to boot from local LKG cache**:

   ```bash
   agentmesh proxy fleet set-flag --use-local-lkg=true
   ```

3. **Verify proxy acknowledgement across all nodes**:

   ```bash
   agentmesh proxy fleet status --wait-for-sync --timeout=30s
   ```

---

## 6. Verification & Recovery Confirmation

- Verify 100% of proxy fleet reports the newly deployed rollback version.
- Ensure zero `ErrReplayDetected` or `ErrSignatureInvalid` logs.
- Confirm end-to-end routing and tool dispatch functions normally.
