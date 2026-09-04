# Control Plane Security & Threat Model

## Threat Vectors & Defenses

1. **Approval Forgery / Stale Approval**:
   - *Defense*: Approvals are cryptographically bound to `ComputeActionHash()`. If action parameters are altered after approval, execution is rejected (`ErrStaleApproval`).
2. **Cross-Tenant Mutation**:
   - *Defense*: Row-level tenant isolation enforces strict verification of `organization_id`. Cross-tenant updates fail closed with 403 Forbidden.
3. **Emergency Kill Switch Tampering**:
   - *Defense*: Freeze state is evaluated server-side on every execution attempt. When frozen, all automated mutations are rejected immediately.
4. **Provider Credential Revocation**:
   - *Defense*: If cloud provider credentials (GKE, Cloud Run) fail validation, providers fail closed without corrupting configuration state.
5. **Config Tampering**:
   - *Defense*: Proxies reject any route configuration with mismatched HMAC or broken previous-config hash chains.
