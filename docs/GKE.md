# Deploying AgentMesh on Google Kubernetes Engine (GKE)

AgentMesh provides native Kubernetes integration on GKE using custom resources and sidecar patterns.

---

## GKE Architecture

```
GKE Cluster: agent-mesh-prod
  ├── Namespace: agentmesh-system
  │     ├── Deployment: agentmesh-controller
  │     └── Deployment: agentmesh-mcp-gateway
  └── Namespace: production-agents
        ├── Pod: finance-agent (ADK Go)
        │     ├── Container: finance-agent (port 8080)
        │     └── Container: agentmesh-proxy (sidecar, port 9090)
        └── Custom Resources:
              ├── AgentMeshAgent: finance-agent
              └── AgentMeshPolicy: finance-strict-policy
```

---

## GKE Workload Identity Setup

1. **Create GCP Service Account and Bind to K8s Service Account**:
   ```bash
   gcloud iam service-accounts create agentmesh-gke-sa --project=${PROJECT_ID}

   gcloud iam service-accounts add-iam-policy-binding agentmesh-gke-sa@${PROJECT_ID}.iam.gserviceaccount.com \
       --role roles/iam.workloadIdentityUser \
       --member "serviceAccount:${PROJECT_ID}.svc.id.goog[agentmesh-system/agentmesh-sa]"
   ```

2. **Annotate Kubernetes Service Account**:
   ```yaml
   apiVersion: v1
   kind: ServiceAccount
   metadata:
     name: agentmesh-sa
     namespace: agentmesh-system
     annotations:
       iam.gke.io/gcp-service-account: agentmesh-gke-sa@${PROJECT_ID}.iam.gserviceaccount.com
   ```

3. **Deploy Operator & CRDs**:
   The AgentMesh operator validates and applies `AgentMeshAgent`, `AgentMeshPolicy`, and `AgentMeshRoute` resources. It only manages pods explicitly opting in via `agentmesh.dev/inject: "true"` annotations, preventing unmanaged pod interception.
