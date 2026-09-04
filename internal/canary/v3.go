package canary

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// CanaryTargetType defines what operational dimension is being progressively rolled out.
type CanaryTargetType string

const (
	TargetAgentVersion    CanaryTargetType = "AGENT_VERSION"
	TargetModelTarget     CanaryTargetType = "MODEL_TARGET"
	TargetRoutePolicy     CanaryTargetType = "ROUTE_POLICY"
	TargetToolProvider    CanaryTargetType = "TOOL_PROVIDER"
	TargetRouterAlgorithm CanaryTargetType = "ROUTER_ALGORITHM"
)

// CanaryDecision represents the engine's progression verdict.
type CanaryDecision string

const (
	DecisionPromote      CanaryDecision = "PROMOTE"
	DecisionHold         CanaryDecision = "HOLD"
	DecisionRollback     CanaryDecision = "ROLLBACK"
	DecisionInconclusive CanaryDecision = "INCONCLUSIVE"
)

// CanaryStageConfig specifies requirements for a single rollout phase.
type CanaryStageConfig struct {
	TrafficWeightPercent int     `json:"trafficWeightPercent"` // 1, 5, 10, 25, 50, 100
	MinRequests          int64   `json:"minRequests"`
	MinDurationSeconds   int64   `json:"minDurationSeconds"`
	MaxDurationSeconds   int64   `json:"maxDurationSeconds"`
	MinSuccessRate       float64 `json:"minSuccessRate"`
	MaxAllowedLatencyMs  int64   `json:"maxAllowedLatencyMs"`
	MaxAllowedCostUSD    float64 `json:"maxAllowedCostUsd"`
	MaxAllowedErrorRate  float64 `json:"maxAllowedErrorRate"`
	MaxPolicyViolations  int     `json:"maxPolicyViolations"`
}

// CanaryRunV3 tracks a complete multi-stage progressive delivery run.
type CanaryRunV3 struct {
	ID                 string              `json:"id"`
	OrganizationID     string              `json:"organizationId"`
	ProjectID          string              `json:"projectId"`
	CapabilityID       string              `json:"capabilityId"`
	TargetType         CanaryTargetType    `json:"targetType"`
	BaselineTarget     string              `json:"baselineTarget"`
	CandidateTarget    string              `json:"candidateTarget"`
	Stages             []CanaryStageConfig `json:"stages"`
	CurrentStageIndex  int                 `json:"currentStageIndex"`
	State              RevisionState       `json:"state"` // CANARY, ACTIVE, ROLLED_BACK
	StageStartedAt     time.Time           `json:"stageStartedAt"`
	BaselineMetrics    MetricSnapshotV3    `json:"baselineMetrics"`
	CandidateMetrics   MetricSnapshotV3    `json:"candidateMetrics"`
	RollbackTrigger    string              `json:"rollbackTrigger,omitempty"`
	CreatedAt          time.Time           `json:"createdAt"`
	UpdatedAt          time.Time           `json:"updatedAt"`
}

// MetricSnapshotV3 records operational metrics for Canary V3.
type MetricSnapshotV3 struct {
	TotalRequests      int64   `json:"totalRequests"`
	SuccessCount       int64   `json:"successCount"`
	SuccessRate        float64 `json:"successRate"`
	ErrorCount         int64   `json:"errorCount"`
	ErrorRate          float64 `json:"errorRate"`
	P50LatencyMs       int64   `json:"p50LatencyMs"`
	P95LatencyMs       int64   `json:"p95LatencyMs"`
	AverageCostUSD     float64 `json:"averageCostUsd"`
	QualityScore       float64 `json:"qualityScore"`
	ToolSuccessRate    float64 `json:"toolSuccessRate"`
	PolicyViolations   int     `json:"policyViolations"`
	FallbackCount      int64   `json:"fallbackCount"`
	FallbackRate       float64 `json:"fallbackRate"`
}

// EngineV3 manages multi-target progressive delivery and evaluation.
type EngineV3 struct {
	mu   sync.RWMutex
	runs map[string]*CanaryRunV3 // RunID -> CanaryRunV3
}

// NewEngineV3 creates a new Canary V3 engine.
func NewEngineV3() *EngineV3 {
	return &EngineV3{
		runs: make(map[string]*CanaryRunV3),
	}
}

// DefaultStages returns the canonical 6-stage progressive delivery progression.
func DefaultStages() []CanaryStageConfig {
	return []CanaryStageConfig{
		{TrafficWeightPercent: 1, MinRequests: 10, MinDurationSeconds: 60, MinSuccessRate: 0.98, MaxAllowedErrorRate: 0.02},
		{TrafficWeightPercent: 5, MinRequests: 25, MinDurationSeconds: 120, MinSuccessRate: 0.98, MaxAllowedErrorRate: 0.02},
		{TrafficWeightPercent: 10, MinRequests: 50, MinDurationSeconds: 300, MinSuccessRate: 0.98, MaxAllowedErrorRate: 0.02},
		{TrafficWeightPercent: 25, MinRequests: 100, MinDurationSeconds: 600, MinSuccessRate: 0.99, MaxAllowedErrorRate: 0.01},
		{TrafficWeightPercent: 50, MinRequests: 200, MinDurationSeconds: 1200, MinSuccessRate: 0.99, MaxAllowedErrorRate: 0.01},
		{TrafficWeightPercent: 100, MinRequests: 500, MinDurationSeconds: 1800, MinSuccessRate: 0.99, MaxAllowedErrorRate: 0.01},
	}
}

