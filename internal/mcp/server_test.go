package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/agentmesh/agentmesh/internal/mcp"
	"github.com/agentmesh/agentmesh/pkg/protocol"
)

func TestAgentMeshMCPIntelligenceServer(t *testing.T) {
	server := mcp.NewAgentMeshMCPServer("tenant_test", func(ctx context.Context, toolName string, args map[string]any) (string, error) {
		if toolName == "inspect_agent" {
			return `{"agentId":"finance-agent","status":"ACTIVE"}`, nil
		}
		return "ok", nil
	})

	// 1. Test initialize
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n"
	inBuf := bytes.NewBufferString(initReq)
	outBuf := &bytes.Buffer{}

	if err := server.ServeStdio(inBuf, outBuf); err != nil {
		t.Fatalf("unexpected error serving stdio: %v", err)
	}

	var resp protocol.JSONRPCResponse
	if err := json.Unmarshal(outBuf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse initialize response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected json-rpc error: %v", resp.Error)
	}

	// 2. Test tools/list
	listReq := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}` + "\n"
	inBuf = bytes.NewBufferString(listReq)
	outBuf.Reset()

	_ = server.ServeStdio(inBuf, outBuf)
	var listResp protocol.JSONRPCResponse
	_ = json.Unmarshal(outBuf.Bytes(), &listResp)
	if listResp.Result == nil {
		t.Fatalf("expected tools list result")
	}

	// 3. Test tools/call inspect_agent
	callReq := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"inspect_agent","arguments":{"agentId":"finance-agent"}}}` + "\n"
	inBuf = bytes.NewBufferString(callReq)
	outBuf.Reset()

	_ = server.ServeStdio(inBuf, outBuf)
	if !strings.Contains(outBuf.String(), "finance-agent") {
		t.Fatalf("expected output to contain finance-agent, got: %s", outBuf.String())
	}
}
