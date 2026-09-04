# RB-04: MCP Tool Drift & Schema Incompatibility

## 1. Metadata

- **Severity**: P2 (Warning) / P1 (if production tools reject legitimate agent calls)
- **Target Component**: MCP Tool Bridge, Tool Registry, Schema Validator (`internal/mcp`)
- **Relevant Alerts**: `MCPToolSchemaDriftDetected`, `MCPToolValidationFailed`, `MCPToolRiskClassMismatch`

---

## 2. Symptoms

- Agents failing to execute tools with `ErrSchemaDrift` or validation errors on parameter schemas.
- Downstream MCP server updated its API without updating the registered AgentMesh tool passport.
- Tool risk classification changed (e.g. from `READ` to `WRITE` or `DESTRUCTIVE`) without updated approval rules.

---

## 3. Immediate Triage (First 5 Minutes)

1. Inspect registered tools and identify schema drift status:

   ```bash
   agentmesh mcp list-tools --tenant=<tenant_id> --show-drift
   ```

2. Run structural diff against target MCP server endpoint:

   ```bash
   agentmesh mcp diff-schema --tool=<tool_name> --server=<mcp_server_url>
   ```

3. Check if recent tool calls were blocked due to newly required approvals:

   ```bash
   agentmesh audit logs --filter="tool=<tool_name>" --last=15m
   ```

---

## 4. Root Cause Analysis

- **Case A: MCP Server Changed Parameter Names or Types (Breaking Drift)**
  - Server added required properties or changed types (e.g., `string` to `integer`).
  - *Mitigation*: AgentMesh halts automatic execution to prevent corrupt database/system updates.
- **Case B: Safe Additive Changes (Non-Breaking Drift)**
  - Server introduced optional parameters or documentation fields.
  - *Fix*: Re-sync and approve updated tool contract in AgentMesh.
- **Case C: Silent Escalation of Tool Risk**
  - Tool previously classified as `READ` now executes stateful updates.
  - *Fix*: Update tool passport to `WRITE`/`DESTRUCTIVE` and configure approval policy.

---

## 5. Remediation Steps

1. **Sync and reconcile updated tool schema**:

   ```bash
   agentmesh mcp sync-tool --tool=<tool_name> --server=<server_id>
   ```

2. **If schema change is breaking and agents need time to adapt, roll back MCP server**:

   ```bash
   kubectl rollout undo deployment/mcp-server-<name>
   ```

3. **If tool requires manual approval, update approval policy**:

   ```bash
   agentmesh policy add-rule --tool=<tool_name> --require-approval=true --tenant=<tenant_id>
   ```

---

## 6. Verification & Recovery Confirmation

- Run test invocation of the affected tool:

  ```bash
  agentmesh mcp test-tool --tool=<tool_name> --params='{"test": true}'
  ```

- Confirm `agentmesh_mcp_tool_drift_total` drops to 0.
- Verify agents successfully complete workflows utilizing the tool.
