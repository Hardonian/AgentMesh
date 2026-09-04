package sdk_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/pkg/contracts"
	"github.com/agentmesh/agentmesh/pkg/passport"
	"github.com/agentmesh/agentmesh/pkg/sdk"
)

func TestClient_OptionsAndHeaders(t *testing.T) {
	customHTTP := &http.Client{Timeout: 5 * time.Second}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-key" {
			t.Errorf("expected Bearer secret-key, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Tenant-ID") != "tenant-123" {
			t.Errorf("expected X-Tenant-ID tenant-123, got %s", r.Header.Get("X-Tenant-ID"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json Content-Type, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(sdk.AgentRegistrationResponse{
			AgentID:      "agent-test",
			ContractHash: "hash123",
			Status:       "ACTIVE",
		})
	}))
	defer server.Close()

	testClient := sdk.NewClient(server.URL, "secret-key",
		sdk.WithTenantID("tenant-123"),
		sdk.WithHTTPClient(customHTTP),
	)
	contract := &contracts.AgentContract{
		Metadata: contracts.Metadata{Name: "agent-test"},
	}

	resp, err := testClient.RegisterAgent(context.Background(), contract)
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}
	if resp.AgentID != "agent-test" || resp.Status != "ACTIVE" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestClient_RegisterAgent_Errors(t *testing.T) {
	client := sdk.NewClient("http://localhost:8080", "key")

	// Nil contract
	_, err := client.RegisterAgent(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil contract")
	}

	// Server error 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	errClient := sdk.NewClient(server.URL, "key")
	contract := &contracts.AgentContract{Metadata: contracts.Metadata{Name: "agent-test"}}
	_, err = errClient.RegisterAgent(context.Background(), contract)
	if err == nil {
		t.Error("expected error on 500 response")
	}
}

func TestClient_InspectAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/agents/agent-abc" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(sdk.AgentDetails{
				AgentID: "agent-abc",
				Status:  "ACTIVE",
				Passport: &passport.AgentPassport{
					Identity: passport.PassportIdentity{AgentID: "agent-abc"},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := sdk.NewClient(server.URL, "key")

	details, err := client.InspectAgent(context.Background(), "agent-abc")
	if err != nil {
		t.Fatalf("InspectAgent failed: %v", err)
	}
	if details.AgentID != "agent-abc" || details.Status != "ACTIVE" {
		t.Errorf("unexpected details: %+v", details)
	}

	// 404 test
	_, err = client.InspectAgent(context.Background(), "non-existent")
	if err == nil {
		t.Error("expected error for 404 agent")
	}
}

func TestClient_EvaluatePolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/policy/evaluate" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(sdk.PolicyEvaluationResponse{
				Effect:   "ALLOW",
				RuleName: "rule-allow-read",
				Reason:   "Allowed by test policy",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := sdk.NewClient(server.URL, "key")
	resp, err := client.EvaluatePolicy(context.Background(), &sdk.PolicyEvaluationRequest{
		SubjectAgentID: "agent-1",
		Tool:           "bigquery.read",
		Action:         "execute",
	})
	if err != nil {
		t.Fatalf("EvaluatePolicy failed: %v", err)
	}
	if resp.Effect != "ALLOW" || resp.RuleName != "rule-allow-read" {
		t.Errorf("unexpected eval response: %+v", resp)
	}
}

func TestClient_Route(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/routing/route" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(sdk.RoutingResponse{
				SelectedAgentID: "agent-best",
				EndpointURL:     "http://agent-best.internal",
				Strategy:        "LOWEST_LATENCY",
				Reasoning:       "Selected based on lowest P95 latency",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := sdk.NewClient(server.URL, "key")
	resp, err := client.Route(context.Background(), &sdk.RoutingRequest{
		RequiredCapability: "financial-research",
		Strategy:           "LOWEST_LATENCY",
		CallerAgentID:      "caller-1",
	})
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}
	if resp.SelectedAgentID != "agent-best" || resp.Strategy != "LOWEST_LATENCY" {
		t.Errorf("unexpected routing response: %+v", resp)
	}
}
