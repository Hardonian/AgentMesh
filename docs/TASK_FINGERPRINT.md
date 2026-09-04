# Task Fingerprint Specification

`TaskFingerprint` represents task characteristics for routing, capability matching, and analytics without retaining sensitive customer prompt payloads.

## Privacy-First Schema

- `FingerprintID`: Deterministic 16-character SHA-256 digest of normalized task dimensions.
- `Capability`: Target task category (e.g. `financial_research`, `sql_generation`).
- `InputSizeClass`: Bracketed size (`SMALL` < 1KB, `MEDIUM` < 32KB, `LARGE` < 512KB, `XLARGE`).
- `OutputSizeClass`: Expected response bracket.
- `Streaming`: Boolean flag indicating stream requirements.
- `RequiredTools`: Alphabetically sorted slice of tool identifiers (e.g. `["bigquery.read"]`).
- `DataClassification`: Security class (`PUBLIC`, `INTERNAL`, `CONFIDENTIAL`, `RESTRICTED`, `SOVEREIGN`).
- `TargetRegion`: Sovereign execution requirement (e.g. `us-central1`, `europe-west1`).
- `MaxLatencyMs` / `MaxCostUSD`: Task budget constraints.
- `DelegationAllowed`: Boolean permission flag.
- `ModelConstraints`: Approved foundation model restrictions.
- `ComplexityClass`: `SIMPLE`, `STANDARD`, `COMPLEX`, `LONG_CHAIN`.
- `StructuredOutput`: Boolean flag for JSON schema outputs.

## Zero Payload Retention

Prompt contents and user query text are NEVER incorporated into the fingerprint digest. All hashing relies on structural, security, and resource dimensions.
