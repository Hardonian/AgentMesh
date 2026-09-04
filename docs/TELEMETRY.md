# OpenTelemetry & Observability

## OpenTelemetry Native

AgentMesh natively integrates with OpenTelemetry:

- **Traces**: Every task invocation, delegation hop, MCP tool execution, and model generation emits an OpenTelemetry span.
- **Metrics**: Exposes Prometheus-compatible metrics on `/metrics` (requests, policy latency, circuit breaker trips, token counts).
- **Structured Logging**: Go 1.26 `log/slog` with JSON handler.

## Privacy Scrubber

By default, AgentMesh scrubs:

- Bearer tokens (`Bearer [REDACTED_SECRET]`)
- Google API keys (`AIza...`)
- OpenAI API keys (`sk-...`)
- AgentMesh credentials (`mesh_...`)
- Authorization headers and passwords

Raw prompt contents and private documents are never recorded in telemetry traces unless explicitly opted into for debug sessions.
