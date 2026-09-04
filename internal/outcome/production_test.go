package outcome

import (
	"testing"
	"time"
)

func TestComputeVerifiedOutcome(t *testing.T) {
	before := ProductionWindowSnapshot{
		StartTime:      time.Now().Add(-2 * time.Hour),
		EndTime:        time.Now().Add(-1 * time.Hour),
		TotalRequests:  1000,
		SuccessRate:    0.98,
		P95LatencyMs:   1200,
		CostPerTaskUSD: 0.050,
		QualityScore:   0.88,
	}

	afterImproved := ProductionWindowSnapshot{
		StartTime:      time.Now().Add(-1 * time.Hour),
		EndTime:        time.Now(),
		TotalRequests:  1200,
		SuccessRate:    0.99,  // +0.01
		P95LatencyMs:   950,   // -250ms (faster)
		CostPerTaskUSD: 0.038, // -$0.012 (cheaper)
		QualityScore:   0.91,  // +0.03
	}

	out := ComputeVerifiedOutcome("act-1", "org-1", "proj-1", "search", "ROUTE", "r-1", before, afterImproved)
	if out.Status != OutcomeImproved {
		t.Fatalf("Expected OutcomeImproved, got %s", out.Status)
	}
	if out.CostDeltaUSD >= 0 {
		t.Fatalf("Expected negative cost delta (savings), got %.4f", out.CostDeltaUSD)
	}
	if out.LatencyDeltaMs >= 0 {
		t.Fatalf("Expected negative latency delta (faster), got %d", out.LatencyDeltaMs)
	}

	// Regressed scenario
	afterRegressed := ProductionWindowSnapshot{
		StartTime:      time.Now().Add(-1 * time.Hour),
		EndTime:        time.Now(),
		TotalRequests:  800,
		SuccessRate:    0.92, // dropped
		P95LatencyMs:   1800, // increased
		CostPerTaskUSD: 0.080, // more expensive
	}
	outReg := ComputeVerifiedOutcome("act-2", "org-1", "proj-1", "search", "ROUTE", "r-1", before, afterRegressed)
	if outReg.Status != OutcomeRegressed {
		t.Fatalf("Expected OutcomeRegressed, got %s", outReg.Status)
	}
}
