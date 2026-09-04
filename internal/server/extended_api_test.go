package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/approval"
	"github.com/agentmesh/agentmesh/internal/audit"
	"github.com/agentmesh/agentmesh/internal/canary"
	"github.com/agentmesh/agentmesh/internal/crypto"
	"github.com/agentmesh/agentmesh/internal/database"
	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/internal/server"
	"github.com/agentmesh/agentmesh/internal/telemetry"
	"github.com/agentmesh/agentmesh/pkg/contracts"
	"github.com/agentmesh/agentmesh/pkg/graph"
	"github.com/agentmesh/agentmesh/pkg/passport"
)

func setupExtendedServer() (*server.Server, database.Store, *routing.Router, *policy.Engine) {
	memStore := database.NewMemoryStore()
	polEngine := policy.NewEngine([]*policy.Policy{})
	routerEng := routing.NewRouter(polEngine)
	collector := telemetry.NewCollector()
	canaryMgr := canary.NewManager()
	approvalSvc := approval.NewService()
	auditLogger := audit.NewLogger()
	kp, _ := crypto.GenerateKeyPair("test_key")

	srv := server.NewServer(
		memStore,
		polEngine,
		routerEng,
		collector,
		canaryMgr,
		approvalSvc,
		auditLogger,
		kp,
	)
	return srv, memStore, routerEng, polEngine
}

func doReq(srv *server.Server, method, path string, body any, tenantID string) *httptest.ResponseRecorder {
	var reqBody *bytes.Reader
	if body != nil {
		switch v := body.(type) {
		case []byte:
			reqBody = bytes.NewReader(v)
		case string:
			reqBody = bytes.NewReader([]byte(v))
		default:
			b, _ := json.Marshal(v)
			reqBody = bytes.NewReader(b)
		}
	} else {
		reqBody = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	} else {
		req.Header.Set("X-Tenant-ID", "tenant-alpha")
	}
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)
	return rec
}

func TestExtendedAPI_ReadinessAndMetrics(t *testing.T) {
	srv, _, _, _ := setupExtendedServer()

	// 1. Readyz
	rec := doReq(srv, "GET", "/readyz", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on /readyz, got %d", rec.Code)
	}

	// 2. Metrics
	rec = doReq(srv, "GET", "/metrics", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on /metrics, got %d", rec.Code)
	}
}

func TestExtendedAPI_AgentsLifecycle(t *testing.T) {
	srv, _, _, _ := setupExtendedServer()

	contract := contracts.AgentContract{
		APIVersion: contracts.ExpectedAPIVersion,
		Kind:       contracts.ExpectedKind,
		Metadata: contracts.Metadata{
			Name:    "agent-lifecycle-test",
			Version: "1.0.0",
		},
		Identity: contracts.IdentityConfig{
			Protocols: []string{"a2a", "mcp"},
		},
		Capabilities: []string{"finance-search", "analytics"},
		Tools: contracts.ToolsConfig{
			Allow: []string{"bigquery.read", "analytics.query"},
		},
	}

	// Create Agent
	rec := doReq(srv, "POST", "/api/v1/agents", contract, "tenant-alpha")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on register, got %d: %s", rec.Code, rec.Body.String())
	}

	// Get Agent Existing
	rec = doReq(srv, "GET", "/api/v1/agents/agent-lifecycle-test", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on get agent, got %d", rec.Code)
	}

	// Get Agent Not Found
	rec = doReq(srv, "GET", "/api/v1/agents/non-existent-agent", nil, "tenant-alpha")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on non existent agent, got %d", rec.Code)
	}

	// List Agents
	rec = doReq(srv, "GET", "/api/v1/agents", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list agents, got %d", rec.Code)
	}

	// Delete Agent
	rec = doReq(srv, "DELETE", "/api/v1/agents/agent-lifecycle-test", nil, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("expected 200 or 204 on delete agent, got %d", rec.Code)
	}
}

