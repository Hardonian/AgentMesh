# Policy Shadow Canaries & Safe Rollouts

Deploying modified security policies to production agent clusters poses severe risks: an overly restrictive rule can silently break legitimate agent workflows, while an overly permissive rule introduces privilege escalation.

AgentMesh implements **Policy Shadow Canaries** (`internal/policy/shadow.go`) to test policy revisions safely against real traffic.

---

## Architecture of Policy Shadow Mode

```
Incoming Request (Agent, Tool, Action)
                    ↓
        ┌───────────┴───────────┐
        ▼                       ▼
  [ Baseline Policy ]     [ Candidate Policy ]
  (LIVE ENFORCEMENT)       (SHADOW EVALUATION)
        │                       │
        ▼                       ▼
  Allow / Deny             Would-Allow / Would-Deny
  Decision Returned             │
                                ▼
                    Discrepancy Counter & Audit
```

### Invariant: Zero Disruption
In shadow mode, the candidate policy **NEVER** enforces decisions on live traffic. The baseline policy continues to govern the request, while the shadow evaluator records:
- Total evaluated traffic
- `WouldAllowCount`, `WouldDenyCount`, `WouldApproveCount`
- Discrepancy events where candidate and baseline diverge

---

## Policy Promotion & Rollback
- **Promotion**: If shadow evaluation yields zero unexpected denials over a designated window, operators promote the candidate to active baseline with one click.
- **Rollback**: A signed snapshot of the `LastKnownGood` policy is permanently retained. In an emergency, `agentmesh-controller` restores the last known good policy within milliseconds.
