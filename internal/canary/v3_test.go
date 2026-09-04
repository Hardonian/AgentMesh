package canary

import (
	"testing"
)

func TestCanaryEngineV3_PromotionAndRollback(t *testing.T) {
	engine := NewEngineV3()

	stages := []CanaryStageConfig{
		{TrafficWeightPercent: 10, MinRequests: 3, MinSuccessRate: 0.90, MaxAllowedErrorRate: 0.20, MaxAllowedLatencyMs: 2000, MaxPolicyViolations: 0},
		{TrafficWeightPercent: 100, MinRequests: 3, MinSuccessRate: 0.90, MaxAllowedErrorRate: 0.20, MaxAllowedLatencyMs: 2000, MaxPolicyViolations: 0},
	}

	// 1. Successful progressive promotion
	run, err := engine.StartRun("org-test", "proj-test", "cap-translate", TargetModelTarget, "gemini-1.5-flash", "gemini-1.5-pro", stages)
	if err != nil {
		t.Fatalf("Failed to start canary run: %v", err)
	}

	// Send 3 successful requests to complete stage 0 (10%)
	for i := 0; i < 3; i++ {
		r, dec, err := engine.RecordSample(run.ID, true, 450, 0.001, false)
		if err != nil {
			t.Fatalf("Failed recording sample: %v", err)
		}
		if i < 2 && dec != DecisionHold {
			t.Fatalf("Expected HOLD before reaching min requests, got %s", dec)
		}
		if i == 2 && dec != DecisionPromote {
			t.Fatalf("Expected PROMOTE after 3 successful requests, got %s", dec)
		}
		if i == 2 && r.CurrentStageIndex != 1 {
			t.Fatalf("Expected stage index 1, got %d", r.CurrentStageIndex)
		}
	}

	// Send 3 successful requests to complete stage 1 (100% full promotion)
	for i := 0; i < 3; i++ {
		r, dec, err := engine.RecordSample(run.ID, true, 420, 0.001, false)
		if err != nil {
			t.Fatalf("Failed recording sample: %v", err)
		}
		if i == 2 {
			if dec != DecisionPromote {
				t.Fatalf("Expected PROMOTE to ACTIVE, got %s", dec)
			}
			if r.State != StateActive {
				t.Fatalf("Expected state ACTIVE, got %s", r.State)
			}
		}
	}

	// 2. Automated Rollback on metric degradation
	run2, err := engine.StartRun("org-test", "proj-test", "cap-rag", TargetAgentVersion, "v1.0", "v2.0-buggy", stages)
	if err != nil {
		t.Fatalf("Failed to start canary run 2: %v", err)
	}

	// Record a policy violation -> immediate rollback
	r2, dec2, err := engine.RecordSample(run2.ID, true, 400, 0.001, true)
	if err != nil {
		t.Fatalf("Failed recording sample: %v", err)
	}
	if dec2 != DecisionRollback {
		t.Fatalf("Expected ROLLBACK on policy violation, got %s", dec2)
	}
	if r2.State != StateRolledBack {
		t.Fatalf("Expected state ROLLED_BACK, got %s", r2.State)
	}
	if r2.RollbackTrigger == "" {
		t.Fatalf("Expected non-empty rollback trigger explanation")
	}
}
