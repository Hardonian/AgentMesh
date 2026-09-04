# Google-Native Reference Architecture

AgentMesh provides a first-class production intelligence and governance layer deeply aligned with Google Cloud technologies: **Google Agent Development Kit (ADK)**, **Gemini Models**, **Vertex AI Model Garden**, **Google-Managed MCP services**, **Cloud Run**, **Google Kubernetes Engine (GKE)**, and **Cloud Observability (Cloud Trace / Cloud Logging)**.

---

## High-Level Topology

```
┌──────────────────────────────────────────────────────────────────┐
│                 Client Applications / Task Triggers              │
└─────────────────────────────────┬────────────────────────────────┘
                                  │
                                  ▼
┌──────────────────────────────────────────────────────────────────┐
│              AgentMesh Go Proxy / Data Plane (:9090)             │
│   • Millisecond Token Verification (Workload Identity / Ed25519) │
│   • Capability Router V2 (Evidence-Weighted Quality/Cost/SLO)   │
│   • Semantic Authorization & Deterministic Policy Engine         │
│   • Delegation Taint Context Propagation                         │
│   • Distributed Circuit Breakers & Retry Budgets                 │
└───────────┬─────────────────────┬────────────────────┬───────────┘
            │                     │                    │
            ▼                     ▼                    ▼
   ┌─────────────────┐   ┌─────────────────┐  ┌─────────────────┐
   │ Google ADK Go   │   │ Gemini / Vertex │  │ Google Managed  │
   │ Agents (GKE /   │   │ AI Endpoints    │  │ MCP Endpoints   │
   │ Cloud Run)      │   │ (Pro / Flash)   │  │ (BigQuery, GCS) │
   └────────┬────────┘   └─────────────────┘  └─────────────────┘
            │
            ▼ (A2A Protocol v0.3.0)
   ┌─────────────────┐
   │ Secondary A2A   │
   │ Delegated Agent │
   └─────────────────┘
```

---

## Key Pillars

### 1. Google ADK Graph Awareness
AgentMesh inspects Google ADK Go projects using static AST analysis without executing untrusted code. It extracts canonical `AgentGraph` topologies, identifies tools, model dependencies, and human-in-the-loop approval nodes, and analyzes graph risk across 9 dimensions including cycles, unbounded delegation, and indirect privilege escalation.

### 2. Gemini & Vertex AI Dynamic Model Routing
Requests targeting model completions are governed by AgentContract policies specifying allowed providers, models, regions, and cost budgets. AgentMesh routes to optimal model choices (e.g. Gemini 1.5 Pro vs Flash) with automatic health tracking and strictly policy-governed fallback.

### 3. Google-Managed MCP Tool Governance
AgentMesh models configured Google Cloud MCP services as governed tool providers. Turnkey policy templates enforce:
- `READ_ONLY` analytics patterns
- `APPROVAL_FOR_WRITE` for mutating operations
- `DENY_DESTRUCTIVE` blocking accidental resource destruction
- `RESTRICT_PROJECT` and `RESTRICT_REGION` boundaries

### 4. GKE & Cloud Run Production Deployment
- **Cloud Run**: Zero-maintenance serverless agent deployment using GCP Workload Identity federation.
- **GKE**: Native Kubernetes CRDs (`AgentMeshAgent`, `AgentMeshPolicy`, `AgentMeshRoute`) reconciled safely with zero unmanaged pod interception.

### 5. Google Cloud Observability Correlation
Every audit event, policy denial, and telemetry span captures standard W3C trace context headers (`traceparent`, `tracestate`) that correlate cleanly with Google Cloud Trace and Cloud Logging without vendor lock-in.
