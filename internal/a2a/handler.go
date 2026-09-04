package a2a

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/pkg/protocol"
)

var (
	ErrTaskNotFound = errors.New("a2a task not found")
)

// Firewall enforces policy boundaries on A2A agent-to-agent interactions.
type Firewall struct {
	policyEngine *policy.Engine
}

func NewFirewall(engine *policy.Engine) *Firewall {
	return &Firewall{policyEngine: engine}
}

// AuthorizeTaskInvocation evaluates whether caller is allowed to invoke target with capability.
func (f *Firewall) AuthorizeTaskInvocation(ctx context.Context, req *protocol.A2ATaskRequest) (*policy.Decision, error) {
	if f.policyEngine == nil {
		return &policy.Decision{Effect: policy.EffectAllow}, nil
	}

	dec := f.policyEngine.Evaluate(ctx, &policy.EvaluationRequest{
		TenantID:        req.Context.TenantID,
		SubjectAgentID:  req.CallerAgentID,
		Capability:      req.Capability,
		Action:          "invoke",
		Resource:        req.TargetAgentID,
		DelegationDepth: len(req.Context.DelegationStack),
		DelegationStack: req.Context.DelegationStack,
	})

	if dec.Effect == policy.EffectDeny {
		return dec, fmt.Errorf("a2a firewall blocked invocation: %s", dec.Reason)
	}

	return dec, nil
}

// Server provides standard A2A endpoints for an agent.
type Server struct {
	card      *protocol.AgentCard
	firewall  *Firewall
	tasksMu   sync.RWMutex
	tasks     map[string]*protocol.A2ATaskResponse
	executeFn func(ctx context.Context, req *protocol.A2ATaskRequest) (*protocol.A2ATaskResponse, error)
}

func NewServer(card *protocol.AgentCard, firewall *Firewall, execFn func(ctx context.Context, req *protocol.A2ATaskRequest) (*protocol.A2ATaskResponse, error)) *Server {
	return &Server{
		card:      card,
		firewall:  firewall,
		tasks:     make(map[string]*protocol.A2ATaskResponse),
		executeFn: execFn,
	}
}

// HandleAgentCard returns the standard agent card JSON.
func (s *Server) HandleAgentCard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.card)
}

// HandleInvoke processes task invocation under A2A Firewall protection.
func (s *Server) HandleInvoke(w http.ResponseWriter, r *http.Request) {
	var req protocol.A2ATaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid json: %v", err), http.StatusBadRequest)
		return
	}

	// 1. Firewall evaluation
	if s.firewall != nil {
		dec, err := s.firewall.AuthorizeTaskInvocation(r.Context(), &req)
		if err != nil {
			resp := protocol.A2ATaskResponse{
				TaskID: req.TaskID,
				State:  protocol.TaskStateFailed,
				Error: &protocol.TaskError{
					Code:    "A2A_FIREWALL_DENIED",
					Message: err.Error(),
					Details: dec,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
	}

	// 2. Execute task
	now := time.Now().UTC()
	var taskResp *protocol.A2ATaskResponse
	var err error

	if s.executeFn != nil {
		taskResp, err = s.executeFn(r.Context(), &req)
	} else {
		taskResp = &protocol.A2ATaskResponse{
			TaskID:      req.TaskID,
			State:       protocol.TaskStateCompleted,
			Result:      json.RawMessage(`{"status":"completed"}`),
			CompletedAt: &now,
		}
	}

	if err != nil {
		taskResp = &protocol.A2ATaskResponse{
			TaskID: req.TaskID,
			State:  protocol.TaskStateFailed,
			Error: &protocol.TaskError{
				Code:    "EXECUTION_FAILED",
				Message: err.Error(),
			},
		}
	}

	s.tasksMu.Lock()
	s.tasks[req.TaskID] = taskResp
	s.tasksMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(taskResp)
}

// HandleGetTask returns task status.
func (s *Server) HandleGetTask(w http.ResponseWriter, r *http.Request, taskID string) {
	s.tasksMu.RLock()
	resp, exists := s.tasks[taskID]
	s.tasksMu.RUnlock()

	if !exists {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
