# RB-10: Emergency Automation Freeze & Kill Switch

## 1. Metadata

- **Severity**: P0 / P1 (Emergency Intervention)
- **Target Component**: Autonomous Control Loop, Control Plane Handlers (`internal/server/control_handlers.go`)
- **Relevant Alerts**: `RunawayAutomationDetected`, `EmergencyFreezeEngaged`, `BudgetCeilingBreached`

---

## 2. Symptoms

- Autonomous agent loop creating excessive mutating actions or runaway API costs.
- Widespread unintended tool executions detected across tenants or multi-agent swarms.
- SRE or Security incident response requires immediate cessation of all autonomous activity.

---

## 3. Immediate Triage (First 1 Minute)

1. **Engage Emergency Freeze immediately**:
   - Via CLI:

     ```bash
     agentmesh control freeze --tenant=<tenant_id> --reason="Runaway tool loop investigation"
     ```

   - Global / Mesh-wide Freeze (All Tenants):

     ```bash
     agentmesh control freeze --all-tenants --reason="Critical security emergency"
     ```

   - Direct HTTP API (using admin API key):

     ```bash
     curl -X POST https://agentmesh-cp/api/v1/control/freeze \
       -H "Authorization: Bearer mesh_..." \
       -H "Content-Type: application/json" \
       -d '{"reason": "Emergency freeze"}'
     ```

---

## 4. What Happens During Emergency Freeze

- All calls to `POST /api/v1/control/actions/:id/execute` are immediately blocked with HTTP 423 Locked (`ErrAutomationFrozen`).
- In-flight pending actions cannot be transitioned to execution.
- Read-only queries, metrics, policy simulations, and manual inspection remain fully functional.
- Zero data loss occurs: actions remain queued in `StatusApproved` or `StatusPending` until unfreeze.

---

## 5. Investigation & Root Cause Identification

1. Review the actions that triggered the freeze:

   ```bash
   agentmesh control actions list --status=pending --limit=50
   ```

2. Identify runaway agents or misconfigured automation policies.
3. Suspend or patch offending agents.

---

## 6. Unfreezing & Resuming Production

1. Once root cause is resolved and offending agents are isolated:

   ```bash
   agentmesh control unfreeze --tenant=<tenant_id> --reason="Investigation complete, rogue agent suspended"
   ```

2. Verify normal execution resumes for safe actions:

   ```bash
   agentmesh control actions list --status=completed
   ```

3. Monitor system telemetry for 30 minutes following unfreeze.
