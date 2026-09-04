# Open-Core Architecture & Licensing Boundary

AgentMesh is built as an open-core enterprise infrastructure platform.

## Apache 2.0 Open Source Components

The open-source core includes:

- **`agentmesh-proxy`**: High-performance Go data plane proxy.
- **`AgentContract`**: Canonical specification, JSON Schema, parser, and deterministic hashing.
- **`AgentBOM` & `AgentPassport`**: Core specifications and generators.
- **`agentmesh` CLI**: Full developer command suite (`init`, `doctor`, `validate`, `explain`, `test`, `ci`).
- **Go SDK**: Programmatic client library (`pkg/sdk`).
- **A2A & MCP Protocol Implementations**: Agent cards, task dispatch, JSON-RPC 2.0 gateway, and MCPGuard.
- **Deterministic Policy Engine**: Reference policy parser and evaluator.
- **OpenTelemetry Instrumentation**: Tracing, metrics, and secret scrubbers.
- **Kubernetes Operator & Helm Chart**: Deployment automation.

## Commercial Extensions

The commercial control plane layer provides:

- Hosted, multi-tenant control plane fleet management.
- Enterprise SSO (SAML 2.0, Okta, Google Workspace SAML) and SCIM provisioning.
- Advanced predictive routing and operational graph intelligence.
- Automated compliance reporting (SOC 2, ISO 27001, HIPAA) from immutable audit logs.
- Dedicated enterprise SLA and 24/7 support.
