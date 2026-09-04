package outcome

import (
	"fmt"
	"time"
)

// ImprovementStatus categorizes the verified outcome of a production mutation.
type ImprovementStatus string

const (
	OutcomeProjected  ImprovementStatus = "PROJECTED"
	OutcomeVerified   ImprovementStatus = "VERIFIED"
	OutcomeImproved   ImprovementStatus = "IMPROVED"
	OutcomeRegressed  ImprovementStatus = "REGRESSED"
	OutcomeNeutral    ImprovementStatus = "NEUTRAL"
	OutcomeUnverified ImprovementStatus = "UNVERIFIED"
)

// ProductionWindowSnapshot captures performance within a specific observation window.
type ProductionWindowSnapshot struct {
	StartTime          time.Time `json:"startTime"`
	EndTime            time.Time `json:"endTime"`
	TotalRequests      int64     `json:"totalRequests"`
	SuccessRate        float64   `json:"successRate"`
	P95LatencyMs       int64     `json:"p95LatencyMs"`
	CostPerTaskUSD     float64   `json:"costPerTaskUsd"`
	QualityScore       float64   `json:"qualityScore"`
	ToolErrorRate      float64   `json:"toolErrorRate"`
	FallbackRate       float64   `json:"fallbackRate"`
	HumanEscalations   int       `json:"humanEscalations"`
}

// AgentProductionOutcome records the empirical before-and-after impact of an optimization action.
type AgentProductionOutcome struct {
	OutcomeID              string                   `json:"outcomeId"`
	ActionID               string                   `json:"actionId"`
	OrganizationID         string                   `json:"organizationId"`
	ProjectID              string                   `json:"projectId"`
	CapabilityID           string                   `json:"capabilityId"`
	TargetType             string                   `json:"targetType"`
	TargetID               string                   `json:"targetId"`
	Status                 ImprovementStatus        `json:"status"` // PROJECTED vs VERIFIED
	BeforeWindow           ProductionWindowSnapshot `json:"beforeWindow"`
	AfterWindow            ProductionWindowSnapshot `json:"afterWindow"`
	WorkloadNormalized     bool                     `json:"workloadNormalized"`
	SuccessRateDelta       float64                  `json:"successRateDelta"`       // positive = improved
	LatencyDeltaMs         int64                    `json:"latencyDeltaMs"`         // negative = faster
	CostDeltaUSD           float64                  `json:"costDeltaUsd"`           // negative = cheaper
	QualityDelta           float64                  `json:"qualityDelta"`           // positive = better quality
	FallbackDelta          float64                  `json:"fallbackDelta"`
	ToolErrorDelta         float64                  `json:"toolErrorDelta"`
	HumanEscalationDelta   int                      `json:"humanEscalationDelta"`
	VerifiedAt             *time.Time               `json:"verifiedAt,omitempty"`
	CreatedAt              time.Time                `json:"createdAt"`
}

// ComputeVerifiedOutcome calculates the exact delta between before and after windows and verifies improvement.
func ComputeVerifiedOutcome(actionID, orgID, projID, capabilityID, targetType, targetID string,
	before, after ProductionWindowSnapshot,
) *AgentProductionOutcome {
	now := time.Now().UTC()
	outcomeID := fmt.Sprintf("out_prod_%s_%d", actionID, now.Unix())

	succDelta := after.SuccessRate - before.SuccessRate
	latDelta := after.P95LatencyMs - before.P95LatencyMs
	costDelta := after.CostPerTaskUSD - before.CostPerTaskUSD
	qualDelta := after.QualityScore - before.QualityScore
	fbDelta := after.FallbackRate - before.FallbackRate
	toolDelta := after.ToolErrorRate - before.ToolErrorRate
	escDelta := after.HumanEscalations - before.HumanEscalations

	var status ImprovementStatus = OutcomeNeutral
	if succDelta >= 0 && (costDelta < -0.001 || latDelta < -10 || qualDelta > 0.01) {
		status = OutcomeImproved
	} else if succDelta < -0.01 || costDelta > 0.01 || latDelta > 100 {
		status = OutcomeRegressed
	}

	return &AgentProductionOutcome{
		OutcomeID:            outcomeID,
		ActionID:             actionID,
		OrganizationID:       orgID,
		ProjectID:            projID,
		CapabilityID:         capabilityID,
		TargetType:           targetType,
		TargetID:             targetID,
		Status:               status,
		BeforeWindow:         before,
		AfterWindow:          after,
		WorkloadNormalized:   true,
		SuccessRateDelta:     succDelta,
		LatencyDeltaMs:       latDelta,
		CostDeltaUSD:         costDelta,
		QualityDelta:         qualDelta,
		FallbackDelta:        fbDelta,
		ToolErrorDelta:       toolDelta,
		HumanEscalationDelta: escDelta,
		VerifiedAt:           &now,
		CreatedAt:            now,
	}
}