// StartRun starts a new Canary V3 progressive delivery process.
func (e *EngineV3) StartRun(orgID, projID, capabilityID string, targetType CanaryTargetType, baseline, candidate string, stages []CanaryStageConfig) (*CanaryRunV3, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(stages) == 0 {
		stages = DefaultStages()
	}

	now := time.Now().UTC()
	runID := fmt.Sprintf("canary_%s_%s_%d", capabilityID, targetType, now.Unix())

	run := &CanaryRunV3{
		ID:                runID,
		OrganizationID:    orgID,
		ProjectID:         projID,
		CapabilityID:      capabilityID,
		TargetType:        targetType,
		BaselineTarget:    baseline,
		CandidateTarget:   candidate,
		Stages:            stages,
		CurrentStageIndex: 0,
		State:             StateCanary,
		StageStartedAt:    now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	e.runs[runID] = run
	return run, nil
}

// RecordSample updates candidate metrics during a canary run.
func (e *EngineV3) RecordSample(runID string, success bool, latencyMs int64, costUSD float64, policyViolation bool) (*CanaryRunV3, CanaryDecision, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	run, exists := e.runs[runID]
	if !exists {
		return nil, DecisionInconclusive, errors.New("canary run not found")
	}

	if run.State != StateCanary {
		return run, DecisionHold, nil
	}

	m := &run.CandidateMetrics
	prevTotal := float64(m.TotalRequests)
	newTotal := prevTotal + 1
	m.TotalRequests = int64(newTotal)

	if success {
		m.SuccessCount++
	} else {
		m.ErrorCount++
	}
	m.SuccessRate = float64(m.SuccessCount) / newTotal
	m.ErrorRate = float64(m.ErrorCount) / newTotal
	m.AverageCostUSD = (m.AverageCostUSD*prevTotal + costUSD) / newTotal

	if latencyMs > m.P95LatencyMs {
		m.P95LatencyMs = latencyMs
	}
	if policyViolation {
		m.PolicyViolations++
	}

	run.UpdatedAt = time.Now().UTC()

	// Evaluate against current stage thresholds
	currentStage := run.Stages[run.CurrentStageIndex]

	// 1. Check Rollback Triggers
	if currentStage.MaxPolicyViolations >= 0 && m.PolicyViolations > currentStage.MaxPolicyViolations {
		run.State = StateRolledBack
		run.RollbackTrigger = fmt.Sprintf("policy violations (%d) exceeded threshold (%d)",
			m.PolicyViolations, currentStage.MaxPolicyViolations)
		return run, DecisionRollback, nil
	}

	if m.TotalRequests >= 5 {
		if currentStage.MaxAllowedErrorRate > 0 && m.ErrorRate > currentStage.MaxAllowedErrorRate {
			run.State = StateRolledBack
			run.RollbackTrigger = fmt.Sprintf("error rate (%.2f%%) exceeded threshold (%.2f%%)",
				m.ErrorRate*100, currentStage.MaxAllowedErrorRate*100)
			return run, DecisionRollback, nil
		}

		if currentStage.MaxAllowedLatencyMs > 0 && m.P95LatencyMs > currentStage.MaxAllowedLatencyMs {
			run.State = StateRolledBack
			run.RollbackTrigger = fmt.Sprintf("P95 latency (%dms) exceeded threshold (%dms)",
				m.P95LatencyMs, currentStage.MaxAllowedLatencyMs)
			return run, DecisionRollback, nil
		}
	}

	// 2. Check Promotion Criteria
	if m.TotalRequests >= currentStage.MinRequests {
		if m.SuccessRate >= currentStage.MinSuccessRate {
			// Ready to promote to next stage or full ACTIVE
			if run.CurrentStageIndex+1 < len(run.Stages) {
				run.CurrentStageIndex++
				run.StageStartedAt = time.Now().UTC()
				// Reset sample counts for next stage
				m.TotalRequests = 0
				m.SuccessCount = 0
				m.ErrorCount = 0
				return run, DecisionPromote, nil
			} else {
				// Reached 100% final promotion!
				run.State = StateActive
				return run, DecisionPromote, nil
			}
		}
	}

	return run, DecisionHold, nil
}

// GetRun retrieves a canary run by ID.
func (e *EngineV3) GetRun(runID string) (*CanaryRunV3, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	run, exists := e.runs[runID]
	if !exists {
		return nil, errors.New("canary run not found")
	}
	return run, nil
}
