package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentmesh/agentmesh/internal/approval"
	"github.com/agentmesh/agentmesh/internal/audit"
	"github.com/agentmesh/agentmesh/internal/canary"
	"github.com/agentmesh/agentmesh/internal/crypto"
	"github.com/agentmesh/agentmesh/internal/database"
	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/internal/server"
	"github.com/agentmesh/agentmesh/internal/telemetry"
	"github.com/agentmesh/agentmesh/pkg/spec"
)

func setupControlTestServer() *server.Server {
	memStore := database.NewMemoryStore()
	polEngine := policy.NewEngine([]*policy.Policy{})
	routerEng := routing.NewRouter(polEngine)
	collector := telemetry.NewCollector()
	canaryMgr := canary.NewManager()
	approvalSvc := approval.NewService()
	auditLogger := audit.NewLogger()
	kp, _ := crypto.GenerateKeyPair("test_key")

	return server.NewServer(
		memStore,
		polEngine,
		routerEng,
		collector,
		canaryMgr,
		approvalSvc,
		auditLogger,
		kp,
	)
}

func TestControlPlaneEndpoints(t *testing.T) {
	srv := setupControlTestServer()

	// 1. Create optimization action
	actPayload := spec.AgentOptimizationAction{
		ActionID:       "act-test-01",
		CapabilityID:   "summarize",
		TargetType:     "ROUTE",
		TargetID:       "route-sum",
		ActionType:     spec.ActionChangeRouteWeight,
		CurrentState:   map[string]any{"weight": 100},
		ProposedState:  map[string]any{"weight": 80},
		Reason:         "minor cost optimization",
		RiskClass:      spec.RiskLow,
		BlastRadius: spec.BlastRadius{
			TrafficPercent: 20,
		},
	}
	body, _ := json.Marshal(actPayload)
	req := httptest.NewRequest("POST", "/api/v1/control/actions", bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", "tenant-test")
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created on action submission, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. List actions
	reqList := httptest.NewRequest("GET", "/api/v1/control/actions", nil)
	reqList.Header.Set("X-Tenant-ID", "tenant-test")
	recList := httptest.NewRecorder()
	srv.Router().ServeHTTP(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on listing actions, got %d", recList.Code)
	}
	var actions []*spec.AgentOptimizationAction
	_ = json.NewDecoder(recList.Body).Decode(&actions)
	if len(actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(actions))
	}

	// 3. Dry-run action
	reqDry := httptest.NewRequest("POST", "/api/v1/control/actions/act-test-01/dry-run", nil)
	reqDry.Header.Set("X-Tenant-ID", "tenant-test")
	recDry := httptest.NewRecorder()
	srv.Router().ServeHTTP(recDry, reqDry)

	if recDry.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on dry-run, got %d", recDry.Code)
	}

	// 4. Approve action
	appPayload := map[string]string{"approver": "lead-sre"}
	appBody, _ := json.Marshal(appPayload)
	reqApp := httptest.NewRequest("POST", "/api/v1/control/actions/act-test-01/approve", bytes.NewReader(appBody))
	reqApp.Header.Set("X-Tenant-ID", "tenant-test")
	recApp := httptest.NewRecorder()
	srv.Router().ServeHTTP(recApp, reqApp)

	if recApp.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on approve, got %d", recApp.Code)
	}

	// 5. Execute action
	reqExec := httptest.NewRequest("POST", "/api/v1/control/actions/act-test-01/execute", nil)
	reqExec.Header.Set("X-Tenant-ID", "tenant-test")
	recExec := httptest.NewRecorder()
	srv.Router().ServeHTTP(recExec, reqExec)

	if recExec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on execute, got %d: %s", recExec.Code, recExec.Body.String())
	}

	// 6. Freeze automation (Kill Switch)
	freezePayload := map[string]string{
		"scope":    "GLOBAL",
		"scopeId":  "all",
		"reason":   "drill",
		"frozenBy": "secops",
	}
	freezeBody, _ := json.Marshal(freezePayload)
	reqFreeze := httptest.NewRequest("POST", "/api/v1/control/freeze", bytes.NewReader(freezeBody))
	recFreeze := httptest.NewRecorder()
	srv.Router().ServeHTTP(recFreeze, reqFreeze)

	if recFreeze.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on freeze, got %d", recFreeze.Code)
	}

	// Now executing an action while frozen must return 403 Forbidden
	reqExecFrozen := httptest.NewRequest("POST", "/api/v1/control/actions/act-test-01/execute", nil)
	reqExecFrozen.Header.Set("X-Tenant-ID", "tenant-test")
	recExecFrozen := httptest.NewRecorder()
	srv.Router().ServeHTTP(recExecFrozen, reqExecFrozen)

	if recExecFrozen.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when frozen, got %d", recExecFrozen.Code)
	}

	// 7. Unfreeze
	unfreezePayload := map[string]string{"scope": "GLOBAL", "scopeId": "all"}
	unfreezeBody, _ := json.Marshal(unfreezePayload)
	reqUnfreeze := httptest.NewRequest("POST", "/api/v1/control/unfreeze", bytes.NewReader(unfreezeBody))
	recUnfreeze := httptest.NewRecorder()
	srv.Router().ServeHTTP(recUnfreeze, reqUnfreeze)

	if recUnfreeze.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on unfreeze, got %d", recUnfreeze.Code)
	}
}
