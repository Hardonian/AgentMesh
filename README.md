# AgentMesh

**The open control plane for A2A and MCP agents.**

*Identity, policy, routing, reliability, and progressive delivery for production AI agent systems.*

---

```
                       AGENTS (ADK / LangGraph / Custom / A2A)
                                       │
                          ┌────────────┴────────────┐
                          ▼                         ▼
                      A2A Request               MCP Request
                          │                         │
                          └────────────┬────────────┘
                                       ▼
                       AGENTMESH DATA PLANE (agentmesh-proxy)
                ┌──────────────────────────────────────────────┐
                │  1. Agent Identity & Scope Verification      │
                │  2. Deterministic Policy Engine (ALLOW/DENY) │
                │  3. HITL Human Approval Interceptor          │
                │  4. Delegation Graph & Cycle Detection       │
                │  5. Capability-Based Routing Engine          │
                │  6. Reliability (Circuit Breakers & Retries) │
                │  7. Budget Enforcement (Tokens/Cost/Calls)   │
                │  8. Privacy Scrubber & OpenTelemetry Export  │
                └──────────────────────┬───────────────────────┘
                                       │
             ┌─────────────────────────┼─────────────────────────┐
             ▼                         ▼                         ▼
       Downstream Agent           MCP Tool Server           Model Provider
       (A2A Protocol)            (Local / Google Cloud)    (Gemini / Vertex / OSS)
             │                         │                         │
             └─────────────────────────┼─────────────────────────┘
                                       ▼
                       AGENTMESH CONTROL PLANE (agentmesh-controller)
                ┌──────────────────────────────────────────────┐
                │  • Agent & Tool Registry                     │
                │  • AgentContract & AgentBOM Management       │
                │  • Agent Passport (Declared vs Measured)     │
                │  • Policy Authoring & Cryptographic Signing  │
                │  • Progressive Delivery & Canary Controller  │
                │  • Evaluation Engine & Regression Tracking   │
                │  • Audit Trail & Multi-Tenant RBAC           │
                └──────────────────────┬───────────────────────┘
                                       │
                          ┌────────────┴────────────┐
                          ▼                         ▼
                PostgreSQL / Datastore      Web Control Plane (Next.js)
```

## What is AgentMesh?

When enterprises deploy autonomous agents, they confront a critical infrastructure challenge:
> **Which agent is allowed to do what, through which tool, under which policy, at what cost, with what reliability, and what should happen when it fails?**

AgentMesh functions conceptually like **Envoy + Istio + OPA + Argo Rollouts specialized for AI agents**. It is a Go-native, vendor-neutral control plane and data plane that sits between autonomous agents and their downstream tools, models, and peer agents.

### Google-First, Never Google-Locked
AgentMesh delivers first-class alignment with Google technologies:
- **Google ADK for Go**: Ingests workflow graph topologies, discovers tools, and synthesizes AgentContracts.
- **Gemini & Vertex AI**: First-class model provider adapters with dynamic token accounting and cost tracking.
- **Google Cloud Run & GKE**: Production multi-stage Dockerfiles, Helm charts, and Kubernetes Operator.
- **Google Managed MCP**: Access governance for BigQuery, Cloud Storage, and Google Maps tool servers.
- **Vendor Neutrality**: Fully interoperable with OpenAI-compatible endpoints, Anthropic Claude, open-weights models, standard MCP servers, and custom A2A agents on any cloud.

---

## The 5 Core Capabilities

| Capability | What AgentMesh Provides |
|---|---|
| **1. Identity** | Cryptographic identities for every agent, tool, and credential (`mesh_...`). Tenant-isolated RBAC with scoped keys. |
| **2. Policy** | Deterministic, typed declarative rules (`ALLOW`, `DENY`, `REQUIRE_APPROVAL`). Zero LLM hallucinations in the authorization path. Data classifications (`PUBLIC`, `INTERNAL`, `CONFIDENTIAL`, `RESTRICTED`). |
| **3. Routing** | Capability-based routing across eligible agents. Multi-stage filtering by health, policy, cost, and latency. Explainable routing decisions (`agentmesh route explain`). |
| **4. Reliability** | 3-state Circuit Breakers (`CLOSED`, `OPEN`, `HALF_OPEN`), safe retries enforcing idempotency (never retrying side-effecting writes), token-bucket rate limits, bounded deadlines, and hard budget enforcement. |
| **5. Progressive Delivery** | Canary traffic splits (1% to 100%), automated rollback upon error/latency regressions, shadow execution mode, and regression test suites. |

