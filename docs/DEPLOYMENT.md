# Deployment Guide

## Local Zero-Dependency Mode
```bash
make dev
```
Launches the control plane with in-memory storage on `:8080` and the proxy on `:9090`.

## Docker Compose
```bash
docker compose up -d
```
Runs PostgreSQL, control plane controller, and data plane proxy in containers with auto-migration.

## Production Kubernetes (Helm)
```bash
helm install agentmesh ./deploy/helm/agentmesh \
  --set database.url="postgres://user:pass@host:5432/agentmesh?sslmode=require"
```

## Google Cloud Run
```bash
gcloud run services replace deploy/cloudrun/service.yaml
```
