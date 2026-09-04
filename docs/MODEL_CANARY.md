# Model Canaries & Multi-Model Progression

## Google Gemini & Vertex Model Upgrades

Transitioning underlying LLMs (e.g., Gemini-1.5-Flash to Gemini-1.5-Pro or new model checkpoints) carries behavioral risks including schema drift, prompt compliance differences, and latency variations.

AgentMesh treats model target changes as first-class canary targets:
1. **Offline Evaluation**: Candidate model evaluated against `EvaluationSuite` and `GoldenTaskSet`.
2. **Shadow Invocations**: Live requests shadowed to candidate with destructive tool calls stripped.
3. **Progressive Traffic Split**: Staged rollouts at 1%, 5%, 10%, 25%, 50%, 100%.
4. **Behavioral Guardrails**: Automated rollback if tool invocation patterns deviate or error rates spike.
