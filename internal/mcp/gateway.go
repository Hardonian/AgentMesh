package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/agentmesh/agentmesh/internal/approval"
	"github.com/agentmesh/agentmesh/internal/budgets"
	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/reliability"
	"github.com/agentmesh/agentmesh/internal/telemetry"
	"github.com/agentmesh/agentmesh/pkg/protocol"
)

// UpstreamToolHandler executes tool logic on the underlying MCP server.
type UpstreamToolHandler func(ctx context.Context, toolName string, args map[string]any) (*protocol.MCPCallToolResult, error)

// Gateway is the standards-compliant MCP reverse proxy with MCPGuard policy enforcement.
type Gateway struct {
	mu             sync.RWMutex
	policyEngine   *policy.Engine
	approvalSvc    *approval.Service
	budgetTracker  *budgets.Tracker
	circuitBreakers map[string]*reliability.CircuitBreaker
	registeredTools map[string]protocol.MCPTool
	upstreamHandler UpstreamToolHandler
}

// NewGateway constructs an MCP Gateway.
func NewGateway(engine *policy.Engine, appSvc *approval.Service, bt *budgets.Tracker, upstream UpstreamToolHandler) *Gateway {
	return &Gateway{
		policyEngine:    engine,
		approvalSvc:     appSvc,
		budgetTracker:   bt,
		circuitBreakers: make(map[string]*reliability.CircuitBreaker),
		registeredTools: make(map[string]protocol.MCPTool),
		upstreamHandler: upstream,
	}
}

// RegisterTool adds a known tool to the MCP Gateway.
func (g *Gateway) RegisterTool(tool protocol.MCPTool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.registeredTools[tool.Name] = tool
}

// HandleRPC processes a raw JSON-RPC message from an MCP client.
func (g *Gateway) HandleRPC(ctx context.Context, tenantID, agentID string, rawReq []byte) *protocol.JSONRPCResponse {
	var req protocol.JSONRPCRequest
	if err := json.Unmarshal(rawReq, &req); err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &protocol.JSONRPCError{
				Code:    protocol.MCPParseError,
				Message: "failed to parse JSON-RPC request",
			},
		}
	}

	switch req.Method {
	case "initialize":
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: protocol.MCPInitializeResult{
				ProtocolVersion: "2024-11-05",
				Capabilities: map[string]any{
					"tools": map[string]any{"listChanged": true},
				},
				ServerInfo: map[string]string{
					"name":    "agentmesh-mcp-gateway",
					"version": "1.0.0",
				},
			},
		}

	case "tools/list":
		g.mu.RLock()
		defer g.mu.RUnlock()

		var allowedTools []protocol.MCPTool
		for _, tool := range g.registeredTools {
			// Policy check: only expose tools permitted to this agent
			if g.policyEngine != nil {
				dec := g.policyEngine.Evaluate(ctx, &policy.EvaluationRequest{
					TenantID:       tenantID,
					SubjectAgentID: agentID,
					Tool:           tool.Name,
					Action:         "list",
				})
				if dec.Effect == policy.EffectDeny {
					continue // Hidden from client by policy
				}
			}
			allowedTools = append(allowedTools, tool)
		}
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  protocol.MCPToolsListResult{Tools: allowedTools},
		}

	case "tools/call":
		var params protocol.MCPCallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &protocol.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &protocol.JSONRPCError{
					Code:    protocol.MCPInvalidParams,
					Message: "invalid tools/call params",
				},
			}
		}

		return g.executeToolCall(ctx, req.ID, tenantID, agentID, &params)

	default:
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &protocol.JSONRPCError{
				Code:    protocol.MCPMethodNotFound,
				Message: fmt.Sprintf("unsupported method %q", req.Method),
			},
		}
	}
}

func (g *Gateway) executeToolCall(ctx context.Context, reqID any, tenantID, agentID string, params *protocol.MCPCallToolParams) *protocol.JSONRPCResponse {
	// 1. Circuit Breaker Check
	cb := g.getCircuitBreaker(params.Name)
	if !cb.Allow() {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      reqID,
			Error: &protocol.JSONRPCError{
				Code:    protocol.MCPCircuitBreakerOpen,
				Message: fmt.Sprintf("circuit breaker OPEN for tool %q", params.Name),
			},
		}
	}

	// 2. Policy Authorization
	if g.policyEngine != nil {
		dec := g.policyEngine.Evaluate(ctx, &policy.EvaluationRequest{
			TenantID:       tenantID,
			SubjectAgentID: agentID,
			Tool:           params.Name,
			Action:         "execute",
		})

		switch dec.Effect {
		case policy.EffectDeny:
			return &protocol.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      reqID,
				Error: &protocol.JSONRPCError{
					Code:    protocol.MCPPolicyDenied,
					Message: fmt.Sprintf("tool execution denied by policy: %s", dec.Reason),
					Data:    dec,
				},
			}

		case policy.EffectRequireApproval:
			// Check if valid approval token was supplied
			approvalToken, _ := params.Arguments["_approvalToken"].(string)
			requestID, _ := params.Arguments["_approvalRequestId"].(string)

			// Clean internal parameters before checking hash
			cleanArgs := make(map[string]any)
			for k, v := range params.Arguments {
				if k != "_approvalToken" && k != "_approvalRequestId" {
					cleanArgs[k] = v
				}
			}

			if approvalToken == "" || requestID == "" {
				// Create pending approval request
				var appReq *approval.Request
				if g.approvalSvc != nil {
					appReq, _ = g.approvalSvc.CreateRequest(tenantID, agentID, params.Name, "execute", cleanArgs, dec.PolicyID, dec.DecisionVersion, dec.Reason, 15*time.Minute)
				}
				return &protocol.JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      reqID,
					Error: &protocol.JSONRPCError{
						Code:    protocol.MCPApprovalRequired,
						Message: fmt.Sprintf("human approval required for tool %q: %s", params.Name, dec.Reason),
						Data:    appReq,
					},
				}
			}

			// Validate provided token against clean arguments
			if err := g.approvalSvc.ValidateApproval(requestID, agentID, params.Name, cleanArgs, approvalToken); err != nil {
				return &protocol.JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      reqID,
					Error: &protocol.JSONRPCError{
						Code:    protocol.MCPApprovalRequired,
						Message: fmt.Sprintf("approval validation failed: %v", err),
					},
				}
			}
			// Approved! Fall through to execution
		}
	}

	// 3. Upstream Execution
	if g.upstreamHandler == nil {
		cb.RecordResult(true)
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      reqID,
			Result: protocol.MCPCallToolResult{
				Content: []protocol.MCPContent{
					{Type: "text", Text: fmt.Sprintf("tool %s executed successfully (mock)", params.Name)},
				},
			},
		}
	}

	res, err := g.upstreamHandler(ctx, params.Name, params.Arguments)
	if err != nil {
		cb.RecordResult(false)
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      reqID,
			Error: &protocol.JSONRPCError{
				Code:    protocol.MCPInternalError,
				Message: telemetry.ScrubSecrets(err.Error()),
			},
		}
	}

	cb.RecordResult(true)
	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      reqID,
		Result:  res,
	}
}

func (g *Gateway) getCircuitBreaker(toolName string) *reliability.CircuitBreaker {
	g.mu.Lock()
	defer g.mu.Unlock()

	cb, exists := g.circuitBreakers[toolName]
	if !exists {
		cb = reliability.NewCircuitBreaker(toolName, 3, 2, 30*time.Second)
		g.circuitBreakers[toolName] = cb
	}
	return cb
}
