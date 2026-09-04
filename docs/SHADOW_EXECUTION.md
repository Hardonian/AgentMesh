# Shadow Execution V2 & Side-Effect Containment

Shadow Execution allows testing candidate models and agents against live production traffic without impacting caller outcomes.

## Side-Effect Containment

To guarantee safety, AgentMesh intercepts tool calls during shadow invocations:
- **Permitted**: Read-only tools (`bigquery.read`, `document.search`, `embeddings.compute`).
- **Suppressed**: Side-effecting tools (`email.send`, `payment.charge`, `database.insert`, `cloud.delete`).

Shadow executions produce a `ShadowInvocationReport` comparing baseline vs candidate outputs, latency, cost, and tool call intent.
