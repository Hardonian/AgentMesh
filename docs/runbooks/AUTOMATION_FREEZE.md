# Runbook: Emergency Automation Freeze & Recovery

## Activation
Trigger global kill switch immediately:
```bash
agentmesh automation freeze "operator manual freeze"
```

## Status Verification
Check active freezes:
```bash
agentmesh automation status
```

## Resuming Automation
After all production signals are verified healthy:
```bash
agentmesh automation unfreeze
```
