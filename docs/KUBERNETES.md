# Kubernetes Deployment & Operator

## Custom Resource Definitions (CRDs)
AgentMesh exposes Kubernetes-native resources under `agentmesh.dev/v1alpha1`:
- `AgentMeshAgent`: Manages agent identity, contract, and deployment configuration.
- `AgentMeshPolicy`: Manages declarative security and tool authorization rules in Kubernetes.

## Operator Reconciliation
The AgentMesh Operator reconciles desired Kubernetes state with the control plane, automatically syncing contracts and policies into the active data-plane routing tables.

## Helm Deployment
```bash
helm install agentmesh ./deploy/helm/agentmesh \
  --namespace agentmesh-system \
  --create-namespace \
  --values ./deploy/helm/agentmesh/values.yaml
```
