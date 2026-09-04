package a2a

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/agentmesh/agentmesh/pkg/protocol"
)

func FuzzA2AMessageDecode(f *testing.F) {
	f.Add([]byte(`{"taskId":"task-123","callerAgentId":"agent-a","targetAgentId":"agent-b","capability":"summarize"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"context":{"delegationStack":["a","b","c"]}}`))

	server := NewServer(&protocol.AgentCard{Name: "test-card"}, nil, nil)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Test protocol unmarshaling
		var req protocol.A2ATaskRequest
		_ = json.Unmarshal(data, &req)

		var card protocol.AgentCard
		_ = json.Unmarshal(data, &card)

		// Test HTTP invoke handler with fuzz body
		httpReq := httptest.NewRequest("POST", "/a2a/invoke", bytes.NewReader(data))
		rec := httptest.NewRecorder()
		server.HandleInvoke(rec, httpReq)
	})
}
