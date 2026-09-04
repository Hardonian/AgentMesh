# Production Routing Intelligence V2

AgentMesh Capability Routing V2 (`internal/routing/v2.go`) moves beyond naive round-robin or static endpoint load balancing. It executes a deterministic, multi-stage filtering and ranking pipeline for every incoming task.

---

## The Routing Pipeline

```
Incoming Task Request (Capability, Allowed Data Classes, Latency & Cost SLOs)
                        ↓
Stage 1: Health & Availability Filtering (Circuit Breakers Closed)
                        ↓
Stage 2: Capability Matching (Declared or Empirical Match)
                        ↓
Stage 3: Tool & Data Policy Verification (Required tools allowed, no RESTRICTED leaks)
                        ↓
Stage 4: Multi-Strategy Objective Ranking (Quality, Cost, Latency, Reliability, Balanced)
                        ↓
Stage 5: Confidence Calculation & Deterministic Auditable Explanation
                        ↓
Selected Candidate Agent + Logged Route Outcome
```

---

## Ranking Strategies
1. **QUALITY**: Prioritizes highest historical task success rate and benchmark evaluation score.
2. **COST**: Minimizes token and tool invocation expenditure per task.
3. **LATENCY**: Prioritizes lowest P95 execution duration.
4. **RELIABILITY**: Minimizes error rates, retries, and circuit breaker trips.
5. **BALANCED**: Balanced composite score optimizing quality ($0.35$), reliability ($0.30$), latency ($0.20$), and cost ($0.15$).

---

## Deterministic Route Explanation V2
Every routing decision produces an auditable breakdown explaining why candidates were selected or disqualified:
```json
{
  "selectedAgentId": "procurement-agent",
  "selectedVersion": "1.0.0",
  "strategy": "BALANCED",
  "score": 0.942,
  "confidence": 0.96,
  "candidates": [
    {
      "agentId": "procurement-agent",
      "eligible": true,
      "evidenceTier": "PRODUCTION_OBSERVED_CAPABILITY",
      "qualityScore": 0.98,
      "p95LatencyMs": 4200,
      "costUsd": 0.0078,
      "compositeScore": 0.942
    },
    {
      "agentId": "legacy-procurement",
      "eligible": false,
      "disqualificationReason": "Data classification RESTRICTED forbidden by agent contract"
    }
  ]
}
```

---

## Offline Regret Analysis
AgentMesh records `RouteOutcome` events to analyze post-execution regret:
- Compares whether an alternative eligible candidate would have yielded lower latency or reduced cost without sacrificing quality.
- Regret reports inform routing weight adjustments without mutating live traffic in an uncalibrated fashion.
