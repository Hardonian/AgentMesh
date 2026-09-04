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

func TestServerPhase3Endpoints(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	// 1. Post Route Outcome V3
	outcomePayload := map[string]any{
		"outcome_id":              "out-test-1",
		"task_id":                 "task-test-1",
		"capability_id":           "search",
		"selected_agent_id":       "agent-a",
		"selected_agent_version":  "1.0.0",
		"success":                 true,
		"latency_ms":              450,
		"cost":                    0.02,
		"route_algorithm_version": "BASELINE_ROUTER_V1",
	}
	body, _ := json.Marshal(outcomePayload)
	resp, err := http.Post(ts.URL+"/api/v1/routes/outcomes/v3", "application/json", bytes.NewReader(body))
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to post outcome v3: %v, status: %v", err, resp.StatusCode)
	}

	// 2. List Route Outcomes V3
	getResp, err := http.Get(ts.URL + "/api/v1/routes/outcomes/v3")
	if err != nil || getResp.StatusCode != http.StatusOK {
		t.Fatalf("failed to list outcomes v3: %v, status: %v", err, getResp.StatusCode)
	}

	// 3. Route Replay
	repResp, err := http.Post(ts.URL+"/api/v1/routes/replay", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil || repResp.StatusCode != http.StatusOK {
		t.Fatalf("failed to replay routes: %v, status: %v", err, repResp.StatusCode)
	}

	// 4. Save and List SLOs
	sloPayload := map[string]any{
		"id":                "slo-test-1",
		"agentId":           "agent-a",
		"capabilityId":      "search",
		"targetSuccessRate": 0.99,
		"maxP95LatencyMs":   2000,
	}
	sloBody, _ := json.Marshal(sloPayload)
	sloPostResp, err := http.Post(ts.URL+"/api/v1/slos", "application/json", bytes.NewReader(sloBody))
	if err != nil || sloPostResp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to post slo: %v, status: %v", err, sloPostResp.StatusCode)
	}

	sloGetResp, err := http.Get(ts.URL + "/api/v1/slos")
	if err != nil || sloGetResp.StatusCode != http.StatusOK {
		t.Fatalf("failed to list slos: %v, status: %v", err, sloGetResp.StatusCode)
	}

	// 5. Proxy Fleet Heartbeat and List
	hbPayload := map[string]any{
		"instanceId":   "proxy-gke-test",
		"cluster":      "gke-cluster",
		"region":       "us-central1",
		"proxyVersion": "1.0.0",
		"health":       "HEALTHY",
	}
	hbBody, _ := json.Marshal(hbPayload)
	hbResp, err := http.Post(ts.URL+"/api/v1/proxy-fleet/heartbeat", "application/json", bytes.NewReader(hbBody))
	if err != nil || hbResp.StatusCode != http.StatusOK {
		t.Fatalf("failed to send heartbeat: %v, status: %v", err, hbResp.StatusCode)
	}

	fleetResp, err := http.Get(ts.URL + "/api/v1/proxy-fleet")
	if err != nil || fleetResp.StatusCode != http.StatusOK {
		t.Fatalf("failed to get proxy fleet: %v, status: %v", err, fleetResp.StatusCode)
	}

	// 6. BigQuery Export Batch
	bqResp, err := http.Post(ts.URL+"/api/v1/analytics/export/bigquery", "application/json", bytes.NewReader([]byte(`{"gcpProject":"test-project"}`)))
	if err != nil || bqResp.StatusCode != http.StatusOK {
		t.Fatalf("failed to export bigquery batch: %v, status: %v", err, bqResp.StatusCode)
	}
}
