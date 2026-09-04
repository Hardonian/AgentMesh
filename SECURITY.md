# Security Policy & Responsible Disclosure — AgentMesh

## 1. Supported Versions

| Version | Status | Security Support Level |
| --- | --- | --- |
| **v1.0.x (Phase 5 Release)** | **Supported** | Full active security patches & zero-day response |
| < v1.0.0 (Alpha / Beta) | Deprecated | Unsupported. Upgrade to v1.0.0+ |

---

## 2. Reporting a Vulnerability

The AgentMesh security team welcomes responsible disclosure from researchers, customers, and community members. If you discover a security vulnerability within AgentMesh (including policy bypasses, confused deputy attacks, delegation escalation, credential leakage, or SSRF risks), **DO NOT OPEN A PUBLIC GITHUB ISSUE**.

Please submit reports directly via encrypted email to:

- **Email**: `security@agentmesh.dev`
- **PGP Fingerprint**: `4A8F B12D 90E3 5C78 EF61 2890 A3D9 41CE 89F1 55B2`
- **PGP Key Available at**: `https://agentmesh.dev/security/pgp-key.asc`

### What to Include in Your Report

1. **Summary**: Clear description of the vulnerability and its potential impact.
2. **Reproduction Steps / PoC**: Minimal, reproducible proof-of-concept code or HTTP payloads.
3. **Affected Components**: Specify whether Data Plane Proxy, Control Plane, A2A, MCP bridge, or Policy Engine is affected.
4. **Environment**: Operating system, Go version, architecture, and deployment mode (GKE, Cloud Run, Bare Metal).

---

## 3. Response Service Level Agreement (SLA)

| Stage | Target Timeframe |
| --- | --- |
| **Initial Acknowledgment** | **< 24 hours** |
| **Triage & Severity Assessment** | **< 48 hours** |
| **Remediation & Patch Delivery (P0/P1)** | **< 7 calendar days** |
| **Coordinated Public Disclosure** | **30 to 90 days** (mutually agreed upon) |

---

## 4. Vulnerability Severity & Bug Bounty Tiers

AgentMesh operates a responsible disclosure bug bounty program for eligible findings:

| Severity | CVSS Range | Examples | Bounty Range |
| --- | --- | --- | --- |
| **Critical (P0)** | 9.0 – 10.0 | Remote Code Execution (RCE), Multi-Tenant Isolation Breach, Deterministic Policy Bypass leading to arbitrary tool execution, SSRF to cloud metadata. | **$5,000 – $15,000** |
| **High (P1)** | 7.0 – 8.9 | Delegation Privilege Escalation, HITL Parameter Tampering / Token Replay, Secret/Credential Leakage in Telemetry. | **$2,000 – $5,000** |
| **Medium (P2)** | 4.0 – 6.9 | Denial of Service via resource exhaustion, CORS/Header misconfigurations, Incomplete audit trails. | **$500 – $2,000** |
| **Low (P3)** | 0.1 – 3.9 | Minor information disclosures, non-exploitable error messages. | **Swag / Hall of Fame** |

---

## 5. Safe Harbor

Activities conducted in good faith according to this policy are considered authorized and we will not initiate legal action against researchers who comply with responsible disclosure guidelines.
