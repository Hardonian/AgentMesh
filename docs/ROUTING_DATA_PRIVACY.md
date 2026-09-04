# Routing Data Privacy & Tenant Isolation

AgentMesh strictly isolates private enterprise operational data from public or shared routing models.

## Tenant Isolation Boundary
- **Private Agents & Tools**: Never exposed outside tenant organization boundaries.
- **Private Routing Outcomes**: Stored with strict tenant foreign keys (`tenant_id` / `organization_id`).
- **Private Model Training**: Organization-specific models train exclusively on tenant-owned outcomes. Enterprise data is NEVER pooled into global routing models without explicit cryptographic consent.

## BigQuery Analytics Export
Exports to BigQuery (`/api/v1/analytics/export/bigquery`) automatically partition tables by tenant ID (e.g. `agentmesh_analytics_<tenant_id>`), guaranteeing zero cross-tenant contamination during offline research and analytical warehousing.
