# Google Enterprise Architecture for AgentMesh

AgentMesh coordinates and governs Google Cloud AI Agent infrastructure across Google ADK, Gemini, Vertex AI, GKE, and Cloud Run.

## End-to-End Enterprise Flow

```text
[Google ADK Go Agent]
          ↓
[AgentMesh Private Proxy (GKE / Cloud Run)]
          ↓
[A2A Protocol / Google Managed MCP Gateway]
          ↓
[Vertex AI / Gemini 1.5 Pro & Flash]
          ↓
[Google Cloud Logging, Cloud Trace & BigQuery Export]
```

## Google Security Alignment

- **Workload Identity Federation**: GKE service accounts exchange Kubernetes tokens for GCP access tokens via Google STS (`sts.googleapis.com`).
- **Google Cloud KMS**: Control plane configuration bundles and routing policies are cryptographically signed using Cloud KMS asymmetric keys.
- **Model Armor Integration**: Prompt injections, jailbreaks, and PII exfiltration are intercepted at the proxy layer using Vertex AI Model Armor filters.
- **Secret Manager References**: Configurations store Google Cloud Secret Manager resource IDs (`projects/*/secrets/*/versions/*`) rather than raw credentials.
