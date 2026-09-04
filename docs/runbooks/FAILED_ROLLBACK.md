# Runbook: Failed Rollback

## Severity
CRITICAL (P0)

## Symptoms
- Rollback step execution reports failure.
- Provider fails to restore prior configuration.
- Traffic remains divided or degraded.

## Immediate Action
1. Activate Emergency Kill Switch:
   ```bash
   agentmesh automation freeze "failed rollback intervention"
   ```
2. Retrieve last known good signed configuration bundle:
   ```bash
   agentmesh action show [action-id]
   ```
3. Manually push signed config directly to proxy fleet:
   ```bash
   agentmesh proxy fleet
   ```
4. Verify proxy traffic status and unfreeze only after 100% baseline health confirmed.
