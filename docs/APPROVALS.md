# Human-in-the-Loop (HITL) Approvals

## Mechanism

When a policy decision produces `REQUIRE_APPROVAL`, the data plane intercepts the tool call, generates an approval request, and blocks execution:

```text
Agent Attempts Tool Call
         │
         ▼
AgentMesh Policy Engine ──► Evaluates: REQUIRE_APPROVAL
         │
         ▼
Generates Approval Request (appr_...) + Returns MCP Error -32002
         │
         ▼
Human Reviewer Inspects Parameters & Approves in UI/API
         │
         ▼
Generates Cryptographic Approval Token (tok_...)
         │
         ▼
Agent Resumes Tool Call with Approval Token ──► Proxy Verifies & Passes Call
```

## Immutable Parameter Binding

Approval tokens bind to:

- Exact requesting Agent ID
- Exact targeted Tool Name
- **SHA-256 Hash of Action Parameters**
- Policy version & expiry timestamp

**Tampering Invariant**: If an attacker or rogue agent alters any argument between approval and execution, the parameter hash mismatches and execution fails with `ErrApprovalTampered`.
