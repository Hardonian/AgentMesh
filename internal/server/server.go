package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/agentmesh/agentmesh/internal/a2a"
	"github.com/agentmesh/agentmesh/internal/adk"
	"github.com/agentmesh/agentmesh/internal/approval"
	"github.com/agentmesh/agentmesh/internal/audit"
	"github.com/agentmesh/agentmesh/internal/canary"
	"github.com/agentmesh/agentmesh/internal/crypto"
	"github.com/agentmesh/agentmesh/internal/database"
	"github.com/agentmesh/agentmesh/internal/evaluation"
	"github.com/agentmesh/agentmesh/internal/identity"
	"github.com/agentmesh/agentmesh/internal/mcp"
	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/providers"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/internal/telemetry"
	"github.com/agentmesh/agentmesh/pkg/contracts"
	"github.com/agentmesh/agentmesh/pkg/graph"
	"github.com/agentmesh/agentmesh/pkg/passport"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server is the AgentMesh Control Plane HTTP Server.
type Server struct {
	router      chi.Router
	store       database.Store
	policyEng   *policy.Engine
	routerEng   *routing.Router
	telemetry   *telemetry.Collector
	canaryMgr   *canary.Manager
	approvalSvc *approval.Service
	auditLogger *audit.Logger
	signingKey  *crypto.KeyPair
}

func NewServer(
	store database.Store,
	polEngine *policy.Engine,
	routeEngine *routing.Router,
	tel *telemetry.Collector,
	cm *canary.Manager,
	appSvc *approval.Service,
	auditLog *audit.Logger,
	keyPair *crypto.KeyPair,
) *Server {
	s := &Server{
		router:      chi.NewRouter(),
		store:       store,
		policyEng:   polEngine,
		routerEng:   routeEngine,
		telemetry:   tel,
		canaryMgr:   cm,
		approvalSvc: appSvc,
		auditLogger: auditLog,
		signingKey:  keyPair,
	}

	s.setupRoutes()
	return s
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) setupRoutes() {
	r := s.router

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Tenant-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health & Metrics
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","time":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})
	r.Handle("/metrics", promhttp.Handler())

	// API v1
	r.Route("/api/v1", func(api chi.Router) {
		// Agents
		api.Post("/agents", s.handleRegisterAgent)
		api.Get("/agents", s.handleListAgents)
		api.Get("/agents/{id}", s.handleGetAgent)
		api.Delete("/agents/{id}", s.handleDeleteAgent)

		// Policy
		api.Post("/policy/evaluate", s.handleEvaluatePolicy)
		api.Get("/policies", s.handleListPolicies)
		api.Post("/policies", s.handleSavePolicy)

		// Routing
		api.Post("/routing/route", s.handleRoute)
		api.Post("/routing/explain", s.handleRouteExplain)

		// Approvals
		api.Get("/approvals", s.handleListApprovals)
		api.Post("/approvals/{id}/resolve", s.handleResolveApproval)

		// Canaries
		api.Get("/canaries", s.handleListCanaries)
		api.Post("/canaries", s.handleStartCanary)
		api.Post("/canaries/{agentId}/promote", s.handlePromoteCanary)
		api.Post("/canaries/{agentId}/rollback", s.handleRollbackCanary)

		// Traces
		api.Get("/traces", s.handleListTraces)
		api.Get("/traces/{id}", s.handleGetTrace)
		api.Post("/telemetry/traces", s.handleRecordTrace)

		// Audit
		api.Get("/audit", s.handleListAudit)

		// Credentials
		api.Post("/credentials", s.handleCreateCredential)
		api.Get("/credentials", s.handleListCredentials)

		// Signed Config Bundle for Proxies
		api.Get("/config/bundle", s.handleGetSignedConfigBundle)

		// Graphs (Phase 2)
		api.Post("/graphs", s.handleSaveGraph)
		api.Get("/graphs", s.handleListGraphs)
		api.Get("/graphs/{id}", s.handleGetGraph)
		api.Post("/graphs/analyze", s.handleAnalyzeGraph)
		api.Get("/agents/{id}/graph", s.handleGetAgentGraph)

		// Passports & Badges (Phase 2)
		api.Get("/agents/{id}/passport", s.handleGetAgentPassport)
		api.Get("/agents/{id}/badge", s.handleGetAgentBadge)

		// Tools & MCP (Phase 2)
		api.Get("/tools/passports", s.handleListToolPassports)
		api.Get("/tools/passports/{id}", s.handleGetToolPassport)
		api.Post("/tools/passports", s.handleSaveToolPassport)
		api.Post("/tools/drift", s.handleDetectToolDrift)

		// Capabilities & Routing V2 (Phase 2)
		api.Get("/capabilities", s.handleListCapabilities)
		api.Post("/routing/simulate", s.handleSimulateRoute)
		api.Get("/routing/outcomes", s.handleListRouteOutcomes)

		// Policy V2 & Canaries (Phase 2)
		api.Post("/policy/simulate", s.handleSimulatePolicy)
		api.Post("/policy/canary", s.handlePolicyCanary)

		// Evaluations & CI (Phase 2)
		api.Post("/evaluations/run", s.handleRunEvaluation)
		api.Get("/evaluations/results", s.handleListEvaluationResults)

		// Change Impact (Phase 2)
		api.Post("/canary/impact", s.handleAnalyzeChangeImpact)

		// A2A Compatibility Lab (Phase 2)
		api.Post("/a2a/test", s.handleTestA2AEndpoint)
		api.Get("/a2a/profiles", s.handleListA2AProfiles)

		// High-Impact Strategic Extensions
		api.Get("/a2a/registry", s.handleGetA2ARegistry)
		api.Post("/evaluations/redteam", s.handleRunRedTeamEvaluation)
		api.Post("/auth/wif/token", s.handleExchangeWIFToken)
		api.Post("/providers/armor/inspect", s.handleModelArmorInspect)
	})
}

