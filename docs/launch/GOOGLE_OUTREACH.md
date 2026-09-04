# Google Ecosystem Technical Outreach Note

**Target Audience**: Google ADK Engineering Team, A2A Protocol Maintainers, Google Cloud Developer Relations, Vertex AI Partner Engineering.

---

## Subject

Technical Intro: AgentMesh — Open Control Plane for Google ADK, A2A & MCP Agents

## Message

Hi Team,

We wanted to share an open-source project we recently released that we think is highly complementary to the Google agent ecosystem: **AgentMesh** (<https://github.com/agentmesh/agentmesh>).

## What AgentMesh Adds

While the **Google Agent Development Kit (ADK) for Go** empowers developers to build sophisticated agent loops, production platform teams often ask how to operate and govern fleets of these agents when deployed across GKE or Cloud Run:

- **A2A Handshake & Delegation Governance**: AgentMesh provides a Go-native data-plane proxy that intercepts inter-agent A2A traffic, enforces call stack depth bounds (< 5 hops), and prevents cyclic loops.
- **Deterministic Tool Containment**: Sits in front of Google-managed MCP servers (BigQuery, Cloud Storage, Google Maps) to enforce deterministic `ALLOW`/`DENY`/`REQUIRE_APPROVAL` rules before tools execute.
- **Capability-Aware Routing**: Routes tasks across registered ADK agents based on real-time P95 latency against SLOs and token budgets.
- **Canary Progressive Delivery**: Allows teams to split traffic between Agent V1 and Agent V2 on GKE, rolling back automatically if error rates or tool regression thresholds are crossed.
- **Zero-Trust Cloud Identity**: Works seamlessly with Google Cloud Workload Identity, eliminating static JSON service account keys.

## Complementary Positioning

AgentMesh does not compete with or replace Google ADK, A2A, or MCP:

- **ADK** is how you *build* agents.
- **A2A** is how agents *communicate*.
- **MCP** is how tools *connect*.
- **AgentMesh** is the *control plane and proxy* that governs their execution, security, and progressive delivery in production.

We would welcome your technical feedback, architectural critiques, or suggestions for deeper ADK / Vertex integration.

Repository: <https://github.com/agentmesh/agentmesh>  
Reference Architecture: <https://github.com/agentmesh/agentmesh/blob/main/docs/GOOGLE_REFERENCE_ARCHITECTURE.md>
