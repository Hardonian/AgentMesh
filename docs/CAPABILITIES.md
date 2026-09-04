# Capabilities & Evidence-Weighted Routing

## Capability Registry

AgentMesh introduces a normalized **Capability Registry** (`internal/routing/v2.go`) defining stable operational capabilities within an enterprise (e.g. `quote_analysis`, `invoice_reconciliation`, `deep_research`, `purchase_request`).

Capabilities bind to:

- **Required Tools**: Outbound tool dependencies necessary to fulfill the task.
- **Allowed Data Classes**: Constraints on input/output sensitivity (`PUBLIC`, `INTERNAL`, `FINANCIAL`, `RESTRICTED`).

---

## The Three Capability Evidence Tiers

AgentMesh rejects unproven claims. When matching agents to requested capabilities, the router assigns an **Evidence Tier**:

| Tier | Source | Description | Weight |
| --- | --- | --- | --- |
| `DECLARED_CAPABILITY` | AgentContract metadata | Declared in agent contract without empirical verification. | 0.30 |
| `EVALUATED_CAPABILITY` | CI Benchmark Suite | Verified against deterministic test cases and assertions. | 0.70 |
| `PRODUCTION_OBSERVED_CAPABILITY` | Live Proxy Telemetry | Observed across live production tasks with high sample count. | 1.00 |

---

## Route Confidence Algorithm

Every routing decision calculates a mathematically bounded confidence score ($0.0 \le C \le 1.0$):

$$C = \text{clamp}\left(w_{\text{tier}} \times \left(0.5 + 0.3 \cdot \min\left(1.0, \frac{\text{samples}}{50}\right) + 0.2 \cdot \text{reliability}\right), 0.1, 1.0\right)$$

- High production sample counts and verified reliability elevate confidence to $> 0.90$.
- Unverified declared claims cap confidence at $\le 0.30$.
