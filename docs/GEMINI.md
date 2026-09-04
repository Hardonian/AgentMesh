# Google Gemini & Vertex AI Integration

## Model Provider Architecture
AgentMesh abstracts foundation models behind a clean `ModelProvider` interface, providing server-side execution, telemetry instrumentation, and token accounting:

- **Gemini Adapter**: Direct integration via `GEMINI_API_KEY`. Never exposes secrets to browser bundles.
- **Vertex AI Adapter**: Native integration using Google Cloud Project, Location, and Application Default Credentials (ADC) or Workload Identity.
- **Dynamic Token Accounting**: Automatically tracks input tokens, output tokens, and context-cached tokens to compute per-task USD expenditures against contract budgets.

## Integration States
External integrations expose transparent statuses:
- `LIVE_VERIFIED`: Live credentials configured and verified.
- `CONFIGURED`: Configuration provided, awaiting invocation.
- `NOT_CONFIGURED`: Missing environment variables.
- `DEGRADED`: Transient upstream provider failures.

*Rule: Never fake successful external service connections when credentials are absent.*
