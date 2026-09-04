# RB-08: Server-Side Request Forgery (SSRF) Incident & Ingress Violation

## 1. Metadata

- **Severity**: P1 (Security Incident)
- **Target Component**: A2A Endpoint Validator, Outbound HTTP Transport (`internal/a2a`, `internal/server`)
- **Relevant Alerts**: `SSRFBlockedAttempt`, `PrivateIPAccessBlocked`, `CloudMetadataAccessAttempt`

---

## 2. Symptoms

- Security alert fired: Agent or external caller attempted to register or invoke a remote endpoint targeting private IP space (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `127.0.0.1/8`, or `169.254.169.254`).
- Error log: `ErrSSRFDetected` or `remote URL resolves to forbidden private IP address`.

---

## 3. Immediate Triage (First 5 Minutes)

1. Inspect the offending request and payload:

   ```bash
   agentmesh audit logs --filter="event=ssrf_blocked" --last=30m
   ```

2. Identify the originating tenant, agent ID, and API key:

   ```bash
   agentmesh audit inspect-event <event_id>
   ```

3. Determine whether the attempt was a misconfiguration or an active exploit.

---

## 4. Invariant Verification

AgentMesh enforces strict SSRF protections in `ValidateSafeRemoteURL`:

- DNS resolution is performed prior to connection.
- Target IP is evaluated against `net.IP.IsPrivate()`, `net.IP.IsLoopback()`, and link-local ranges (including AWS/GCP metadata `169.254.169.254`).
- Non-HTTP/HTTPS schemes (e.g. `file://`, `gopher://`) are immediately rejected.
- All requests targeting forbidden IPs fail closed.

---

## 5. Remediation & Containment Steps

1. **Revoke the compromised or offending API key immediately**:

   ```bash
   agentmesh credentials revoke --id=<credential_id> --reason="Security violation: SSRF attempt"
   ```

2. **If internal malicious agent detected, suspend agent execution**:

   ```bash
   agentmesh agents suspend --id=<agent_id> --tenant=<tenant_id>
   ```

3. **Verify egress firewall rules on GKE/Cloud Run nodes**:
   - Ensure network policy drops egress from proxy pods to internal cloud metadata endpoints if not already enforced.

---

## 6. Verification & Post-Incident Actions

- Confirm the target endpoint was blocked and zero outbound HTTP connections were established to the private IP.
- File security incident report and update SIEM / SOC dashboard.
