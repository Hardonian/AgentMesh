package reliability

import (
	"math"
	"sort"
	"sync"
	"time"
)

// EvidenceConfidence classifies certainty level based on sample size and freshness.
type EvidenceConfidence string

const (
	ConfidenceHigh      EvidenceConfidence = "HIGH_EVIDENCE"
	ConfidenceMedium    EvidenceConfidence = "MEDIUM_EVIDENCE"
	ConfidenceLow       EvidenceConfidence = "LOW_EVIDENCE"
	ConfidenceColdStart EvidenceConfidence = "COLD_START"
)

// RollingWindow captures metrics for a specific duration window.
type RollingWindow struct {
	WindowDuration time.Duration `json:"windowDuration"`
	SampleCount    int64         `json:"sampleCount"`
	SuccessCount   int64         `json:"successCount"`
	SuccessRate    float64       `json:"successRate"`
	P50LatencyMs   int64         `json:"p50LatencyMs"`
	P95LatencyMs   int64         `json:"p95LatencyMs"`
	P99LatencyMs   int64         `json:"p99LatencyMs"`
	AverageCostUSD float64       `json:"averageCostUsd"`
	ToolFailures   int64         `json:"toolFailures"`
	Timeouts       int64         `json:"timeouts"`
}

// ReliabilityProfile maintains statistical performance baselines per agent, version, and capability.
type ReliabilityProfile struct {
	AgentID             string             `json:"agentId"`
	Version             string             `json:"version"`
	CapabilityID        string             `json:"capabilityId"`
	TenantID            string             `json:"tenantId"`
	TotalSamples        int64              `json:"totalSamples"`
	OverallSuccessRate  float64            `json:"overallSuccessRate"`
	P50LatencyMs        int64              `json:"p50LatencyMs"`
	P95LatencyMs        int64              `json:"p95LatencyMs"`
	P99LatencyMs        int64              `json:"p99LatencyMs"`
	AverageCostUSD      float64            `json:"averageCostUsd"`
	MaxObservedCostUSD  float64            `json:"maxObservedCostUsd"`
	ToolCallSuccessRate float64            `json:"toolCallSuccessRate"`
	TimeoutRate         float64            `json:"timeoutRate"`
	Confidence          EvidenceConfidence `json:"confidence"`
	Window1m            RollingWindow      `json:"window1m"`
	Window5m            RollingWindow      `json:"window5m"`
	Window1h            RollingWindow      `json:"window1h"`
	Window24h           RollingWindow      `json:"window24h"`
	IncidentActive      bool               `json:"incidentActive"`
	LastUpdated         time.Time          `json:"lastUpdated"`
}

// OutcomeObservation captures a single runtime outcome for aggregation.
type OutcomeObservation struct {
	Success     bool
	LatencyMs   int64
	CostUSD     float64
	ToolSuccess bool
	IsTimeout   bool
	Timestamp   time.Time
}

// ReliabilityTracker maintains and computes statistical profiles in-memory.
type ReliabilityTracker struct {
	mu           sync.RWMutex
	observations map[string][]OutcomeObservation // tenant:agent:cap -> history
	profiles     map[string]*ReliabilityProfile  // tenant:agent:cap -> profile
}

// NewReliabilityTracker creates a thread-safe reliability profile tracker.
func NewReliabilityTracker() *ReliabilityTracker {
	return &ReliabilityTracker{
		observations: make(map[string][]OutcomeObservation),
		profiles:     make(map[string]*ReliabilityProfile),
	}
}

func profileKey(tenantID, agentID, capabilityID string) string {
	return tenantID + ":" + agentID + ":" + capabilityID
}

