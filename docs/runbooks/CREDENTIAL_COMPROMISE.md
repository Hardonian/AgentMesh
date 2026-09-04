# Runbook: Credential Compromise & Rotation

## Symptoms

- Compromise of API keys, proxy tokens, or provider service accounts.
- Suspicious cross-tenant invocation patterns.

## Immediate Action

1. Revoke affected credential:

   ```bash
   agentmesh auth oidc-verify [token]
   ```

2. Execution providers fail closed automatically when credentials are revoked.
3. Rotate signing keys:

   ```bash
   agentmesh doctor
   ```

4. Issue new Workload Identity tokens for GKE and Cloud Run service accounts.
