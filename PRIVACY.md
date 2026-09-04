# Privacy & Data Governance Architecture — AgentMesh

## 1. Zero-Prompt Storage Principle

AgentMesh operates as a stateless intelligence and policy enforcement proxy:

- **No Raw Prompt Persistence**: AgentMesh **does not store, log, or persist raw LLM prompts or unstructured model outputs**.
- **Payload Inspection in Transient Memory Only**: Payloads are processed in transient memory for the duration of the request context solely for policy evaluation, tool schema validation, and parameter hashing.
- **Immediate Discard**: Once the request lifecycle completes, in-memory buffers are deallocated and garbage collected.

---

## 2. Telemetry Minimization & PII Scrubbing

1. **Automated Credential Redaction**:
   - All structured telemetry spans, traces, and metrics pass through `internal/telemetry` regex scrubbers.
   - Bearer tokens, OpenAI keys, Anthropic keys, AWS secrets, GCP credentials, and passwords are automatically redacted into `[REDACTED_*]` tokens before emission.
2. **Deterministic Parameter Hashes**:
   - High-privilege and HITL operations record `sha256(canonical_json(parameters))` rather than storing sensitive customer PII or transaction values in cleartext.
3. **No Secondary Data Utilization**:
   - Customer telemetry and operational metrics are never used to train proprietary machine learning models without explicit written opt-in consent.

---

## 3. Data Retention & Lifecycle Policies

| Data Category | Storage Location | Default Retention Period | Deletion Mechanism |
| --- | --- | --- | --- |
| **Audit Logs** | PostgreSQL (Encrypted at Rest) | 90 days (configurable up to 365 days) | Automated background vacuum & partition dropping |
| **Telemetry Spans** | OpenTelemetry Collector / Jaeger | 14 days | Rolling TTL in storage backend |
| **Metrics & Aggregations** | Prometheus / Google Cloud Monitoring | 30 days | Standard TSDB retention policy |
| **HITL Approval Records** | PostgreSQL | 90 days | Soft-delete / Archival to cold encrypted storage |
| **Agent Contracts & Passports** | PostgreSQL | Active lifecycle of the agent | Hard-delete on agent deregistration |

---

## 4. Multi-Tenant Data Isolation (GDPR & CCPA Compliance)

- **PostgreSQL Row-Level Security (RLS)**: Every tenant's data is isolated using PostgreSQL RLS policies (`CREATE POLICY tenant_isolation_policy`). Cross-tenant queries are blocked at the engine level.
- **Right to Erasure (GDPR Article 17)**: Tenants can execute a complete tenant purge via `DELETE /api/v1/tenants/:id`, which cascades across all agents, passports, routes, policies, and credentials.
- **Data Export (GDPR Article 20)**: Tenants can export their complete configuration, contracts, and audit trails in standard JSON format at any time.

---

## 5. Public Passport Sanitization

When agents are published or inspected via public/unauthenticated endpoints:

- Internal network topology, private VPC IPs, cluster node IDs, and microservice names are completely stripped (`SanitizeForPublic`).
- Only verified capability badges, SLO reliability scores, and public contact information are rendered.
- If an agent passport is marked `is_public: false`, all unauthenticated retrieval requests return `nil` / HTTP 404.