func TestExtendedAPI_Policies(t *testing.T) {
	srv, _, _, _ := setupExtendedServer()

	// 1. Save Policy
	pol := policy.Policy{
		ID:          "pol-001",
		TenantID:    "tenant-alpha",
		Name:        "Test Security Policy",
		Description: "Deny destructive updates",
		Rules: []policy.Rule{
			{
				Name:    "rule-deny-del",
				Actions: []string{"bigquery.delete"},
				Effect:  policy.EffectDeny,
			},
			{
				Name:    "rule-hitl-write",
				Actions: []string{"bigquery.write"},
				Effect:  policy.EffectRequireApproval,
			},
			{
				Name:    "rule-allow-read",
				Actions: []string{"bigquery.read"},
				Effect:  policy.EffectAllow,
			},
		},
	}

	rec := doReq(srv, "POST", "/api/v1/policies", pol, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("expected 200/201 on save policy, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. List Policies
	rec = doReq(srv, "GET", "/api/v1/policies", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list policies, got %d", rec.Code)
	}

	// 3. Evaluate Policy - Allow
	evalPayload := map[string]any{
		"agent_id": "test-agent",
		"tool":     "bigquery.read",
		"action":   "bigquery.read",
		"params":   map[string]any{"dataset": "public"},
	}
	rec = doReq(srv, "POST", "/api/v1/policy/evaluate", evalPayload, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on policy evaluate, got %d", rec.Code)
	}

	// 4. Simulate Policy
	simPayload := map[string]any{
		"policy": pol,
		"actions": []string{
			"bigquery.read",
			"bigquery.write",
			"bigquery.delete",
		},
	}
	rec = doReq(srv, "POST", "/api/v1/policy/simulate", simPayload, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on policy simulate, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. Policy Canary
	canaryPayload := map[string]any{
		"policy_id": "pol-001",
		"weight":    20,
	}
	rec = doReq(srv, "POST", "/api/v1/policy/canary", canaryPayload, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("expected 200/201 on policy canary, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestExtendedAPI_RoutingAndCapabilities(t *testing.T) {
	srv, store, routerEng, polEngine := setupExtendedServer()

	// Add policy allowing deep-research capability
	pol := &policy.Policy{
		ID:       "pol-routing-allow",
		TenantID: "tenant-alpha",
		Rules: []policy.Rule{
			{
				Name:   "allow-all",
				Effect: policy.EffectAllow,
			},
		},
	}
	_ = store.SavePolicy(httptest.NewRequest("GET", "/", nil).Context(), pol)
	allPols, _ := store.ListPolicies(httptest.NewRequest("GET", "/", nil).Context(), "tenant-alpha")
	polEngine.SetPolicies(allPols)

	// Register routing candidate in engine
	c := &contracts.AgentContract{
		APIVersion: contracts.ExpectedAPIVersion,
		Kind:       contracts.ExpectedKind,
		Metadata: contracts.Metadata{
			Name:    "routing-agent-1",
			Version: "1.0.0",
		},
		Capabilities: []string{"deep-research", "summarization"},
	}
	routerEng.RegisterCandidate(&routing.AgentRouteCandidate{
		AgentID:      "routing-agent-1",
		EndpointURL:  "http://localhost:8080",
		Status:       "HEALTHY",
		Contract:     c,
		SuccessRate:  0.99,
		P95LatencyMs: 150,
		AverageCost:  0.01,
	})

	// 1. Route Request
	routeReq := routing.RouteRequest{
		CallerAgentID:      "caller-agent",
		RequiredCapability: "deep-research",
		Strategy:           routing.StrategyBalanced,
	}
	rec := doReq(srv, "POST", "/api/v1/routing/route", routeReq, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on routing/route, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Route Explain
	rec = doReq(srv, "POST", "/api/v1/routing/explain", routeReq, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on routing/explain, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Route Simulate
	simReq := map[string]any{
		"capability": "deep-research",
		"strategy":   "LATENCY_OPTIMAL",
	}
	rec = doReq(srv, "POST", "/api/v1/routing/simulate", simReq, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on routing/simulate, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. List Capabilities
	rec = doReq(srv, "GET", "/api/v1/capabilities", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on capabilities, got %d", rec.Code)
	}
}

func TestExtendedAPI_ApprovalsAndCanaries(t *testing.T) {
	srv, _, _, _ := setupExtendedServer()

	// 1. List Approvals
	rec := doReq(srv, "GET", "/api/v1/approvals", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on approvals list, got %d", rec.Code)
	}

	// 2. Resolve Approval (not found test)
	resolvePayload := map[string]any{
		"decision": "APPROVED",
		"reviewer": "alice@security.corp",
	}
	rec = doReq(srv, "POST", "/api/v1/approvals/non-existent-app-id/resolve", resolvePayload, "tenant-alpha")
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 or 404 on non-existent approval resolve, got %d", rec.Code)
	}

	// 3. List Canaries
	rec = doReq(srv, "GET", "/api/v1/canaries", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on canaries list, got %d", rec.Code)
	}

	// 4. Start Canary
	canaryReq := map[string]any{
		"agentId":          "agent-v2",
		"baselineVersion":  "1.0.0",
		"candidateVersion": "2.0.0-rc1",
		"initialWeight":    15,
		"maxErrorRate":     0.05,
		"maxLatencyMs":     500,
	}
	rec = doReq(srv, "POST", "/api/v1/canaries", canaryReq, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("expected 200/201 on start canary, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. Promote Canary
	rec = doReq(srv, "POST", "/api/v1/canaries/agent-v2/promote", map[string]int{"newWeight": 50}, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on promote canary, got %d: %s", rec.Code, rec.Body.String())
	}

	// 6. Rollback Canary
	rec = doReq(srv, "POST", "/api/v1/canaries/agent-v2/rollback", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on rollback canary, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestExtendedAPI_TelemetryTracesAndAudit(t *testing.T) {
	srv, _, _, _ := setupExtendedServer()

	// 1. Record Trace
	tracePayload := map[string]any{
		"trace_id":   "trace-alpha-001",
		"agent_id":   "agent-1",
		"duration":   125000000,
		"tokens":     500,
		"cost":       0.0025,
		"success":    true,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}
	rec := doReq(srv, "POST", "/api/v1/telemetry/traces", tracePayload, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated && rec.Code != http.StatusAccepted {
		t.Fatalf("expected 200/201/202 on record trace, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. List Traces
	rec = doReq(srv, "GET", "/api/v1/traces", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list traces, got %d", rec.Code)
	}

	// 3. Get Trace by ID
	rec = doReq(srv, "GET", "/api/v1/traces/trace-alpha-001", nil, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Fatalf("expected 200 or 404 on get trace, got %d", rec.Code)
	}

	// 4. List Audit
	rec = doReq(srv, "GET", "/api/v1/audit", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list audit, got %d", rec.Code)
	}
}

func TestExtendedAPI_CredentialsAndGraphs(t *testing.T) {
	srv, _, _, _ := setupExtendedServer()

	// 1. Create Credential
	credReq := map[string]any{
		"name":        "Test Operator Token",
		"role":        "ADMIN",
		"permissions": []string{"*"},
	}
	rec := doReq(srv, "POST", "/api/v1/credentials", credReq, "tenant-alpha")
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("expected 201 on create credential, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. List Credentials
	rec = doReq(srv, "GET", "/api/v1/credentials", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list credentials, got %d", rec.Code)
	}

	// 3. Save Graph
	agGraph := graph.AgentGraph{
		GraphID:    "graph-agent-01",
		AgentID:    "graph-agent-01",
		Version:    "1.0.0",
		Entrypoint: "node-1",
		Nodes: []graph.Node{
			{ID: "node-1", Name: "RootAgent", Type: "AGENT"},
			{ID: "node-2", Name: "DBTool", Type: "TOOL"},
		},
		Edges: []graph.Edge{
			{FromID: "node-1", ToID: "node-2"},
		},
	}
	rec = doReq(srv, "POST", "/api/v1/graphs", agGraph, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("expected 200/201 on save graph, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. List Graphs
	rec = doReq(srv, "GET", "/api/v1/graphs", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list graphs, got %d", rec.Code)
	}

	// 5. Get Graph by ID
	rec = doReq(srv, "GET", "/api/v1/graphs/graph-agent-01", nil, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Fatalf("expected 200 or 404 on get graph, got %d", rec.Code)
	}

	// 6. Analyze Graph
	rec = doReq(srv, "POST", "/api/v1/graphs/analyze", agGraph, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on analyze graph, got %d: %s", rec.Code, rec.Body.String())
	}

	// 7. Get Agent Graph
	rec = doReq(srv, "GET", "/api/v1/agents/graph-agent-01/graph", nil, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Fatalf("expected 200 or 404 on get agent graph, got %d", rec.Code)
	}
}

func TestExtendedAPI_PassportsAndTools(t *testing.T) {
	srv, store, _, _ := setupExtendedServer()

	// Seed passport in store via AgentRecord
	p := passport.AgentPassport{
		APIVersion: "agentmesh.dev/v1alpha1",
		Kind:       "AgentPassport",
		Identity: passport.PassportIdentity{
			AgentID:      "passport-agent-01",
			Organization: "tenant-alpha",
		},
		Reliability: passport.ReliabilityMetrics{
			TaskSuccessRate: 0.995,
		},
	}
	agentRec := &database.AgentRecord{
		ID:       "passport-agent-01",
		TenantID: "tenant-alpha",
		Name:     "passport-agent-01",
		Passport: &p,
	}
	_ = store.SaveAgent(httptest.NewRequest("GET", "/", nil).Context(), agentRec)

	// 1. Get Agent Passport
	rec := doReq(srv, "GET", "/api/v1/agents/passport-agent-01/passport", nil, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Fatalf("expected 200 or 404 on passport, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Get Agent Badge
	rec = doReq(srv, "GET", "/api/v1/agents/passport-agent-01/badge", nil, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Fatalf("expected 200 or 404 on badge, got %d", rec.Code)
	}

	// 3. Save Tool Passport
	tp := map[string]any{
		"tool_id":   "tool-bigquery-v1",
		"name":      "BigQuery Query Executor",
		"risk":      "READ",
		"version":   "1.0.0",
		"schema":    `{"type":"object"}`,
		"tenant_id": "tenant-alpha",
	}
	rec = doReq(srv, "POST", "/api/v1/tools/passports", tp, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("expected 200/201 on save tool passport, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. List Tool Passports
	rec = doReq(srv, "GET", "/api/v1/tools/passports", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list tool passports, got %d", rec.Code)
	}

	// 5. Get Tool Passport
	rec = doReq(srv, "GET", "/api/v1/tools/passports/tool-bigquery-v1", nil, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Fatalf("expected 200 or 404 on get tool passport, got %d", rec.Code)
	}

	// 6. Detect Tool Drift
	driftReq := map[string]any{
		"tool_id":    "tool-bigquery-v1",
		"new_schema": `{"type":"object","properties":{"new_param":{"type":"string"}}}`,
	}
	rec = doReq(srv, "POST", "/api/v1/tools/drift", driftReq, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on detect tool drift, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestExtendedAPI_ChangeImpactAndEvaluations(t *testing.T) {
	srv, _, _, _ := setupExtendedServer()

	// 1. Analyze Change Impact
	candidate := &contracts.AgentContract{
		APIVersion: contracts.ExpectedAPIVersion,
		Kind:       contracts.ExpectedKind,
		Metadata: contracts.Metadata{
			Name:    "test-impact-agent",
			Version: "1.1.0",
		},
		Capabilities: []string{"search"},
	}
	impactReq := map[string]any{
		"candidate": candidate,
	}
	rec := doReq(srv, "POST", "/api/v1/canary/impact", impactReq, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on change impact, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Run Evaluation
	evalReq := map[string]any{
		"agent_id":    "test-eval-agent",
		"benchmark":   "safety-standard-v1",
		"sample_size": 10,
	}
	rec = doReq(srv, "POST", "/api/v1/evaluations/run", evalReq, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusAccepted {
		t.Fatalf("expected 200/202 on run evaluation, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. List Evaluation Results
	rec = doReq(srv, "GET", "/api/v1/evaluations/results", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on evaluation results, got %d", rec.Code)
	}

	// 4. A2A Registry
	rec = doReq(srv, "GET", "/api/v1/a2a/registry", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on a2a registry, got %d", rec.Code)
	}
}

func TestExtendedAPI_FreezeAndUnfreeze(t *testing.T) {
	srv, _, _, _ := setupExtendedServer()

	// 1. Freeze Automation
	freezeReq := map[string]any{
		"scope":    "PROJECT",
		"scopeId":  "proj-finance",
		"reason":   "High anomaly detected in model outputs",
		"frozenBy": "security-incident-lead",
	}
	rec := doReq(srv, "POST", "/api/v1/control/freeze", freezeReq, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on control freeze, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Unfreeze Automation
	unfreezeReq := map[string]any{
		"scope":   "PROJECT",
		"scopeId": "proj-finance",
	}
	rec = doReq(srv, "POST", "/api/v1/control/unfreeze", unfreezeReq, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on control unfreeze, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestExtendedAPI_AdvancedControlAndA2A(t *testing.T) {
	srv, _, _, _ := setupExtendedServer()

	// 1. List Route Outcomes
	rec := doReq(srv, "GET", "/api/v1/routing/outcomes", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on routing/outcomes, got %d", rec.Code)
	}

	// 2. List A2A Profiles
	rec = doReq(srv, "GET", "/api/v1/a2a/profiles", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on a2a/profiles, got %d", rec.Code)
	}

	// 3. Routing Specs: List and Save
	specPayload := map[string]any{
		"specId":       "spec-001",
		"capabilityId": "deep-research",
		"strategy":     "BALANCED",
	}
	rec = doReq(srv, "POST", "/api/v1/control/specs/routing", specPayload, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("expected 200/201 on save routing spec, got %d", rec.Code)
	}

	rec = doReq(srv, "GET", "/api/v1/control/specs/routing", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list routing specs, got %d", rec.Code)
	}

	// 4. Production Outcomes
	rec = doReq(srv, "GET", "/api/v1/control/outcomes", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on control/outcomes, got %d", rec.Code)
	}

	// 5. Routers Management
	rec = doReq(srv, "GET", "/api/v1/routers", nil, "tenant-alpha")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on get routers, got %d", rec.Code)
	}

	shadowReq := map[string]any{
		"modelId": "candidate-ml-v1",
	}
	rec = doReq(srv, "POST", "/api/v1/routers/shadow", shadowReq, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Fatalf("expected 200 or 404 on set router shadow, got %d", rec.Code)
	}

	promoteReq := map[string]any{
		"modelId": "candidate-ml-v1",
	}
	rec = doReq(srv, "POST", "/api/v1/routers/promote", promoteReq, "tenant-alpha")
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Fatalf("expected 200 or 404 on promote router, got %d", rec.Code)
	}
}

