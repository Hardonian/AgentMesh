# MCP Governance & Tool Gateway

AgentMesh functions as an enterprise security gateway for the **Model Context Protocol (MCP)**. It governs agent tool access with deterministic policies, schema drift tracking, and Google-managed tool integrations.

---

## Tool Risk Classification
Tools are categorized into canonical risk classes (`ToolRiskClass`):
- `READ`: Safe, read-only data extraction (e.g. `bigquery.read`, `web.search`).
- `WRITE`: Data modification or external side effects (e.g. `gmail.send`, `database.update`).
- `DESTRUCTIVE`: Resource termination or data deletion (e.g. `gke.cluster.delete`, `s3.bucket.drop`).
- `FINANCIAL`: Money movement or purchase commitments (e.g. `stripe.charge`, `sap.po.create`).
- `INFRASTRUCTURE`: Cloud resource provisioning or configuration changes.

Classification sources:
- `DECLARED`: Stated in server metadata.
- `PROVIDER_METADATA`: Extracted from trusted provider descriptors.
- `ADMIN_ASSIGNED`: Explicitly designated by enterprise security operators.
- `INFERRED`: Derived via AST or semantic keyword analysis. Weak inference is never trusted alone for high-risk operations.

---

## Tool Fingerprinting & Schema Drift
Every tool version receives a deterministic SHA-256 fingerprint computed across:
- Server identity & Tool Name
- Version & Provider
- Risk Class
- Normalized Input & Output JSON Schemas

### Conservative Drift Detection
When a tool schema updates, AgentMesh compares old vs new fingerprints:
- `UNCHANGED`: Fingerprints match exactly.
- `COMPATIBLE_CHANGE`: Optional properties added; no fields removed.
- `POTENTIALLY_BREAKING`: Existing property removed or modified.
- `BREAKING`: New required fields introduced without backward compatibility.

Breaking drift automatically marks affected agent evaluation baselines and policies as **STALE**, requiring re-verification before progressive canary rollout.