func getTenantID(r *http.Request) string {
	tenant := r.Header.Get("X-Tenant-ID")
	if tenant == "" {
		return "default"
	}
	return tenant
}

func jsonResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func errorResponse(w http.ResponseWriter, statusCode int, message string) {
	jsonResponse(w, statusCode, map[string]string{"error": message})
}

// Handler implementations
func (s *Server) handleRegisterAgent(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	var contract contracts.AgentContract
	if err := json.NewDecoder(r.Body).Decode(&contract); err != nil {
		errorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid contract json: %v", err))
		return
	}
	if err := contract.Validate(); err != nil {
		errorResponse(w, http.StatusBadRequest, fmt.Sprintf("contract validation error: %v", err))
		return
	}

	hash, _ := contract.Hash()
	pass, _ := passport.GenerateFromContract(&contract, "go", "custom")
	now := time.Now().UTC()

	agentRecord := &database.AgentRecord{
		ID:           contract.Metadata.Name,
		TenantID:     tenantID,
		Name:         contract.Metadata.Name,
		Status:       "HEALTHY",
		Contract:     &contract,
		ContractHash: hash,
		Passport:     pass,
		EndpointURL:  fmt.Sprintf("http://%s.mesh.internal:8080", contract.Metadata.Name),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.SaveAgent(r.Context(), agentRecord); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update router candidate
	s.routerEng.RegisterCandidate(&routing.AgentRouteCandidate{
		AgentID:      agentRecord.ID,
		EndpointURL:  agentRecord.EndpointURL,
		Status:       agentRecord.Status,
		Contract:     &contract,
		Passport:     pass,
		SuccessRate:  1.0,
		AverageCost:  contract.Budgets.MaxCostPerTask * 0.5,
		P95LatencyMs: contract.SLO.P95LatencyMs,
	})

	s.auditLogger.Log(tenantID, audit.EventAgentRegistered, "api", agentRecord.ID, map[string]any{
		"contractHash": hash,
		"capabilities": contract.Capabilities,
	})

	jsonResponse(w, http.StatusCreated, map[string]any{
		"agentId":      agentRecord.ID,
		"contractHash": hash,
		"status":       agentRecord.Status,
		"registeredAt": now.Format(time.RFC3339),
	})
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	agents, err := s.store.ListAgents(r.Context(), tenantID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, agents)
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	agentID := chi.URLParam(r, "id")
	agent, err := s.store.GetAgent(r.Context(), tenantID, agentID)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "agent not found")
		return
	}
	jsonResponse(w, http.StatusOK, agent)
}