---

## Core Primitives

### 1. AgentContract
The canonical specification defining an agent's capabilities, tool allow/deny lists, delegation bounds, budgets, and SLOs:

```yaml
apiVersion: agentmesh.dev/v1
kind: AgentContract

metadata:
  name: procurement-agent
  organization: acme-corp

identity:
  protocols:
    - a2a
    - mcp

capabilities:
  - vendor_search
  - quote_analysis
  - purchase_request

tools:
  allow:
    - bigquery.read
    - drive.read
    - internal.erp.quote
  deny:
    - gmail.send
    - payment.execute

delegation:
  allow:
    - finance-agent
    - research-agent
  maxDepth: 3

budgets:
  max_cost_per_task: 0.25
  max_tokens_per_task: 100000

slo:
  p95_latency_ms: 12000
  success_rate: 0.995

approval:
  required_for:
    - purchase_request
```

### 2. Agent Passport
Combines declared contract specifications with empirical operational evidence:
- Clearly separates `DECLARED` claims from `MEASURED` and `INFERRED` performance.
- Tracks empirical success rates, measured P95 latencies, average task costs, and tool reliability scorecards.
- *Rule: Never present declared capability as proven performance.*

### 3. AgentBOM (Agent Bill of Materials)
Machine-readable inventory detailing an agent's runtime, models, MCP tools, delegation targets, data classifications, and dependencies.

### 4. A2A Firewall & MCPGuard
- **A2A Firewall**: Enforces policy on agent-to-agent interactions, terminating delegation cycles (`A -> B -> A`) and preventing privilege escalation through delegation.
- **MCPGuard**: Sits as a reverse proxy gateway in front of Model Context Protocol (MCP) servers, validating tool execution requests, enforcing data classifications, and intercepting sensitive operations for Human-in-the-Loop (HITL) approval.

---

## Quickstart

### Prerequisites
- Go 1.26+ installed
- Node.js & pnpm (optional for web control plane)
- Docker (optional for containerized deployment)

### 1. Clone and Build
```bash
git clone https://github.com/agentmesh/agentmesh.git
cd AgentMesh

# Build all binaries into bin/
make build
```

### 2. Start Local Control Plane (Zero External Dependencies)
```bash
# Launches controller on port 8080 with in-memory datastore
make dev
```

In a second terminal, launch the high-performance data plane proxy:
```bash
./bin/agentmesh-proxy
# Proxy is online on :9090
```

### 3. Register an Agent and Validate Contract
```bash
# Initialize a starter contract
./bin/agentmesh init

# Validate the contract
./bin/agentmesh contract validate agent.contract.yaml

# Register with control plane
./bin/agentmesh agent register agent.contract.yaml

# Inspect agent and its Agent Passport
./bin/agentmesh agent inspect my-first-agent
```

---

## Go SDK Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/agentmesh/agentmesh/pkg/contracts"
    "github.com/agentmesh/agentmesh/pkg/sdk"
)

