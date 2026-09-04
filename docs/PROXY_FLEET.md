# Enterprise Private Proxy Fleet

AgentMesh data-plane proxies can be deployed across private enterprise environments (GKE, Cloud Run, on-premise Kubernetes, or VMs) while remaining governed by a central control plane.

## Architecture

- **Outbound-Only Connectivity**: Proxies initiate outbound connections to the control plane, eliminating inbound firewall rules.
- **Signed Configuration Distribution**: Control plane distributes cryptographically signed configuration bundles.
- **Cluster & Region Tracking**: Tracks proxy instances across GKE clusters and regions (e.g. `gke-prod-us-central1`, `cloudrun-europe-west1`).

## Progressive Proxy Rollouts (Canaries)

Operators can target canary proxy versions to a subset of fleet instances. The control plane monitors proxy health, heartbeat frequency, and error rate during canary rollouts.

## CLI Fleet Inspection

```bash
agentmesh proxy fleet
```

Outputs total proxy counts, healthy vs canary counts, and version distribution across clusters.
