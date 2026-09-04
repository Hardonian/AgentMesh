# Automation Policy V2 & Governance Modes

## Execution Modes

Organizations configure one of four operational modes:
- **`ADVISORY` (Default)**: AgentMesh generates recommendations and change plans. No automated mutations are applied.
- **`APPROVAL_REQUIRED`**: Automated workflows prepare change requests; a human operator must review and approve the exact action hash.
- **`GUARDED_AUTOMATION`**: Low-risk changes (minor route weight shifts, healthy failovers) execute automatically under strict policy guardrails.
- **`FULL_POLICY_AUTOMATION`**: Complete automated progression for mature organizations that explicitly opt in.

## Policy Schema

```yaml
automation:
  mode: guarded_automation
  allow:
    - CHANGE_ROUTE_WEIGHT
    - CHANGE_CANARY_PERCENT
  approval_required:
    - CHANGE_MODEL_TARGET
    - CHANGE_AGENT_VERSION
  deny:
    - CHANGE_TOOL_PROVIDER

requirements:
  min_reliability: 0.99
  min_eval_pass_rate: 0.97
  min_sample_count: 50

blast_radius:
  max_canary_percent: 25
  max_spend_usd: 100.0

economics:
  min_cost_improvement_percent: 10.0
```