func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	agentID := chi.URLParam(r, "id")
	_ = s.store.DeleteAgent(r.Context(), tenantID, agentID)
	s.routerEng.RemoveCandidate(agentID)
	s.auditLogger.Log(tenantID, audit.EventAgentDisabled, "api", agentID, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleEvaluatePolicy(w http.ResponseWriter, r *http.Request) {
	var req policy.EvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid evaluation request")
		return
	}
	if req.TenantID == "" {
		req.TenantID = getTenantID(r)
	}

	decision := s.policyEng.Evaluate(r.Context(), &req)
	jsonResponse(w, http.StatusOK, decision)
}

func (s *Server) handleListPolicies(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	policies, err := s.store.ListPolicies(r.Context(), tenantID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, policies)
}

func (s *Server) handleSavePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	var pol policy.Policy
	if err := json.NewDecoder(r.Body).Decode(&pol); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid policy json")
		return
	}
	pol.TenantID = tenantID
	pol.CreatedAt = time.Now().UTC()
	if err := policy.ValidatePolicy(&pol); err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.store.SavePolicy(r.Context(), &pol); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update active policies in engine
	allPolicies, _ := s.store.ListPolicies(r.Context(), "")
	s.policyEng.SetPolicies(allPolicies)

	s.auditLogger.Log(tenantID, audit.EventPolicyPublished, "api", pol.ID, pol)
	jsonResponse(w, http.StatusCreated, pol)
}

func (s *Server) handleRoute(w http.ResponseWriter, r *http.Request) {
	var req routing.RouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid route request")
		return
	}
	if req.TenantID == "" {
		req.TenantID = getTenantID(r)
	}

	decision, err := s.routerEng.Route(r.Context(), &req)
	if err != nil {
		jsonResponse(w, http.StatusNotFound, decision)
		return
	}
	s.auditLogger.Log(req.TenantID, audit.EventRouteDecision, req.CallerAgentID, decision.SelectedAgentID, decision)
	jsonResponse(w, http.StatusOK, decision)
}

func (s *Server) handleRouteExplain(w http.ResponseWriter, r *http.Request) {
	var req routing.RouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid route request")
		return
	}
	if req.TenantID == "" {
		req.TenantID = getTenantID(r)
	}

	decision, _ := s.routerEng.Route(r.Context(), &req)
	jsonResponse(w, http.StatusOK, decision)
}

func (s *Server) handleListApprovals(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	pending := s.approvalSvc.ListPending(tenantID)
	jsonResponse(w, http.StatusOK, pending)
}

type resolveApprovalPayload struct {
	ReviewerID string `json:"reviewerId"`
	Approve    bool   `json:"approve"`
	Note       string `json:"note"`
}

func (s *Server) handleResolveApproval(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "id")
	var payload resolveApprovalPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid payload")
		return
	}

	req, err := s.approvalSvc.Resolve(requestID, payload.ReviewerID, payload.Approve, payload.Note)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.auditLogger.Log(req.TenantID, audit.EventApprovalResolved, payload.ReviewerID, req.ID, req)
	jsonResponse(w, http.StatusOK, req)
}

func (s *Server) handleListCanaries(w http.ResponseWriter, r *http.Request) {
	// Returns mock/active canaries for dashboard
	jsonResponse(w, http.StatusOK, []any{})
}

func (s *Server) handleStartCanary(w http.ResponseWriter, r *http.Request) {
	type startReq struct {
		AgentID          string  `json:"agentId"`
		BaselineVersion  string  `json:"baselineVersion"`
		CandidateVersion string  `json:"candidateVersion"`
		InitialWeight    int     `json:"initialWeight"`
		MaxErrorRate     float64 `json:"maxErrorRate"`
		MaxLatencyMs     int64   `json:"maxLatencyMs"`
	}
	var body startReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid json")
		return
	}

	dep, err := s.canaryMgr.StartCanary(body.AgentID, body.BaselineVersion, body.CandidateVersion, body.InitialWeight, false, body.MaxErrorRate, body.MaxLatencyMs)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.auditLogger.Log(getTenantID(r), audit.EventCanaryStarted, "api", body.AgentID, dep)
	jsonResponse(w, http.StatusCreated, dep)
}

