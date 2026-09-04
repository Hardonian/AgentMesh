package shadow

import (
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

func TestShadowExecutionAndContainment(t *testing.T) {
	mgr := NewManager()

	// 1. Tool suppression
	tools := []string{"bigquery.read", "payment.charge", "document.summarize", "email.send"}
	permitted, suppressed := FilterPermittedShadowTools(tools)

	if len(permitted) != 2 {
		t.Fatalf("Expected 2 permitted read-only tools, got %d: %v", len(permitted), permitted)
	}
	if len(suppressed) != 2 {
		t.Fatalf("Expected 2 suppressed side-effecting tools, got %d: %v", len(suppressed), suppressed)
	}

	// 2. Record shadow execution
	report, err := mgr.RecordShadowExecution(
		"task-101",
		"finance_report",
		ModeRealTrafficWithoutSideEffects,
		"agent-v1",
		"agent-v2-candidate",
		1200,
		950,
		0.05,
		0.03,
		tools,
	)
	if err != nil {
		t.Fatalf("Failed to record shadow execution: %v", err)
	}

	if !report.SideEffectsContained {
		t.Fatalf("Expected side effects to be contained")
	}
	if len(report.SuppressedToolCalls) != 2 {
		t.Fatalf("Expected 2 suppressed calls in report")
	}

	// 3. GoldenTaskSet evaluation
	gts := &spec.GoldenTaskSet{
		ID:           "gts-1",
		CapabilityID: "finance_report",
		Version:      "v1",
		Tasks: []spec.GoldenTask{
			{TaskID: "gt-1", PermittedTools: []string{"bigquery.read"}},
			{TaskID: "gt-2", PermittedTools: []string{"document.summarize"}},
		},
		CreatedAt: time.Now().UTC(),
	}

	passed, rate, err := mgr.EvaluateGoldenTaskSet("agent-v2-candidate", gts)
	if err != nil || !passed || rate < 1.0 {
		t.Fatalf("Expected golden task set to pass with 100%%, got %v, rate %.2f", passed, rate)
	}
}
