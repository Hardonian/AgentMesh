# 10-Minute Deep Technical Engineering Demo — AgentMesh

## Audience

Platform Engineers, Security Architects, Site Reliability Engineers (SREs), Go Infrastructure Developers.

---

## 1. System Health & Environment Verification (00:00 – 01:30)

```bash
agentmesh doctor
```

**Talking Points**:

- Shows Go runtime version, Docker, Kubernetes client, and Control Plane status.
- Clarifies that while Google Cloud (ADC, Gemini) is deeply supported, local development works 100% offline in mock/simulator mode.

---

## 2. Declarative AgentContract & Verification (01:30 – 03:00)

```bash
agentmesh contract validate examples/google-adk/agentmesh.yaml
```

**Talking Points**:

- Explain declarative capabilities, tool allow/deny lists, and delegation depth limits.
- Show how contracts are versioned in Git and verified in CI before deployment.

---

## 3. Deterministic Policy Simulation (03:00 – 05:00)

```bash
agentmesh policy simulate --agent=finance-agent --tool=bigquery.read
agentmesh policy simulate --agent=finance-agent --tool=payments.execute
```

**Talking Points**:

- Show deterministic `ALLOW` vs `REQUIRE_APPROVAL`.
- Explain how human-in-the-loop tokens bind to `sha256(canonical_json(params))`.
- Emphasize zero LLM hallucination risk in the authorization path.

---

## 4. Multi-Agent Delegation & Cycle Detection (05:00 – 07:00)

```bash
agentmesh a2a graph detect-cycles --tenant=tenant-1
```

**Talking Points**:

- Explain the ordered call stack mechanism in `internal/a2a`.
- Demonstrate how cyclic loops (`Agent A -> Agent B -> Agent A`) trigger immediate `ErrCycleDetected` rather than spinning out of control.
- Highlight Confused Deputy protection via delegation taint propagation.

---

## 5. Failure Injection & Outage Resilience (07:00 – 09:00)

```bash
# Simulate Control Plane failure by stopping server
agentmesh demo run
```

**Talking Points**:

- Show that the edge proxy continues routing and enforcing policy using its in-memory Last Known Good (LKG) signed bundle.
- Emphasize that the data plane is fully decoupled from the control plane hot path.

---

## 6. Secret Scrubbing & OpenTelemetry Trace (09:00 – 10:00)

```bash
agentmesh demo run --json
```

**Talking Points**:

- Point out the generated OpenTelemetry trace.
- Demonstrate that regex scrubbers in `internal/telemetry` redacted all secret keys into `[REDACTED_*]` tokens before storage.
