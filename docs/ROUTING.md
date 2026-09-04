# Capability-Based Routing Engine

## Routing Pipeline
When an agent or client requests task routing by capability, AgentMesh executes a deterministic multi-stage pipeline:

```
Task Request (Capability = "financial_analysis")
                     │
                     ▼
             1. Candidate Discovery (Matches capability)
                     │
                     ▼
             2. Availability Filter (Excludes DISABLED & UNHEALTHY)
                     │
                     ▼
             3. Policy Eligibility Filter (Caller allowed to invoke target)
                     │
                     ▼
             4. Budget & Cost Ceiling Filter (Avg cost <= max task budget)
                     │
                     ▼
             5. Latency SLO Filter (P95 latency <= caller max latency)
                     │
                     ▼
             6. Strategy Ranking & Deterministic Tie-Break
                     │
                     ▼
             Selected Agent + Route Evidence Explanation
```

## Supported Strategies
- `LOWEST_COST`: Minimizes USD expenditure.
- `LOWEST_LATENCY`: Minimizes P95 response time.
- `HIGHEST_RELIABILITY`: Maximizes empirical success rate.
- `HIGHEST_QUALITY`: Maximizes evaluation scorecard.
- `BALANCED`: Weighted composite (40% reliability, 30% latency, 30% cost).

## Route Explanation CLI
```bash
agentmesh route explain financial_analysis
```
Returns a structured breakdown explaining why the winning candidate was selected and why other candidates were excluded.