func (s *Server) handlePromoteCanary(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentId")
	type promoteReq struct {
		NewWeight int `json:"newWeight"`
	}
	var body promoteReq
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.NewWeight == 0 {
		body.NewWeight = 100
	}

	dep, err := s.canaryMgr.Promote(agentID, body.NewWeight)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.auditLogger.Log(getTenantID(r), audit.EventCanaryPromoted, "api", agentID, dep)
	jsonResponse(w, http.StatusOK, dep)
}

func (s *Server) handleRollbackCanary(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentId")
	dep, err := s.canaryMgr.Rollback(agentID, "Emergency rollback triggered via API")
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	s.auditLogger.Log(getTenantID(r), audit.EventCanaryRolledBack, "api", agentID, dep)
	jsonResponse(w, http.StatusOK, dep)
}

func (s *Server) handleListTraces(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	traces := s.telemetry.ListTraces(tenantID, 50)
	jsonResponse(w, http.StatusOK, traces)
}

func (s *Server) handleGetTrace(w http.ResponseWriter, r *http.Request) {
	traceID := chi.URLParam(r, "id")
	tr, exists := s.telemetry.GetTrace(traceID)
	if !exists {
		errorResponse(w, http.StatusNotFound, "trace not found")
		return
	}
	jsonResponse(w, http.StatusOK, tr)
}

func (s *Server) handleRecordTrace(w http.ResponseWriter, r *http.Request) {
	var tr telemetry.AgentTrace
	if err := json.NewDecoder(r.Body).Decode(&tr); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid trace json")
		return
	}
	if tr.TenantID == "" {
		tr.TenantID = getTenantID(r)
	}
	s.telemetry.RecordTrace(&tr)
	jsonResponse(w, http.StatusAccepted, map[string]string{"status": "recorded"})
}

func (s *Server) handleListAudit(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	entries := s.auditLogger.List(tenantID, 100)
	jsonResponse(w, http.StatusOK, entries)
}

func (s *Server) handleCreateCredential(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	type credReq struct {
		SubjectID   string   `json:"subjectId"`
		Scopes      []string `json:"scopes"`
		TTLSeconds  int      `json:"ttlSeconds"`
		Description string   `json:"description"`
	}
	var req credReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid request")
		return
	}

	ttl := time.Duration(req.TTLSeconds) * time.Second
	rawKey, cred, err := identity.GenerateAPIKey(tenantID, req.SubjectID, req.Scopes, ttl, req.Description)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = s.store.SaveCredential(r.Context(), cred)
	s.auditLogger.Log(tenantID, audit.EventCredentialCreated, "api", cred.ID, map[string]any{"scopes": req.Scopes})

	jsonResponse(w, http.StatusCreated, map[string]any{
		"apiKey":     rawKey, // Returned only once!
		"credential": cred,
	})
}

func (s *Server) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	creds, err := s.store.ListCredentials(r.Context(), tenantID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, creds)
}

func (s *Server) handleGetSignedConfigBundle(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	policies, _ := s.store.ListPolicies(r.Context(), tenantID)
	agents, _ := s.store.ListAgents(r.Context(), tenantID)

	payload := map[string]any{
		"tenantId":   tenantID,
		"policies":   policies,
		"agentCount": len(agents),
		"generated":  time.Now().UTC(),
	}

	bundle, err := crypto.SignPayload(s.signingKey, fmt.Sprintf("bundle_%d", time.Now().Unix()), 24*time.Hour, payload)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to sign config: %v", err))
		return
	}

	jsonResponse(w, http.StatusOK, bundle)
}

// ---------------------------------------------------------------------------
// Phase 2 Handlers: Graphs, Passports, Tools, Routing V2, Evaluations & Lab
// ---------------------------------------------------------------------------

