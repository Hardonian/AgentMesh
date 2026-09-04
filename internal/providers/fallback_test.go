package providers_test

import (
	"context"
	"testing"

	"github.com/agentmesh/agentmesh/internal/providers"
)

func TestModelFallback(t *testing.T) {
	router := providers.NewModelRouter()
	router.RegisterTarget(&providers.ModelTarget{
		ModelID:      "failing-model-id",
		Provider:     "gemini",
		HealthStatus: "UNAVAILABLE",
	})

	req := &providers.GenerateRequest{
		Prompt: "Hello, AgentMesh",
	}

	// Case 1: Permitted fallback
	resp, fbEvent, err := router.GenerateWithFallback(context.Background(), "failing-model-id", "gemini-1.5-flash", []string{"*"}, req)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}
	if resp == nil || resp.ContentText == "" {
		t.Fatalf("expected generated content from fallback model")
	}
	if fbEvent == nil || !fbEvent.AllowedByPolicy {
		t.Fatalf("expected recorded fallback event allowed by policy")
	}

	// Case 2: Forbidden fallback by policy
	_, fbEvent2, err2 := router.GenerateWithFallback(context.Background(), "failing-model-id", "claude-3-5-sonnet", []string{"gemini-*"}, req)
	if err2 == nil {
		t.Fatalf("expected error when fallback model is forbidden by allowedModels policy")
	}
	if fbEvent2 == nil || fbEvent2.AllowedByPolicy {
		t.Fatalf("expected forbidden fallback event recorded")
	}
}
