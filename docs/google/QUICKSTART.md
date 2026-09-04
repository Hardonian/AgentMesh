# Google Cloud & Google ADK Production Quickstart

This guide details how to deploy and operate AgentMesh alongside the Google Agent Ecosystem:

- **Google Agent Development Kit (ADK) for Go**
- **Google Gemini & Vertex AI Models**
- **Google Cloud Run & Google Kubernetes Engine (GKE)**
- **Google Cloud Workload Identity**
- **Google Cloud Observability / OpenTelemetry**

---

## 1. Authentication via Workload Identity

AgentMesh supports Google Cloud Workload Identity, eliminating static JSON service account keys:

```bash
# Authorize local development via Application Default Credentials (ADC)
gcloud auth application-default login

# For GKE: Bind Kubernetes service account to GCP IAM role
gcloud iam service-accounts add-iam-policy-binding \
  agentmesh-sa@PROJECT_ID.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:PROJECT_ID.svc.id.goog[agentmesh/agentmesh-proxy]"
```

---

## 2. Ingesting Google ADK Agent Topologies

AgentMesh inspects compiled Google ADK Go agents, extracts their declared tool sets, and validates their graph topologies:

```bash
# Inspect an ADK agent directory
agentmesh adk graph inspect ./my-adk-agent

# Validate delegation depth and cycle freedom
agentmesh adk graph validate ./my-adk-agent

# Export canonical AgentContract
agentmesh adk contract export ./my-adk-agent > agentmesh.yaml
```

---

## 3. Configuring Gemini & Vertex AI Model Routing

Configure AgentMesh to route model queries to Gemini with automatic token cost accounting:

```yaml
# config/providers.yaml
providers:
  - id: gemini-1.5-pro
    type: vertex_gemini
    project: ${GCP_PROJECT_ID}
    location: us-central1
    model: gemini-1.5-pro-002
    costPerMillionInputTokensMicroUSD: 1250000   # $1.25 per 1M tokens
    costPerMillionOutputTokensMicroUSD: 5000000  # $5.00 per 1M tokens

  - id: gemini-1.5-flash
    type: vertex_gemini
    project: ${GCP_PROJECT_ID}
    location: us-central1
    model: gemini-1.5-flash-002
    costPerMillionInputTokensMicroUSD: 75000     # $0.075 per 1M tokens
    costPerMillionOutputTokensMicroUSD: 300000   # $0.30 per 1M tokens
```

---

## 4. Deploying to Google Kubernetes Engine (GKE)

Deploy AgentMesh using our official Helm chart:

```bash
# Add AgentMesh Helm repository
helm repo add agentmesh https://charts.agentmesh.dev
helm repo update

# Install AgentMesh onto GKE cluster
helm install agentmesh agentmesh/agentmesh \
  --namespace agentmesh --create-namespace \
  --set gcp.projectId="my-gcp-project" \
  --set gcp.workloadIdentity="true" \
  --set proxy.replicaCount=3 \
  --set controller.replicaCount=2
```

Verify pod readiness:

```bash
kubectl get pods -n agentmesh
```

---

## 5. Google-Managed MCP Governance

Govern access to Google Cloud MCP tools (BigQuery, Cloud Storage, Google Maps) through AgentMesh policy:

```yaml
rules:
  # Allow data extraction from specific BigQuery dataset
  - id: BQ-DATASET-READ
    effect: allow
    agent: analytics-agent
    tool: bigquery.read
    constraints:
      dataset: "sales_analytics_prod"

  # Deny table drops or broad mutations
  - id: BQ-TABLE-DROP
    effect: deny
    agent: "*"
    tool: bigquery.drop_table
```

All queries pass through the proxy with automated secret scrubbing and audit logging.
