package protocol_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/pkg/protocol"
)

func TestA2AProtocol_MarshalingAndStates(t *testing.T) {
	states := []protocol.TaskState{
		protocol.TaskStatePending,
		protocol.TaskStateRunning,
		protocol.TaskStateCompleted,
		protocol.TaskStateFailed,
		protocol.TaskStateCancelled,
		protocol.TaskStateBlocked,
	}

	for _, st := range states {
		if string(st) == "" {
			t.Errorf("empty task state constant")
		}
	}

	card := protocol.AgentCard{
		Name:        "researcher",
		Description: "Market research agent",
		Version:     "v1.0.0",
		Capabilities: []protocol.AgentCapability{
			{Name: "data-mining", Description: "Mines data", Tags: []string{"web", "search"}},
		},
		EndpointURL: "https://agent.internal/a2a",
		Protocols:   []string{"a2a/v1"},
		Authentication: protocol.AuthConfig{
			Type:   "bearer",
			Header: "Authorization",
		},
	}

	data, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("failed to marshal AgentCard: %v", err)
	}

	var decoded protocol.AgentCard
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal AgentCard: %v", err)
	}

	if decoded.Name != "researcher" || len(decoded.Capabilities) != 1 {
		t.Errorf("unexpected decoded card: %+v", decoded)
	}

	// Task Request
	now := time.Now().UTC()
	req := protocol.A2ATaskRequest{
		TaskID:        "task-123",
		CallerAgentID: "agent-caller",
		TargetAgentID: "agent-target",
		Capability:    "data-mining",
		Parameters:    map[string]any{"query": "AI infra"},
		Context: protocol.TaskContext{
			TraceID:         "trace-99",
			SpanID:          "span-1",
			DelegationStack: []string{"agent-caller"},
			MaxDelegation:   5,
			RemainingBudget: 10.0,
			TenantID:        "tenant-1",
		},
		Deadline:        &now,
		StreamRequested: true,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal A2ATaskRequest: %v", err)
	}

	var reqDecoded protocol.A2ATaskRequest
	if err := json.Unmarshal(reqBytes, &reqDecoded); err != nil {
		t.Fatalf("failed to unmarshal A2ATaskRequest: %v", err)
	}
	if reqDecoded.TaskID != "task-123" || reqDecoded.Context.TenantID != "tenant-1" {
		t.Errorf("unexpected decoded request: %+v", reqDecoded)
	}

	// Task Response
	res := protocol.A2ATaskResponse{
		TaskID:      "task-123",
		State:       protocol.TaskStateCompleted,
		Result:      json.RawMessage(`{"status":"done"}`),
		CostUSD:     0.05,
		TotalTokens: 1200,
		Artifacts: []protocol.TaskArtifact{
			{Name: "report.pdf", MimeType: "application/pdf", URI: "gs://bucket/report.pdf"},
		},
	}
	resBytes, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}
	var resDecoded protocol.A2ATaskResponse
	if err := json.Unmarshal(resBytes, &resDecoded); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resDecoded.State != protocol.TaskStateCompleted || len(resDecoded.Artifacts) != 1 {
		t.Errorf("unexpected decoded response: %+v", resDecoded)
	}

	// Stream and Cancel
	stream := protocol.A2AStreamMessage{
		TaskID:    "task-123",
		EventType: "delta",
		Payload:   "chunk 1",
		Timestamp: now,
	}
	cancel := protocol.A2ACancelRequest{
		TaskID: "task-123",
		Reason: "Timeout exceeded",
	}

	if stream.EventType != "delta" || cancel.Reason != "Timeout exceeded" {
		t.Errorf("unexpected stream or cancel values")
	}
}

func TestMCPProtocol_JSONRPCAndTools(t *testing.T) {
	// Constants check
	if protocol.MCPParseError != -32700 || protocol.MCPPolicyDenied != -32001 {
		t.Errorf("unexpected MCP error codes")
	}

	// JSONRPC Request & Response
	rpcReq := protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"read","arguments":{"path":"/tmp"}}`),
	}

	reqBytes, err := json.Marshal(rpcReq)
	if err != nil {
		t.Fatalf("failed to marshal JSONRPCRequest: %v", err)
	}

	var rpcReqDecoded protocol.JSONRPCRequest
	if err := json.Unmarshal(reqBytes, &rpcReqDecoded); err != nil {
		t.Fatalf("failed to unmarshal JSONRPCRequest: %v", err)
	}
	if rpcReqDecoded.Method != "tools/call" {
		t.Errorf("unexpected method: %s", rpcReqDecoded.Method)
	}

	// Call tool result
	toolRes := protocol.MCPCallToolResult{
		Content: []protocol.MCPContent{
			{Type: "text", Text: "Operation complete"},
		},
		IsError: false,
	}
	resBytes, err := json.Marshal(toolRes)
	if err != nil {
		t.Fatalf("failed to marshal tool result: %v", err)
	}
	var toolResDecoded protocol.MCPCallToolResult
	if err := json.Unmarshal(resBytes, &toolResDecoded); err != nil {
		t.Fatalf("failed to unmarshal tool result: %v", err)
	}
	if len(toolResDecoded.Content) != 1 || toolResDecoded.Content[0].Text != "Operation complete" {
		t.Errorf("unexpected tool result: %+v", toolResDecoded)
	}

	// Initialize
	initParams := protocol.MCPInitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo:      map[string]string{"name": "agentmesh-client"},
	}
	initRes := protocol.MCPInitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo:      map[string]string{"name": "bigquery-mcp"},
	}

	if initParams.ProtocolVersion != initRes.ProtocolVersion {
		t.Errorf("mismatched protocol version")
	}
}
