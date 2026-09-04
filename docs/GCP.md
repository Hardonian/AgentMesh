# Google Cloud Platform (GCP) Deployment & Security

## Cloud Run
Google Cloud Run provides an ideal serverless container deployment for the AgentMesh Control Plane:
- Fast autoscaling (0 to N instances).
- Native Google Cloud IAM authentication.
- Cloud Run service configuration provided in `deploy/cloudrun/service.yaml`.

## Google Kubernetes Engine (GKE)
For production Kubernetes environments, GKE supports:
- **Proxy Modes**: Run `agentmesh-proxy` as a high-performance sidecar next to agent pods or as a node-level DaemonSet.
- **Workload Identity**: Map Kubernetes ServiceAccounts to Google Cloud IAM ServiceAccounts to access Vertex AI and BigQuery without long-lived JSON keys.
- **Helm**: Deploy full stack with `deploy/helm/agentmesh`.
