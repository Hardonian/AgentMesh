# Agent Performance CI & Regression Gates

AgentMesh integrates evaluation suites directly into developer CI/CD workflows using `.agentmesh.yml` (`internal/evaluation/ci.go`).

---

## Configuration Schema (`.agentmesh.yml`)

```yaml
version: 1

agent:
  name: finance-agent
  version: "1.1.0"

evaluation:
  suite: finance-regression
  suite_file: ./evals/finance_suite.yaml

thresholds:
  min_quality: 0.95
  max_p95_latency_ms: 3000
  max_cost_per_task: 0.05
  max_error_rate: 0.01

policy:
  require_pass: true

canary:
  enabled: true
  initial_weight: 10
```

---

## CI Execution Gate
Whenever prompts, model targets, tools, or graph topologies are updated, CI runs:
```bash
agentmesh eval run ./evals/finance_suite.yaml
```

AgentMesh calculates:
- Overall Quality Score ($0.0$ to $1.0$)
- P95 Execution Latency
- Combined Task Cost (Model + Tools + Delegations)
- Policy Compliance Check
- Comparison against historical `EvaluationBaseline`

### Gating Verdict: `SafeToCanary`
If all thresholds pass and no policy violations are detected:
```
✓ Agent v1.1.0 Performance CI:
  Quality:      +2.1%  (PASS)
  P95 Latency:  -7.3%  (PASS)
  Cost/Task:    -14.0% (PASS)
  Policy:       PASS
  Safe to Canary: YES
```
If any threshold fails, CI exits with a non-zero code, preventing candidate deployment.
