package mcp

import (
	"bytes"
	"context"
	"testing"
)

func FuzzMCPFrameDecode(f *testing.F) {
	f.Add([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	f.Add([]byte(`{"jsonrpc":"2.0","id":"abc","method":"tools/call","params":{"name":"inspect_agent","arguments":{"agentId":"agent-1"}}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`not json`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"unknown_method"}`))

	server := NewAgentMeshMCPServer("tenant-test", func(ctx context.Context, toolName string, args map[string]any) (string, error) {
		return "ok", nil
	})

	f.Fuzz(func(t *testing.T, data []byte) {
		in := bytes.NewReader(append(data, '\n'))
		out := &bytes.Buffer{}
		_ = server.ServeStdio(in, out)
	})
}
