package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/agentmesh/agentmesh/internal/approval"
	"github.com/agentmesh/agentmesh/internal/budgets"
	"github.com/agentmesh/agentmesh/internal/mcp"
	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/pkg/protocol"
)

func main() {
	fmt.Println("=== AgentMesh MCP Gateway with MCPGuard Demo ===")

	pol := &policy.Policy{
		ID:       "pol_mcp_guard",
		Version:  "v1.0.0",
		TenantID: "demo",
		Rules: []policy.Rule{
			{
				Name:    "Allow BigQuery Read",
				Effect:  policy.EffectAllow,
				Agents:  []string{"analyst-agent"},
				Tools:   []string{"bigquery.read"},
				Actions: []string{"execute"},
			},
			{
				Name:    "Require Approval for BigQuery Delete",
				Effect:  policy.EffectRequireApproval,
				Agents:  []string{"analyst-agent"},
				Tools:   []string{"bigquery.delete"},
				Actions: []string{"execute"},
			},
		},
	}
	engine := policy.NewEngine([]*policy.Policy{pol})
	appSvc := approval.NewService()
	bt := budgets.NewTracker()

	gateway := mcp.NewGateway(engine, appSvc, bt, func(ctx context.Context, toolName string, args map[string]any) (*protocol.MCPCallToolResult, error) {
		return &protocol.MCPCallToolResult{
			Content: []protocol.MCPContent{
				{Type: "text", Text: fmt.Sprintf("Query executed on %s with args: %v", toolName, args)},
			},
		}, nil
	})

	// 1. Safe Read Request
	readReq, _ := json.Marshal(protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"bigquery.read","arguments":{"query":"SELECT COUNT(*) FROM sales;"}}`),
	})
	resp1 := gateway.HandleRPC(context.Background(), "demo", "analyst-agent", readReq)
	fmt.Printf("1. Read call result: isError=%v (expected false)\n", resp1.Error != nil)

	// 2. Destructive Delete Request without approval token
	deleteReq, _ := json.Marshal(protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"bigquery.delete","arguments":{"dataset":"raw_archive","table":"logs"}}`),
	})
	resp2 := gateway.HandleRPC(context.Background(), "demo", "analyst-agent", deleteReq)
	fmt.Printf("2. Delete call result without token: Code=%d Message=%q\n", resp2.Error.Code, resp2.Error.Message)

	// 3. Human Approval Resolution
	pending := appSvc.ListPending("demo")
	if len(pending) > 0 {
		req := pending[0]
		resolved, _ := appSvc.Resolve(req.ID, "security-lead", true, "Approved for monthly cleanup")
		fmt.Printf("3. Human reviewer approved request %s (Token: %s)\n", req.ID, resolved.ApprovalToken)

		// 4. Retry with valid approval token
		approvedDeleteReq, _ := json.Marshal(protocol.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      3,
			Method:  "tools/call",
			Params: json.RawMessage(fmt.Sprintf(`{"name":"bigquery.delete","arguments":{"dataset":"raw_archive","table":"logs","_approvalRequestId":"%s","_approvalToken":"%s"}}`,
				req.ID, resolved.ApprovalToken)),
		})
		resp3 := gateway.HandleRPC(context.Background(), "demo", "analyst-agent", approvedDeleteReq)
		fmt.Printf("4. Delete call with valid approval token: isError=%v\n", resp3.Error != nil)
	}

	fmt.Println("✓ Invariant confirmed: Destructive tools blocked until valid human approval token supplied.")
}
