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
	"github.com/agentmesh/agentmesh/pkg/contracts"
)

func setupTestServer() *httptest.Server {
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

	return httptest.NewServer(srv.Router())
}

func TestServerEndpoints(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	// 1. Health check
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	// 2. Register agent
	contract := contracts.AgentContract{
		APIVersion: contracts.ExpectedAPIVersion,
		Kind:       contracts.ExpectedKind,
		Metadata: contracts.Metadata{
			Name: "test-procurement-agent",
		},
		Identity: contracts.IdentityConfig{
			Protocols: []string{"a2a", "mcp"},
		},
		Capabilities: []string{"procurement"},
		Tools: contracts.ToolsConfig{
			Allow: []string{"bigquery.read"},
		},
	}
	body, _ := json.Marshal(contract)

	regResp, err := http.Post(ts.URL+"/api/v1/agents", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("agent registration failed: %v", err)
	}
	if regResp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201 Created, got %d", regResp.StatusCode)
	}

	// 3. List agents
	listResp, err := http.Get(ts.URL + "/api/v1/agents")
	if err != nil {
		t.Fatalf("list agents failed: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", listResp.StatusCode)
	}

	// 4. Signed bundle
	bundleResp, err := http.Get(ts.URL + "/api/v1/config/bundle")
	if err != nil {
		t.Fatalf("get bundle failed: %v", err)
	}
	if bundleResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for signed bundle, got %d", bundleResp.StatusCode)
	}
}