// RecordObservation records a new task outcome and recalculates rolling windows.
func (rt *ReliabilityTracker) RecordObservation(
	tenantID, agentID, version, capabilityID string,
	obs OutcomeObservation,
) *ReliabilityProfile {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	key := profileKey(tenantID, agentID, capabilityID)
	if obs.Timestamp.IsZero() {
		obs.Timestamp = time.Now().UTC()
	}

	rt.observations[key] = append(rt.observations[key], obs)

	// Keep bounded recent history (e.g. 1000 latest samples)
	if len(rt.observations[key]) > 1000 {
		rt.observations[key] = rt.observations[key][len(rt.observations[key])-1000:]
	}

	now := time.Now().UTC()
	history := rt.observations[key]

	w1m := computeWindow(history, now, 1*time.Minute)
	w5m := computeWindow(history, now, 5*time.Minute)
	w1h := computeWindow(history, now, 1*time.Hour)
	w24h := computeWindow(history, now, 24*time.Hour)

	totalSamples := int64(len(history))
	var successCount int64
	var toolSuccessCount int64
	var timeoutCount int64
	var totalCost float64
	var maxCost float64
	latencies := make([]int64, 0, totalSamples)

	for _, o := range history {
		if o.Success {
			successCount++
		}
		if o.ToolSuccess {
			toolSuccessCount++
		}
		if o.IsTimeout {
			timeoutCount++
		}
		totalCost += o.CostUSD
		if o.CostUSD > maxCost {
			maxCost = o.CostUSD
		}
		latencies = append(latencies, o.LatencyMs)
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p50 := percentile(latencies, 50)
	p95 := percentile(latencies, 95)
	p99 := percentile(latencies, 99)

	var overallSuccess float64
	var toolSuccessRate float64
	var timeoutRate float64
	var avgCost float64
	if totalSamples > 0 {
		overallSuccess = float64(successCount) / float64(totalSamples)
		toolSuccessRate = float64(toolSuccessCount) / float64(totalSamples)
		timeoutRate = float64(timeoutCount) / float64(totalSamples)
		avgCost = totalCost / float64(totalSamples)
	}

	confidence := ConfidenceColdStart
	switch {
	case totalSamples >= 100:
		confidence = ConfidenceHigh
	case totalSamples >= 25:
		confidence = ConfidenceMedium
	case totalSamples >= 5:
		confidence = ConfidenceLow
	}

	// Trigger incident if 5m window shows critical failure spike (>50% errors with >= 5 samples)
	incidentActive := false
	if w5m.SampleCount >= 5 && w5m.SuccessRate < 0.50 {
		incidentActive = true
	}

	profile := &ReliabilityProfile{
		AgentID:             agentID,
		Version:             version,
		CapabilityID:        capabilityID,
		TenantID:            tenantID,
		TotalSamples:        totalSamples,
		OverallSuccessRate:  overallSuccess,
		P50LatencyMs:        p50,
		P95LatencyMs:        p95,
		P99LatencyMs:        p99,
		AverageCostUSD:      avgCost,
		MaxObservedCostUSD:  maxCost,
		ToolCallSuccessRate: toolSuccessRate,
		TimeoutRate:         timeoutRate,
		Confidence:          confidence,
		Window1m:            w1m,
		Window5m:            w5m,
		Window1h:            w1h,
		Window24h:           w24h,
		IncidentActive:      incidentActive,
		LastUpdated:         now,
	}

	rt.profiles[key] = profile
	return profile
}

// GetProfile returns the calculated reliability profile.
func (rt *ReliabilityTracker) GetProfile(tenantID, agentID, capabilityID string) (*ReliabilityProfile, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	p, ok := rt.profiles[profileKey(tenantID, agentID, capabilityID)]
	return p, ok
}

func computeWindow(history []OutcomeObservation, now time.Time, duration time.Duration) RollingWindow {
	cutoff := now.Add(-duration)
	var count int64
	var success int64
	var toolFails int64
	var timeouts int64
	var totalCost float64
	lats := make([]int64, 0)

	for _, o := range history {
		if o.Timestamp.After(cutoff) {
			count++
			if o.Success {
				success++
			}
			if !o.ToolSuccess {
				toolFails++
			}
			if o.IsTimeout {
				timeouts++
			}
			totalCost += o.CostUSD
			lats = append(lats, o.LatencyMs)
		}
	}

	var sRate float64
	var avgCost float64
	if count > 0 {
		sRate = float64(success) / float64(count)
		avgCost = totalCost / float64(count)
	}

	sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })
	return RollingWindow{
		WindowDuration: duration,
		SampleCount:    count,
		SuccessCount:   success,
		SuccessRate:    sRate,
		P50LatencyMs:   percentile(lats, 50),
		P95LatencyMs:   percentile(lats, 95),
		P99LatencyMs:   percentile(lats, 99),
		AverageCostUSD: avgCost,
		ToolFailures:   toolFails,
		Timeouts:       timeouts,
	}
}

func percentile(sorted []int64, p int) int64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(p)/100.0*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