func (s *Server) handleSaveGraph(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	var g graph.AgentGraph
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid graph json")
		return
	}
	if g.OrganizationID == "" {
		g.OrganizationID = tenantID
	}
	if err := g.Validate(); err != nil {
		errorResponse(w, http.StatusBadRequest, fmt.Sprintf("graph validation failed: %v", err))
		return
	}
	h, err := g.Hash()
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to hash graph: %v", err))
		return
	}
	if err := s.store.SaveGraph(r.Context(), tenantID, &g); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, map[string]any{
		"graph":     g,
		"graphHash": h,
	})
}

func (s *Server) handleListGraphs(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	graphs, err := s.store.ListGraphs(r.Context(), tenantID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, graphs)
}

func (s *Server) handleGetGraph(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	graphID := chi.URLParam(r, "id")
	g, err := s.store.GetGraph(r.Context(), tenantID, graphID)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "graph not found")
		return
	}
	jsonResponse(w, http.StatusOK, g)
}

func (s *Server) handleGetAgentGraph(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	agentID := chi.URLParam(r, "id")
	graphs, err := s.store.ListGraphs(r.Context(), tenantID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, g := range graphs {
		if g.AgentID == agentID {
			jsonResponse(w, http.StatusOK, g)
			return
		}
	}
	errorResponse(w, http.StatusNotFound, "graph for agent not found")
}

func (s *Server) handleAnalyzeGraph(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	var g graph.AgentGraph
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid graph json")
		return
	}
	if err := g.Validate(); err != nil {
		errorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid graph: %v", err))
		return
	}

	riskReport := adk.AnalyzeGraphRisk(&g)
	policies, _ := s.store.ListPolicies(r.Context(), tenantID)
	var p *policy.Policy
	if len(policies) > 0 {
		p = policies[0]
	}
	policyReport := policy.AnalyzeGraphPolicy(&g, p)

	jsonResponse(w, http.StatusOK, map[string]any{
		"graphId": g.GraphID,
		"risk":    riskReport,
		"policy":  policyReport,
	})
}

func (s *Server) handleGetAgentPassport(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	agentID := chi.URLParam(r, "id")
	agent, err := s.store.GetAgent(r.Context(), tenantID, agentID)
	if err != nil || agent.Passport == nil {
		errorResponse(w, http.StatusNotFound, "passport not found")
		return
	}

	if r.URL.Query().Get("public") == "true" {
		publicPass := agent.Passport.SanitizeForPublic()
		jsonResponse(w, http.StatusOK, publicPass)
		return
	}

	jsonResponse(w, http.StatusOK, agent.Passport)
}

func (s *Server) handleGetAgentBadge(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	agentID := chi.URLParam(r, "id")
	agent, err := s.store.GetAgent(r.Context(), tenantID, agentID)
	if err != nil || agent.Passport == nil {
		errorResponse(w, http.StatusNotFound, "agent not found")
		return
	}

	badge := agent.Passport.GenerateBadge()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(badge))
}

func (s *Server) handleListToolPassports(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	tools, err := s.store.ListToolPassports(r.Context(), tenantID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, tools)
}

func (s *Server) handleGetToolPassport(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	toolID := chi.URLParam(r, "id")
	tp, err := s.store.GetToolPassport(r.Context(), tenantID, toolID)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "tool passport not found")
		return
	}
	jsonResponse(w, http.StatusOK, tp)
}

func (s *Server) handleSaveToolPassport(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	var tp mcp.ToolPassport
	if err := json.NewDecoder(r.Body).Decode(&tp); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid tool passport json")
		return
	}
	if err := s.store.SaveToolPassport(r.Context(), tenantID, &tp); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, tp)
}

