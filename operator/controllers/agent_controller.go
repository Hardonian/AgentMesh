package controllers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/agentmesh/agentmesh/operator/api/v1alpha1"
	"github.com/agentmesh/agentmesh/pkg/sdk"
)

// AgentReconciler synchronizes Kubernetes AgentMeshAgent resources with the AgentMesh control plane.
type AgentReconciler struct {
	client *sdk.Client
	logger *slog.Logger
}

func NewAgentReconciler(client *sdk.Client) *AgentReconciler {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	return &AgentReconciler{
		client: client,
		logger: logger,
	}
}

// ReconcileAgent handles addition/update of an AgentMeshAgent CR.
func (r *AgentReconciler) ReconcileAgent(ctx context.Context, agentCR *v1alpha1.AgentMeshAgent) (*v1alpha1.AgentMeshAgentStatus, error) {
	r.logger.Info("Reconciling AgentMeshAgent", "name", agentCR.Spec.Name)

	resp, err := r.client.RegisterAgent(ctx, &agentCR.Spec.Contract)
	if err != nil {
		r.logger.Error("Failed to register agent with control plane", "error", err)
		return &v1alpha1.AgentMeshAgentStatus{
			Phase:           "Error",
			RegisteredInCP:  false,
			LastHeartbeatAt: time.Now().UTC().Format(time.RFC3339),
		}, err
	}

	r.logger.Info("Successfully synchronized agent with AgentMesh control plane",
		"agentId", resp.AgentID,
		"contractHash", resp.ContractHash,
	)

	return &v1alpha1.AgentMeshAgentStatus{
		Phase:           "Active",
		ContractHash:    resp.ContractHash,
		RegisteredInCP:  true,
		LastHeartbeatAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}
