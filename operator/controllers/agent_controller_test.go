package controllers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentmesh/agentmesh/operator/api/v1alpha1"
	"github.com/agentmesh/agentmesh/operator/controllers"
	"github.com/agentmesh/agentmesh/pkg/contracts"
	"github.com/agentmesh/agentmesh/pkg/sdk"
)

func TestAgentReconciler_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(sdk.AgentRegistrationResponse{
			AgentID:      "agent-k8s-1",
			ContractHash: "sha256:abcd1234",
			Status:       "ACTIVE",
		})
	}))
	defer server.Close()

	client := sdk.NewClient(server.URL, "test-key")
	reconciler := controllers.NewAgentReconciler(client)

	agentCR := &v1alpha1.AgentMeshAgent{
		APIVersion: "agentmesh.dev/v1alpha1",
		Kind:       "AgentMeshAgent",
		Spec: v1alpha1.AgentMeshAgentSpec{
			Name: "finance-k8s",
			Contract: contracts.AgentContract{
				Metadata: contracts.Metadata{Name: "finance-k8s"},
			},
			EndpointURL: "http://finance-k8s.svc.cluster.local:8080",
			Replicas:    3,
		},
	}

	status, err := reconciler.ReconcileAgent(context.Background(), agentCR)
	if err != nil {
		t.Fatalf("expected successful reconcile, got err: %v", err)
	}

	if status.Phase != "Active" {
		t.Errorf("expected status Active, got %s", status.Phase)
	}
	if !status.RegisteredInCP {
		t.Errorf("expected RegisteredInCP to be true")
	}
	if status.ContractHash != "sha256:abcd1234" {
		t.Errorf("unexpected contract hash: %s", status.ContractHash)
	}
	if status.LastHeartbeatAt == "" {
		t.Errorf("expected LastHeartbeatAt timestamp")
	}
}

func TestAgentReconciler_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid contract payload", http.StatusBadRequest)
	}))
	defer server.Close()

	client := sdk.NewClient(server.URL, "test-key")
	reconciler := controllers.NewAgentReconciler(client)

	agentCR := &v1alpha1.AgentMeshAgent{
		Spec: v1alpha1.AgentMeshAgentSpec{
			Name: "bad-agent",
		},
	}

	status, err := reconciler.ReconcileAgent(context.Background(), agentCR)
	if err == nil {
		t.Fatalf("expected error on registration failure")
	}

	if status.Phase != "Error" {
		t.Errorf("expected status Error, got %s", status.Phase)
	}
	if status.RegisteredInCP {
		t.Errorf("expected RegisteredInCP to be false")
	}
}
