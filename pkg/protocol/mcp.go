package protocol

import (
	"encoding/json"
)

// MCP JSON-RPC 2.0 Standard Error Codes
const (
	MCPParseError     = -32700
	MCPInvalidRequest = -32600
	MCPMethodNotFound = -32601
	MCPInvalidParams  = -32602
	MCPInternalError  = -32603

	// AgentMesh Specific Policy/Security Error Codes
	MCPPolicyDenied       = -32001
	MCPApprovalRequired   = -32002
	MCPBudgetExceeded     = -32003
	MCPCircuitBreakerOpen = -32004
	MCPRateLimited        = -32005
)

// JSONRPCRequest represents a standard JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"` // Must be "2.0"
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a standard JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError holds error details for a failed JSON-RPC message.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// MCPTool defines a registered MCP tool specification.
type MCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"` // JSON Schema
}

// MCPToolsListResult is returned by the 'tools/list' method.
type MCPToolsListResult struct {
	Tools []MCPTool `json:"tools"`
}

// MCPCallToolParams is passed in the 'tools/call' method.
type MCPCallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// MCPContent is a typed content item returned from an MCP tool call.
type MCPContent struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
}

// MCPCallToolResult is returned by the 'tools/call' method.
type MCPCallToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// MCPInitializeParams is passed when initializing an MCP session.
type MCPInitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]any         `json:"capabilities"`
	ClientInfo      map[string]string      `json:"clientInfo"`
}

// MCPInitializeResult is returned upon successful MCP session initialization.
type MCPInitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      map[string]string `json:"serverInfo"`
}
