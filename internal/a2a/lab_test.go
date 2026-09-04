package a2a_test

import (
	"context"
	"testing"

	"github.com/agentmesh/agentmesh/internal/a2a"
	"github.com/agentmesh/agentmesh/pkg/protocol"
)

func TestA2ACompatibilityLab(t *testing.T) {
	card := &protocol.AgentCard{
		Name:         "test-agent",
		Version:      "1.0.0",
		Description:  "Automated test target",
		Capabilities: []protocol.AgentCapability{
			{Name: "default", Description: "Default capability"},
			{Name: "analysis", Description: "Analysis capability"},
		},
		Protocols:    []string{"a2a"},
	}

	server := a2a.NewServer(card, nil, func(ctx context.Context, req *protocol.A2ATaskRequest) (*protocol.A2ATaskResponse, error) {
		return &protocol.A2ATaskResponse{
			TaskID: req.TaskID,
			State:  protocol.TaskStateCompleted,
			Result: []byte(`{"result":"success"}`),
		}, nil
	})

	handler := server.Handler()
	lab := a2a.NewCompatibilityLab()

	profile, err := lab.TestEndpoint(context.Background(), "test-agent", "1.0.0", "http://localhost", handler)
	if err != nil {
		t.Fatalf("failed to run compatibility lab: %v", err)
	}

	if profile.Status != a2a.StatusCompatible {
		t.Fatalf("expected status COMPATIBLE, got %s (passed: %d/%d)", profile.Status, profile.TestsPassed, profile.TestsTotal)
	}

	matrix := a2a.GenerateInteroperabilityMatrix()
	if len(matrix) < 2 {
		t.Fatalf("expected at least 2 matrix rows, got %d", len(matrix))
	}
}
