# 90-Second Product Walkthrough Video Script — AgentMesh v1.0

## Visual & Voiceover Timeline

| Time | Visual on Screen | Voiceover Narration |
| --- | --- | --- |
| **00:00 – 00:10** | Title card: AgentMesh logo with tagline *"The open control plane for A2A and MCP agents"* | "As organizations deploy autonomous AI agents, they quickly run into an infrastructure problem: Who governs what an agent is allowed to do, which tools it can call, and what happens when it fails?" |
| **00:10 – 00:25** | High-level SVG architecture diagram showing Agents -> AgentMesh Proxy -> MCP Tools & Gemini | "AgentMesh provides the zero-trust control plane between agents and the systems they can reach. Written in Go, it delivers sub-15ms proxy overhead with deterministic security gates." |
| **00:25 – 00:45** | Terminal running `agentmesh demo run`; zooming in on deterministic policy decision | "Watch what happens when an agent requests a tool. A read-only query is allowed instantly. But when it tries to execute a payment or database drop, AgentMesh halts execution and demands a single-use human approval token cryptographically bound to the exact parameters." |
| **00:45 – 01:05** | Browser UI showing Agent Operational Graph and OpenTelemetry trace breakdown | "Multi-agent delegation loops are detected and killed before they exhaust token budgets. Every action produces an OpenTelemetry trace with automated secret scrubbing for API keys and passwords." |
| **01:05 – 01:20** | Web dashboard showing canary deployment (Agent V1 vs Agent V2) and automatic rollback | "Deploying a new agent revision? Use canary traffic splits. If error rates or latency regresses, AgentMesh automatically rolls back traffic to the stable baseline in under 60 seconds." |
| **01:20 – 01:30** | GitHub repo hero with `git clone` command and documentation links | "AgentMesh is open-source under Apache-2.0, deeply integrated with Google ADK, Gemini, and GKE, and 100% vendor neutral. Try the 60-second demo on GitHub today." |
