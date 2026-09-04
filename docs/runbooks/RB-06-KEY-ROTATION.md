# RB-06: Cryptographic Key Rotation Without Downtime

## 1. Metadata

- **Severity**: P2 (Routine Operation) / P1 (Emergency Revocation)
- **Target Component**: Cryptographic Subsystem, Key Registry (`internal/crypto`, `internal/config`)
- **Relevant Alerts**: `KeyExpirationWarning`, `KeyRevoked`, `UntrustedSignatureDetected`

---

## 2. Symptoms

- Scheduled key rotation window reached (every 90 days recommended).
- OR: Potential compromise of an Ed25519 signing private key or mTLS CA.
- Need to rotate root or tenant keys without dropping active A2A calls or rejecting valid bundles.

---

## 3. Immediate Triage & Pre-Checks

1. Check current active and trusted keypairs:

   ```bash
   agentmesh crypto keys list
   ```

2. Verify key health and expiration timestamps:

   ```bash
   agentmesh crypto keys status --key-id=<current_key_id>
   ```

---

## 4. Key Rotation Architecture (Dual-Key Overlap)

AgentMesh uses a dual-verification model during rotation:

1. **Phase 1: Pre-publish New Public Key**
   - The new public key `Key_B` is registered in the trusted key registry across all proxies while `Key_A` remains the active signing key.
   - Proxies now trust signatures from BOTH `Key_A` and `Key_B`.
2. **Phase 2: Switch Active Signing Key**
   - The Control Plane begins signing new bundles using `Key_B`.
   - Any proxy already has `Key_B` in its trusted ring, so verification succeeds instantly with 0ms downtime.
3. **Phase 3: Retire Old Key**
   - After 24-48 hours (or after all proxies have acknowledged bundle signed by `Key_B`), `Key_A` is marked revoked or retired.

---

## 5. Execution Steps

1. **Generate new Ed25519 Keypair in KMS / Vault**:

   ```bash
   agentmesh crypto keys generate --algorithm=ed25519 --label="control-plane-2026-q4"
   ```

2. **Distribute new public key to proxy trust ring**:

   ```bash
   agentmesh crypto keys register-trusted --public-key-file=new_key.pub --key-id="cp-2026-q4"
   ```

3. **Wait for proxy ring sync confirmation**:

   ```bash
   agentmesh proxy fleet verify-trusted-key --key-id="cp-2026-q4"
   ```

4. **Switch signing authority**:

   ```bash
   agentmesh config set-active-signing-key --key-id="cp-2026-q4"
   ```

5. **If Emergency Revocation (Key Compromise)**:
   - Revoke compromised key immediately:

     ```bash
     agentmesh crypto keys revoke --key-id="cp-compromised-id" --reason="Key compromise"
     ```

   - All proxies immediately invalidate any bundles signed by the revoked key.

---

## 6. Verification & Recovery Confirmation

- Publish a test canary bundle signed with the new key.
- Verify 100% of proxy fleet verifies and accepts the signature:

  ```bash
  agentmesh proxy fleet status --check-signing-key
  ```

- Ensure zero `ErrSignatureInvalid` errors in proxy logs.
