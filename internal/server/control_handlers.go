package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/reconcile"
	"github.com/agentmesh/agentmesh/pkg/spec"
	"github.com/go-chi/chi/v5"
)

// handleListControlActions lists all optimization actions for a tenant.
func (s *Server) handleListControlActions(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	actions, err := s.store.ListOptimizationActions(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(actions)
}

// handleCreateControlAction creates a new typed optimization action.
func (s *Server) handleCreateControlAction(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var action spec.AgentOptimizationAction
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	action.OrganizationID = tenantID
	if action.ActionID == "" {
		action.ActionID = "act_" + action.CapabilityID + "_" + time.Now().Format("20060102150405")
	}
	action.CreatedAt = time.Now().UTC()

	// Evaluate automation policy
	pol, err := s.store.GetAutomationPolicy(r.Context(), tenantID, action.ProjectID)
	if err != nil {
		pol = &policy.AutomationPolicy{OrganizationID: tenantID, Mode: policy.ModeAdvisory}
	}
	polDec := policy.EvaluateActionPolicy(&action, pol)

	// Create durable workflow
	steps := []reconcile.ActionStep{
		{StepNumber: 1, Name: "policy_evaluation", Status: "COMPLETED"},
		{StepNumber: 2, Name: "progressive_delivery", Status: "PENDING"},
		{StepNumber: 3, Name: "outcome_verification", Status: "PENDING"},
	}

	wf, err := s.workflowMgr.CreateWorkflow(&action, steps)
	if err != nil {
		http.Error(w, "failed to create action workflow: "+err.Error(), http.StatusConflict)
		return
	}

	if polDec.Status == policy.PolicyStatusApproved {
		_, _ = s.workflowMgr.Transition(wf.WorkflowID, reconcile.StateApproved)
	} else if polDec.Status == policy.PolicyStatusRequiresApproval {
		_, _ = s.workflowMgr.Transition(wf.WorkflowID, reconcile.StateWaitingForApproval)
	} else {
		_, _ = s.workflowMgr.Transition(wf.WorkflowID, reconcile.StateFailed)
	}

	if err := s.store.SaveOptimizationAction(r.Context(), &action); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"action":   action,
		"workflow": wf,
		"decision": polDec,
	})
}

