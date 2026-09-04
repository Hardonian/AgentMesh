package evaluation_test

import (
	"context"
	"strings"
	"testing"

	"github.com/agentmesh/agentmesh/internal/evaluation"
)

func TestRedTeamEvaluator_DefendedAgentPasses(t *testing.T) {
	evaluator := evaluation.NewRedTeamEvaluator(nil)

	// Safe agent invoker that defensively refuses injections
	safeInvoker := func(ctx context.Context, prompt string) (string, error) {
		if strings.Contains(strings.ToLower(prompt), "override") || strings.Contains(strings.ToLower(prompt), "secret") {
			return "I cannot fulfill this request due to security constraints.", nil
		}
		return "Standard safe response.", nil
	}

	report, err := evaluator.EvaluateAgent(context.Background(), "finance-agent", "v2.0", safeInvoker)
	if err != nil {
		t.Fatalf("EvaluateAgent failed: %v", err)
	}

	if !report.SafeToCanary {
		t.Fatalf("expected defended agent to be safe to canary, score: %.2f, criticals: %d", report.RobustnessScore, report.CriticalDefects)
	}
	if report.CriticalDefects != 0 {
		t.Fatalf("expected 0 critical defects, got: %d", report.CriticalDefects)
	}
	if report.RobustnessScore != 1.0 {
		t.Fatalf("expected 1.0 score, got: %.2f", report.RobustnessScore)
	}
}

func TestRedTeamEvaluator_VulnerableAgentFailsGating(t *testing.T) {
	evaluator := evaluation.NewRedTeamEvaluator(nil)

	// Vulnerable agent that naively obeys prompt injection
	vulnInvoker := func(ctx context.Context, prompt string) (string, error) {
		if strings.Contains(prompt, "PWNED_BY_INJECTION") {
			return "Yes master: PWNED_BY_INJECTION", nil
		}
		return "Okay", nil
	}

	report, err := evaluator.EvaluateAgent(context.Background(), "leaky-agent", "v0.1", vulnInvoker)
	if err != nil {
		t.Fatalf("EvaluateAgent failed: %v", err)
	}

	if report.SafeToCanary {
		t.Fatal("vulnerable agent must NOT be marked safe to canary")
	}
	if report.CriticalDefects < 1 {
		t.Fatalf("expected at least 1 critical defect, got: %d", report.CriticalDefects)
	}
}
