# Runbook: Model Provider Outage (Gemini / Vertex)

## Symptoms
- HTTP 429 Rate Limit or 503 Service Unavailable errors on model inference.
- Model router triggers repeated fallbacks.

## Recovery Steps
1. Inspect active model router health:
   ```bash
   agentmesh router status
   ```
2. Switch capability model target to secondary regional endpoint or fallback model family.
3. Review fallback success rates in telemetry dashboard.