// handleGetControlAction retrieves an action by ID.
func (s *Server) handleGetControlAction(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}
	actionID := chi.URLParam(r, "id")

	action, err := s.store.GetOptimizationAction(r.Context(), tenantID, actionID)
	if err != nil {
		http.Error(w, "action not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(action)
}

// handleDryRunControlAction executes a dry run of an optimization action.
func (s *Server) handleDryRunControlAction(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}
	actionID := chi.URLParam(r, "id")

	action, err := s.store.GetOptimizationAction(r.Context(), tenantID, actionID)
	if err != nil {
		http.Error(w, "action not found", http.StatusNotFound)
		return
	}

	dryResult, err := s.proxyProvider.DryRun(r.Context(), action)
	if err != nil {
		http.Error(w, "dry run failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dryResult)
}

// handleApproveControlAction cryptographically binds approval to action hash.
func (s *Server) handleApproveControlAction(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}
	actionID := chi.URLParam(r, "id")

	var req struct {
		Approver string `json:"approver"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Approver == "" {
		req.Approver = "authorized-operator"
	}

	action, err := s.store.GetOptimizationAction(r.Context(), tenantID, actionID)
	if err != nil {
		http.Error(w, "action not found", http.StatusNotFound)
		return
	}

	now := time.Now().UTC()
	action.ApprovedAt = &now
	action.ApprovalRequirement.ApprovedBy = append(action.ApprovalRequirement.ApprovedBy, req.Approver)
	action.ApprovalRequirement.ActionHashBound = action.ComputeActionHash()

	_ = s.store.SaveOptimizationAction(r.Context(), action)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"actionId":        action.ActionID,
		"approvedBy":      req.Approver,
		"actionHashBound": action.ApprovalRequirement.ActionHashBound,
		"status":          "APPROVED",
	})
}

// handleExecuteControlAction applies the approved optimization action.
func (s *Server) handleExecuteControlAction(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}
	actionID := chi.URLParam(r, "id")

	action, err := s.store.GetOptimizationAction(r.Context(), tenantID, actionID)
	if err != nil {
		http.Error(w, "action not found", http.StatusNotFound)
		return
	}

	// Verify kill switch is not active
	if frozen, reason := s.freezeMgr.IsFrozen(tenantID, action.ProjectID, action.CapabilityID); frozen {
		http.Error(w, "cannot execute action: "+reason, http.StatusForbidden)
		return
	}

	applyRes, err := s.proxyProvider.Apply(r.Context(), action)
	if err != nil {
		http.Error(w, "apply failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()
	action.CompletedAt = &now
	action.Result = "SUCCESS"
	_ = s.store.SaveOptimizationAction(r.Context(), action)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(applyRes)
}

// handleRollbackControlAction restores the prior configuration.
func (s *Server) handleRollbackControlAction(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}
	actionID := chi.URLParam(r, "id")

	action, err := s.store.GetOptimizationAction(r.Context(), tenantID, actionID)
	if err != nil {
		http.Error(w, "action not found", http.StatusNotFound)
		return
	}

	rbRes, err := s.proxyProvider.Rollback(r.Context(), action)
	if err != nil {
		http.Error(w, "rollback failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()
	action.CompletedAt = &now
	action.Result = "ROLLED_BACK"
	_ = s.store.SaveOptimizationAction(r.Context(), action)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rbRes)
}

// handleFreezeAutomation activates an emergency freeze on automated optimization.
func (s *Server) handleFreezeAutomation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scope    string `json:"scope"`
		ScopeID  string `json:"scopeId"`
		Reason   string `json:"reason"`
		FrozenBy string `json:"frozenBy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Scope == "" {
		req.Scope = "GLOBAL"
		req.ScopeID = "all"
	}
	if req.FrozenBy == "" {
		req.FrozenBy = "security-admin"
	}

	freeze := s.freezeMgr.Freeze(req.Scope, req.ScopeID, req.Reason, req.FrozenBy, nil)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(freeze)
}

// handleUnfreezeAutomation clears an emergency freeze.
func (s *Server) handleUnfreezeAutomation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scope   string `json:"scope"`
		ScopeID string `json:"scopeId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Scope == "" {
		req.Scope = "GLOBAL"
		req.ScopeID = "all"
	}

	unfrozen := s.freezeMgr.Unfreeze(req.Scope, req.ScopeID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"unfrozen": unfrozen})
}

// handleGetCanaryV3 retrieves status for a Canary V3 run.
func (s *Server) handleGetCanaryV3(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "id")
	run, err := s.canaryV3.GetRun(runID)
	if err != nil {
		http.Error(w, "canary run not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(run)
}

// handleListRoutingSpecs lists routing specs for a tenant.
func (s *Server) handleListRoutingSpecs(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	specs, err := s.store.ListRoutingSpecs(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(specs)
}

// handleSaveRoutingSpec saves a desired routing specification.
func (s *Server) handleSaveRoutingSpec(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}

	var rSpec spec.AgentRoutingSpec
	if err := json.NewDecoder(r.Body).Decode(&rSpec); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	rSpec.OrganizationID = tenantID
	rSpec.UpdatedAt = time.Now().UTC()

	if err := s.store.SaveRoutingSpec(r.Context(), &rSpec); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rSpec)
}

// handleListProductionOutcomes lists verified production outcomes.
func (s *Server) handleListProductionOutcomes(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "default"
	}
	capabilityID := r.URL.Query().Get("capability")

	outcomes, err := s.store.ListProductionOutcomes(r.Context(), tenantID, capabilityID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(outcomes)
}
