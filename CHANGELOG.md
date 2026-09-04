# Changelog — AgentMesh

All notable changes to this project are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.0] - 2026-09-04

### Added

- **Core Proxy**: High-performance Go data-plane proxy with sub-15ms routing latency.
- **A2A Protocol**: Native Agent-to-Agent communication proxy with cryptographic peer handshakes.
- **MCP Tool Governance**: Gateway for Model Context Protocol (MCP) tool servers with schema drift detection.
- **Deterministic Policy Engine**: Pure Go policy gates (`ALLOW`, `DENY`, `REQUIRE_APPROVAL`) with zero LLM authorization overhead.
- **Human-In-The-Loop (HITL)**: Single-use approval tokens cryptographically bound to parameter SHA-256 digests.
- **Capability-Aware Routing**: Multi-stage candidate scoring by health, policy, P95 latency against SLA, and token cost.
- **Google Integrations**: Google ADK for Go graph inspection, Gemini / Vertex AI model adapters, GKE Helm charts, and Google Managed MCP support.
- **Progressive Delivery**: Canary weighted traffic splits (1% to 100%), automated rollback on SLA regression, and shadow testing mode.
- **Observability**: OpenTelemetry distributed tracing across multi-agent delegation chains with automated regex secret scrubbing.
- **Multi-Tenant Isolation**: PostgreSQL 16+ Row-Level Security (RLS) and fail-closed tenant scoping (`ErrEmptyTenant`).
- **CLI Commands**: Added `agentmesh version` and `agentmesh demo run` for 60-second zero-dependency evaluation.
- **Web Control Plane**: Production-grade Next.js 15 dashboard for agent graphs, approval queues, and fleet health.

### Security & Hardening

- 15 passing P0 adversarial red team attack scenarios (`tests/phase5_redteam_test.go`).
- 35 passing Definition-of-Done certification tests (`tests/phase5_certification_test.go`).
- Zero data races detected under Go race detector (`go test -race ./...`).
- Zero linter/compiler errors under `go vet ./...`.
- 8 mutational fuzz test suites covering all parser and decoder interfaces.
- Strict 10MB request and contract body size limits to prevent DoS/OOM attacks.
- SSRF prevention blocking private RFC1918 ranges, loopback, and cloud metadata (`169.254.169.254`).
