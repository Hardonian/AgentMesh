# Deploying AgentMesh on Google Cloud Run

Google Cloud Run provides a serverless execution environment ideal for autoscaling ADK Go agents and the AgentMesh proxy.

---

## Deployment Architecture

```text
Internet / VPC Network
        ↓
Cloud Run Service: agentmesh-proxy
  • CPU: 2 vCPU, Memory: 1GiB
  • Concurrency: 250
  • Auth: Workload Identity / IAM
  • Container: agentmesh-proxy:v2.0.0
        ↓ (Internal HTTP / gRPC)
Cloud Run Service: adk-procurement-agent
  • Autoscaling: 0 to 50 instances
```

---

## Recommended Deployment Steps

1. **Build Container Image**:

   ```bash
   gcloud builds submit --tag gcr.io/${PROJECT_ID}/agentmesh-proxy:v2.0.0 .
   ```

2. **Deploy Proxy with Workload Identity**:

   ```bash
   gcloud run deploy agentmesh-proxy \
       --image gcr.io/${PROJECT_ID}/agentmesh-proxy:v2.0.0 \
       --region europe-west1 \
       --service-account agentmesh-proxy-sa@${PROJECT_ID}.iam.gserviceaccount.com \
       --set-env-vars CONTROL_PLANE_URL=https://controller.internal.mesh,PROXY_PORT=8080 \
       --cpu 2 --memory 1Gi \
       --min-instances 1 --max-instances 20
   ```

3. **Security Invariant**: Never bake static GCP service account JSON keys into container images. Always bind the Cloud Run Service Account using Google IAM roles (`roles/aiplatform.user`, `roles/bigquery.dataViewer`).
