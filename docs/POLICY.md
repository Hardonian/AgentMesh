# Deterministic Policy Engine

## Principles

1. **No LLMs in the Authorization Path**: Policy decisions must be mathematically deterministic, typed, and auditable.
2. **Default Deny**: Privileged operations, untrusted tool calls, and cross-tenant interactions default to `DENY`.
3. **Explicit Deny Precedence**: An explicit `DENY` rule immediately overrides any matching `ALLOW` rule.

## Policy Schema

```yaml
id: pol_enterprise_safety
version: "1.0.0"
name: "Enterprise Data Access Policy"
tenantId: "acme-corp"
rules:
  - name: "Allow BigQuery Read for Finance"
    effect: ALLOW
    agents: ["finance-agent"]
    tools: ["bigquery.read"]
    actions: ["read"]
    allowedDataClasses: ["INTERNAL", "CONFIDENTIAL"]

  - name: "Require Approval for BigQuery Delete"
    effect: REQUIRE_APPROVAL
    agents: ["*"]
    tools: ["bigquery.delete", "payment.*"]
    actions: ["delete", "execute"]

  - name: "Explicit Deny Restricted Data to Research"
    effect: DENY
    agents: ["research-agent"]
    denyDataClasses: ["RESTRICTED"]
```

## Policy Validation & Dry-Run CLI

```bash
# Validate policy syntax
agentmesh policy validate policy.yaml

# Test policy decisions
agentmesh policy test policy.yaml
```
