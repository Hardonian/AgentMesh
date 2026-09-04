# Routing Security & Anti-Poisoning Architecture

As routing decisions increasingly depend on operational feedback, AgentMesh implements defensive boundaries to prevent routing poisoning, confused deputy exploits, and health spoofing.

## Threat Model & Defenses

1. **Malicious Success Falsification**:
   - Threat: A rogue agent reports fake success to inflate its routing preference.
   - Defense: Provenance tiering. Proxy-measured outcomes and test-runner verifications are given higher trust weight than client self-reports.
2. **Learned Routing Poisoning**:
   - Threat: Adversarial task inputs crafted to degrade model weights.
   - Defense: Outcome validation filter. Tasks resulting in policy denials or invalid schemas are pruned before training dataset ingestion.
3. **Health Spoofing**:
   - Threat: An agent attempts to send heartbeat or health status for another agent.
   - Defense: Authenticated credentials and service account token binding.
4. **Policy Bypass Prevention**:
   - Invariant: No learned routing model can ever propose or execute a candidate that does not satisfy Step 1 deterministic policy eligibility.
