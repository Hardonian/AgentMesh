# RB-09: Secret Leak Remediation & Telemetry Scrubbing

## 1. Metadata

- **Severity**: P1 (Security Incident)
- **Target Component**: Telemetry Subsystem, Secret Scrubber (`internal/telemetry`)
- **Relevant Alerts**: `SecretLeakDetected`, `UnredactedTokenFoundInSpan`

---

## 2. Symptoms

- Telemetry trace, metric label, or log message contains an unredacted credential, token, or private key.
- A downstream SIEM, Datadog, or Cloud Logging alert flagged API keys in payload logs.

---

## 3. Immediate Triage (First 5 Minutes)

1. Identify the leaked secret value and determine its type:
   - OpenAI key (`sk-live-...`)
   - Anthropic key (`sk-ant-...`)
   - Google API key (`AIza...`)
   - AWS Secret Access Key (40 chars)
   - Database / Bearer password
2. Locate the specific trace or log stream where exposure occurred:

   ```bash
   agentmesh telemetry trace inspect <trace_id>
   ```

---

## 4. Remediation Steps

1. **Immediate Credential Invalidation**:
   - Rotate the leaked secret in the origin provider (OpenAI, AWS, GCP, DB) immediately.
   - Do NOT wait for log purging before revoking the exposed key.
2. **Purge Leaked Telemetry from Downstream Log Stores**:
   - Issue deletion request in Google Cloud Logging / Datadog / Elasticsearch for the affected index and time window:

     ```bash
     gcloud logging logs delete agentmesh-traces --quiet
     ```

3. **Inspect and Update Secret Scrubber Patterns in `internal/telemetry`**:
   - Check why the pattern was not scrubbed by regex filters (`reBearer`, `reOpenAI`, `reAWSSecret`, etc.).
   - Add new regex signature to `telemetry.go` to catch any non-standard credential format.

---

## 5. Verification & Post-Incident Actions

- Test updated scrubber with sample payload containing the pattern:

  ```bash
  go test -v ./internal/telemetry -run TestSecretScrubber
  ```

- Confirm all subsequent traces show `[REDACTED_*]` placeholders.
- Verify invalidation of the old compromised key.