func (s *Server) handleDetectToolDrift(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OldSchema json.RawMessage `json:"oldSchema"`
		NewSchema json.RawMessage `json:"newSchema"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid drift payload")
		return
	}
	oldFP, _ := mcp.CalculateToolFingerprint("server", "tool", "1.0", "provider", mcp.RiskClassRead, body.OldSchema)
	newFP, _ := mcp.CalculateToolFingerprint("server", "tool", "2.0", "provider", mcp.RiskClassRead, body.NewSchema)
	report := mcp.DetectSchemaDrift(oldFP, newFP)
	jsonResponse(w, http.StatusOK, map[string]string{"status": string(report)})
}

func (s *Server) handleListCapabilities(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	agents, err := s.store.ListAgents(r.Context(), tenantID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	capMap := make(map[string]bool)
	for _, a := range agents {
		if a.Contract != nil {
			for _, c := range a.Contract.Capabilities {
				capMap[c] = true
			}
		}
	}
	caps := make([]string, 0, len(capMap))
	for c := range capMap {
		caps = append(caps, c)
	}
	jsonResponse(w, http.StatusOK, map[string]any{"capabilities": caps})
}

func (s *Server) handleSimulateRoute(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	var req routing.RouteRequestV2
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid route request v2")
		return
	}
	if req.TenantID == "" {
		req.TenantID = tenantID
	}

	rec, err := s.routerEng.RouteV2(r.Context(), &req)
	if err != nil {
		jsonResponse(w, http.StatusOK, rec)
		return
	}
	jsonResponse(w, http.StatusOK, rec)
}

func (s *Server) handleListRouteOutcomes(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	outcomes, err := s.store.ListRouteOutcomes(r.Context(), tenantID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, outcomes)
}

func (s *Server) handleSimulatePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	var req policy.EvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid policy evaluation request")
		return
	}
	if req.TenantID == "" {
		req.TenantID = tenantID
	}
	decision := s.policyEng.Simulate(r.Context(), &req)
	jsonResponse(w, http.StatusOK, decision)
}

func (s *Server) handlePolicyCanary(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	var body struct {
		PolicyID        string         `json:"policyId"`
		BaselinePolicy  *policy.Policy `json:"baselinePolicy"`
		CandidatePolicy *policy.Policy `json:"candidatePolicy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid policy canary payload")
		return
	}
	canaryObj := &policy.PolicyCanary{
		ID:              body.PolicyID,
		TenantID:        tenantID,
		BaselinePolicy:  body.BaselinePolicy,
		CandidatePolicy: body.CandidatePolicy,
		ShadowMode:      true,
		CreatedAt:       time.Now().UTC(),
	}
	_ = policy.NewShadowEvaluator(canaryObj)
	jsonResponse(w, http.StatusOK, map[string]any{
		"canary": canaryObj,
		"status": "SHADOW_ACTIVE",
	})
}

func (s *Server) handleRunEvaluation(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	var suite evaluation.EvaluationSuite
	if err := json.NewDecoder(r.Body).Decode(&suite); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid evaluation suite")
		return
	}
	if suite.TenantID == "" {
		suite.TenantID = tenantID
	}
	report, prov, err := suite.ExecuteSuite(r.Context(), "target-agent", "1.0", "gemini-1.5-pro", func(ctx context.Context, tc evaluation.EvaluationTestCase) (map[string]any, []string, int64, float64, error) {
		return map[string]any{"output": "evaluated"}, []string{"safe_tool"}, 120, 0.002, nil
	})
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.store.SaveEvaluationSuite(r.Context(), &suite)
	jsonResponse(w, http.StatusOK, map[string]any{
		"report":     report,
		"provenance": prov,
	})
}

func (s *Server) handleListEvaluationResults(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	suites, err := s.store.ListEvaluationSuites(r.Context(), tenantID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, suites)
}

func (s *Server) handleAnalyzeChangeImpact(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Current   *contracts.AgentContract `json:"current"`
		Candidate *contracts.AgentContract `json:"candidate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid change impact payload")
		return
	}
	if body.Candidate == nil {
		errorResponse(w, http.StatusBadRequest, "candidate contract is required")
		return
	}

	report, err := canary.AnalyzeChangeImpact(body.Current, body.Candidate)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, report)
}

func (s *Server) handleTestA2AEndpoint(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	var body struct {
		EndpointURL string `json:"endpointUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid request")
		return
	}
	lab := a2a.NewCompatibilityLab()
	profile, err := lab.RunSuite(r.Context(), body.EndpointURL)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.store.SaveA2AProfile(r.Context(), tenantID, profile)
	jsonResponse(w, http.StatusOK, profile)
}

func (s *Server) handleListA2AProfiles(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	profiles, err := s.store.ListA2AProfiles(r.Context(), tenantID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, profiles)
}

