# Agent Performance Continuous Delivery (CD)

AgentMesh evolves Performance CI into a complete deployment lifecycle:

```text
Agent Revision → Contract Validation → Graph Policy Validation → Benchmark Evaluation → Candidate → Shadow → Progressive Canary → Promotion → Verified Outcome
```

## `.agentmesh.yml` V2 Specification

```yaml
version: 2

performance_cd:
  enabled: true
  mode: guarded_automation
  strategy: canary

  stages:
    - 1
    - 5
    - 10
    - 25
    - 50
    - 100

  promotion:
    min_success_rate: 0.99
    max_p95_regression_percent: 5.0
    max_cost_regression_percent: 5.0

  rollback:
    max_error_rate_percent: 1.0
    max_policy_violation_count: 0
```
