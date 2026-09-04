package reliability

import (
	"testing"
	"time"
)

func TestReliabilityTrackerRollingWindows(t *testing.T) {
	tracker := NewReliabilityTracker()
	tenant := "acme-corp"
	agent := "research-agent"
	ver := "1.0.0"
	cap := "deep_search"

	now := time.Now().UTC()

	// Add 10 observations: 9 successes, 1 failure with timeout
	for i := 0; i < 9; i++ {
		tracker.RecordObservation(tenant, agent, ver, cap, OutcomeObservation{
			Success:     true,
			LatencyMs:   int64(200 + i*50),
			CostUSD:     0.02,
			ToolSuccess: true,
			IsTimeout:   false,
			Timestamp:   now.Add(-time.Duration(i) * time.Second),
		})
	}

	p := tracker.RecordObservation(tenant, agent, ver, cap, OutcomeObservation{
		Success:     false,
		LatencyMs:   5000,
		CostUSD:     0.05,
		ToolSuccess: false,
		IsTimeout:   true,
		Timestamp:   now,
	})

	if p.TotalSamples != 10 {
		t.Fatalf("expected 10 total samples, got %d", p.TotalSamples)
	}
	if p.OverallSuccessRate != 0.90 {
		t.Errorf("expected 0.90 success rate, got %f", p.OverallSuccessRate)
	}
	if p.Confidence != ConfidenceLow {
		t.Errorf("expected ConfidenceLow for 10 samples, got %s", p.Confidence)
	}
	if p.P50LatencyMs == 0 || p.P95LatencyMs == 0 {
		t.Errorf("expected non-zero latency percentiles, got P50=%d P95=%d", p.P50LatencyMs, p.P95LatencyMs)
	}
	if p.TimeoutRate != 0.10 {
		t.Errorf("expected 0.10 timeout rate, got %f", p.TimeoutRate)
	}
	if p.IncidentActive {
		t.Errorf("expected incident false for 90%% success rate")
	}

	// Trigger incident by recording 10 consecutive failures (making total 9 successes, 11 failures = 45% success rate)
	for i := 0; i < 10; i++ {
		p = tracker.RecordObservation(tenant, agent, ver, cap, OutcomeObservation{
			Success:     false,
			LatencyMs:   5000,
			CostUSD:     0.05,
			ToolSuccess: false,
			IsTimeout:   true,
			Timestamp:   now,
		})
	}

	if !p.IncidentActive {
		t.Errorf("expected incident to be active after severe failure spike")
	}
}
