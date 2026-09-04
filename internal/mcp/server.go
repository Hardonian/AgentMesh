package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/agentmesh/agentmesh/pkg/protocol"
)

// AgentMeshMCPServer exposes read-only AgentMesh intelligence via the Model Context Protocol.
type AgentMeshMCPServer struct {
	tenantID   string
	tools      []protocol.MCPTool
	dispatchFn func(ctx context.Context, toolName string, args map[string]any) (string, error)
}

// NewAgentMeshMCPServer creates the AgentMesh MCP Intelligence Server.
func NewAgentMeshMCPServer(tenantID string, dispatch func(ctx context.Context, toolName string, args map[string]any) (string, error)) *AgentMeshMCPServer {
	server := &AgentMeshMCPServer{
		tenantID:   tenantID,
		dispatchFn: dispatch,
		tools: []protocol.MCPTool{
			{
				Name:        "inspect_agent",
				Description: "Inspect an agent's registered contract, identity, and lifecycle status.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"agentId":{"type":"string"}},"required":["agentId"]}`),
			},
			{
				Name:        "get_agent_passport",
				Description: "Retrieve an agent's Agent Passport V2 operational evidence, reliability, and provenance.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"agentId":{"type":"string"}},"required":["agentId"]}`),
			},
			{
				Name:        "inspect_graph",
				Description: "Retrieve an agent's operational AgentGraph workflow topology and risk findings.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"agentId":{"type":"string"}},"required":["agentId"]}`),
			},
			{
				Name:        "explain_policy",
				Description: "Evaluate deterministic policy for an agent and tool, returning an auditable reason.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"agentId":{"type":"string"},"tool":{"type":"string"},"action":{"type":"string"}},"required":["agentId","tool"]}`),
			},
			{
				Name:        "simulate_route",
				Description: "Simulate capability-based task routing across eligible agents with multi-stage ranking.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"capability":{"type":"string"},"strategy":{"type":"string"}},"required":["capability"]}`),
			},
			{
				Name:        "get_evaluation",
				Description: "Retrieve benchmark evaluation scorecards and regression reports.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"agentId":{"type":"string"}},"required":["agentId"]}`),
			},
			{
				Name:        "compare_versions",
				Description: "Compare two agent versions for change impact and security-sensitive delta.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"agentId":{"type":"string"},"versionA":{"type":"string"},"versionB":{"type":"string"}},"required":["agentId","versionA","versionB"]}`),
			},
			{
				Name:        "get_tool_risk",
				Description: "Get risk classification, fingerprint, and schema drift status for an MCP tool.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"toolName":{"type":"string"}},"required":["toolName"]}`),
			},
			{
				Name:        "explain_route",
				Description: "Reconstruct and explain candidate eligibility, scoring, and reasons for a routing decision.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"taskId":{"type":"string"}},"required":["taskId"]}`),
			},
			{
				Name:        "get_reliability",
				Description: "Retrieve statistical rolling reliability profile and P50/P95 latency percentiles for an agent.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"agentId":{"type":"string"},"capabilityId":{"type":"string"}},"required":["agentId"]}`),
			},
			{
				Name:        "get_capability_health",
				Description: "Retrieve aggregated operational health status and SLO compliance across agents supporting a capability.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"capabilityId":{"type":"string"}},"required":["capabilityId"]}`),
			},
			{
				Name:        "get_slo_status",
				Description: "Evaluate and return AgentSLO compliance status and remaining error budget.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"agentId":{"type":"string"},"capabilityId":{"type":"string"}},"required":["agentId"]}`),
			},
			{
				Name:        "compare_agents",
				Description: "Compare two agents on empirical reliability, cost efficiency, and latency.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"agentA":{"type":"string"},"agentB":{"type":"string"},"capabilityId":{"type":"string"}},"required":["agentA","agentB"]}`),
			},
			{
				Name:        "get_route_history",
				Description: "Retrieve recent canonical routing outcomes and failure taxonomy events.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"capabilityId":{"type":"string"},"limit":{"type":"integer"}},"required":[]}`),
			},
			{
				Name:        "show_recommendation",
				Description: "Retrieve pending candidate route optimization recommendations and cost/latency savings.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"capabilityId":{"type":"string"}},"required":["capabilityId"]}`),
			},
			{
				Name:        "show_action",
				Description: "Inspect an optimization action, dry-run diff, and risk classification.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"actionId":{"type":"string"}},"required":["actionId"]}`),
			},
			{
				Name:        "show_canary",
				Description: "Retrieve multi-stage progress, stage metrics, and rollback triggers for a canary rollout.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"runId":{"type":"string"}},"required":["runId"]}`),
			},
			{
				Name:        "explain_route_change",
				Description: "Explain the architectural, risk, and economic delta between current and proposed routing specs.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"capabilityId":{"type":"string"}},"required":["capabilityId"]}`),
			},
			{
				Name:        "show_rollback",
				Description: "Show the deterministic rollback plan, verification criteria, and last known good state for an action.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"actionId":{"type":"string"}},"required":["actionId"]}`),
			},
			{
				Name:        "show_autonomy_readiness",
				Description: "Evaluate and report structured autonomy readiness dimensions (telemetry health, SLOs, rollback drills).",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"capabilityId":{"type":"string"}},"required":["capabilityId"]}`),
			},
		},
	}
	return server
}

// ServeStdio executes the MCP JSON-RPC protocol loop over an input reader and output writer.
func (s *AgentMeshMCPServer) ServeStdio(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	encoder := json.NewEncoder(w)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req protocol.JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			_ = encoder.Encode(&protocol.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error: &protocol.JSONRPCError{
					Code:    protocol.MCPParseError,
					Message: fmt.Sprintf("invalid json-rpc: %v", err),
				},
			})
			continue
		}

		resp := s.handleRequest(context.Background(), &req)
		if resp != nil {
			_ = encoder.Encode(resp)
		}
	}

	return scanner.Err()
}

func (s *AgentMeshMCPServer) handleRequest(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]string{
					"name":    "agentmesh-intelligence-server",
					"version": "2.0.0",
				},
				"capabilities": map[string]any{
					"tools": map[string]bool{"listChanged": false},
				},
			},
		}

	case "tools/list":
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: protocol.MCPToolsListResult{
				Tools: s.tools,
			},
		}

	case "tools/call":
		var params protocol.MCPCallToolParams
		paramBytes, _ := json.Marshal(req.Params)
		if err := json.Unmarshal(paramBytes, &params); err != nil {
			return &protocol.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &protocol.JSONRPCError{
					Code:    protocol.MCPInvalidParams,
					Message: "invalid tool call params",
				},
			}
		}

		outputStr := fmt.Sprintf("AgentMesh Intelligence result for %s", params.Name)
		var err error
		if s.dispatchFn != nil {
			outputStr, err = s.dispatchFn(ctx, params.Name, params.Arguments)
		}

		if err != nil {
			return &protocol.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: protocol.MCPCallToolResult{
					IsError: true,
					Content: []protocol.MCPContent{
						{Type: "text", Text: fmt.Sprintf("Execution error: %v", err)},
					},
				},
			}
		}

		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: protocol.MCPCallToolResult{
				IsError: false,
				Content: []protocol.MCPContent{
					{Type: "text", Text: outputStr},
				},
			},
		}

	default:
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &protocol.JSONRPCError{
				Code:    protocol.MCPMethodNotFound,
				Message: fmt.Sprintf("unsupported method: %s", req.Method),
			},
		}
	}
}