func main() {
    client := sdk.NewClient("http://127.0.0.1:8080", "mesh_api_key")

    // Evaluate policy before tool invocation
    decision, err := client.EvaluatePolicy(context.Background(), &sdk.PolicyEvaluationRequest{
        SubjectAgentID: "procurement-agent",
        Tool:           "bigquery.read",
        Action:         "read",
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("Policy Decision: %s (Reason: %s)\n", decision.Effect, decision.Reason)
}
```

---

## 15 Critical Invariants Enforced by AgentMesh

1. **Denied Tool Block**: A tool denied by policy can never execute through the proxy.
2. **Pre-Approval Interception**: Approval-required tools are blocked until a cryptographically verified human approval token is provided.
3. **Tenant Isolation**: Agent A in Tenant A can never invoke tools or policies belonging to Tenant B.
4. **Anti-Privilege Escalation**: Agent A cannot gain access to a tool it lacks permission for simply by delegating to Agent B.
5. **Cycle Termination**: Delegation loops (`A -> B -> A`) are detected and aborted immediately.
6. **Hard Budget Enforcement**: Budget overflow (tokens, cost, tool calls) immediately halts subsequent execution.
7. **Policy-Eligible Routing**: The routing engine never selects an agent disqualified by policy.
8. **Signed Configuration Integrity**: Tampered or expired signed configuration bundles are rejected by proxies.
9. **Offline Data Plane Survivability**: If the control plane drops, the proxy continues serving traffic using cached last-known-good configuration.
10. **Telemetry Privacy**: Secrets, bearer tokens, and API keys are scrubbed before emission to traces.
11. **Idempotent Retries Only**: Non-idempotent operations (`NON_RETRYABLE`) are never retried automatically.
12. **Automated Canary Rollback**: Canary revisions breaching SLO error/latency thresholds automatically roll back to the baseline revision.
13. **Disabled Agent Exclusion**: Disabled agents are immediately excluded from routing and proxy dispatch.
14. **Credential Expiry**: Expired or revoked API credentials immediately fail authentication.
15. **Scoped RBAC**: API keys with limited scopes cannot perform unauthorized management actions.

---

## Repository Layout

```
├── cmd/
│   ├── agentmesh/             # Unified Go CLI binary
│   ├── agentmesh-proxy/       # Standalone high-performance data plane proxy
│   ├── agentmesh-controller/  # Standalone control plane REST API server
│   └── agentmesh-worker/      # Background worker for evaluations and canaries
├── pkg/
│   ├── contracts/             # AgentContract specification, validation, hashing
│   ├── agentbom/              # AgentBOM inventory specification & generator
│   ├── passport/              # Agent Passport (declared vs measured claims)
│   ├── protocol/              # A2A and MCP JSON-RPC protocol models
│   └── sdk/                   # Official Go SDK client
├── internal/
│   ├── identity/              # Agent/tool identity, API keys, scopes
│   ├── policy/                # Deterministic policy engine (ALLOW/DENY/APPROVAL)
│   ├── routing/               # Capability routing & explanation engine
│   ├── reliability/           # Circuit breakers, safe retries, rate limiters
│   ├── budgets/               # Token, cost, and tool-call limits
│   ├── delegation/            # Delegation graph, cycle detection, anti-escalation
│   ├── approval/              # HITL approval workflow with parameter binding
│   ├── evaluation/            # Evaluation test runner & regression detector
│   ├── canary/                # Progressive delivery & automated rollback
│   ├── telemetry/             # OpenTelemetry, waterfall trace, secret scrubber
│   ├── audit/                 # Append-only hash-chained audit log
│   ├── cost/                  # Pricing intelligence & token accounting
│   ├── crypto/                # Ed25519 signing & multi-key verification
│   ├── config/                # Configuration & proxy cache
│   ├── database/              # PostgreSQL schema, migrations, memory store
│   ├── a2a/                   # A2A server, card inspector, firewall
│   ├── mcp/                   # MCP reverse proxy gateway & MCPGuard
│   ├── adk/                   # Google ADK Go integration & graph topology
│   ├── providers/             # Gemini, Vertex AI, and local model adapters
│   └── server/                # HTTP REST router & OpenAPI endpoints
├── operator/                  # Kubernetes Operator (CRDs & reconcilers)
├── deploy/                    # Docker, Helm chart, Kubernetes, Cloud Run
├── examples/                  # Working runnable examples (ADK, A2A, MCP, Canary)
├── tests/                     # 15-point invariant test suite
├── web/control-plane/         # Next.js / TypeScript dashboard
└── docs/                      # Comprehensive technical documentation
```

---

## License

AgentMesh is licensed under the [Apache License 2.0](LICENSE).
Notice and trademark information is documented in [NOTICE](NOTICE) and [TRADEMARKS.md](TRADEMARKS.md).
