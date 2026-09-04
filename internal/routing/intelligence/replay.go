package intelligence

import (
	"context"
	"math"
	"time"

	"github.com/agentmesh/agentmesh/internal/routing"
)

// CounterfactualStatus marks whether a metric was observed or estimated.
type CounterfactualStatus string

const (
	StatusObserved  CounterfactualStatus = "OBSERVED"
	StatusEstimated CounterfactualStatus = "ESTIMATED"
	StatusUnknown   CounterfactualStatus = "UNKNOWN"
)

// RegretMetrics records estimated regret vs optimal alternative.
type RegretMetrics struct {
	CostRegret           float64              `json:"costRegret"`
	LatencyRegretMs      int64                `json:"latencyRegretMs"`
	QualityRegret        float64              `json:"qualityRegret"`
	CounterfactualStatus CounterfactualStatus `json:"counterfactualStatus"`
}

// ReplayComparison compares historical route selection against candidate router selection.
type ReplayComparison struct {
	TaskID                 string         `json:"taskId"`
	CapabilityID           string         `json:"capabilityId"`
	HistoricalSelected     string         `json:"historicalSelected"`
	CandidateRouterChoice  string         `json:"candidateRouterChoice"`
	DecisionsMatch         bool           `json:"decisionsMatch"`
	EstimatedCostDeltaUSD  float64        `json:"estimatedCostDeltaUsd"`  // Negative means savings
	EstimatedLatencyDelta  int64          `json:"estimatedLatencyDeltaMs"` // Negative means speedup
	Regret                 RegretMetrics  `json:"regret"`
}

// ReplaySummary aggregates offline replay results across a corpus of tasks.
type ReplaySummary struct {
	TotalReplayedTasks int                `json:"totalReplayedTasks"`
	AgreementCount     int                `json:"agreementCount"`
	AgreementRate      float64            `json:"agreementRate"`
	TotalCostDeltaUSD  float64            `json:"totalCostDeltaUsd"`
	AvgLatencyDeltaMs  float64            `json:"avgLatencyDeltaMs"`
	CostImprovementPct float64            `json:"costImprovementPct"`
	Comparisons        []ReplayComparison `json:"comparisons"`
	EvaluatedAt        time.Time          `json:"evaluatedAt"`
}

// ReplayEngine executes offline routing replay without live traffic impact.
type ReplayEngine struct {
	router *BaselineRouterV1
}

// NewReplayEngine creates an offline routing replay evaluator.
func NewReplayEngine(router *BaselineRouterV1) *ReplayEngine {
	return &ReplayEngine{router: router}
}

// ReplayCorpus evaluates the candidate router against historical RoutingOutcomes.
func (re *ReplayEngine) ReplayCorpus(
	ctx context.Context,
	historicalOutcomes []*routing.CanonicalRoutingOutcome,
	candidates []*CandidateAgent,
) (*ReplaySummary, error) {
	summary := &ReplaySummary{
		TotalReplayedTasks: len(historicalOutcomes),
		Comparisons:        make([]ReplayComparison, 0, len(historicalOutcomes)),
		EvaluatedAt:        time.Now().UTC(),
	}

	var totalHistCost float64
	var totalCandidateCost float64
	var totalLatencyDelta int64

	for _, hist := range historicalOutcomes {
		fp := hist.RequestFeatures
		if fp == nil {
			continue
		}

		// Run candidate router simulation
		res, err := re.router.Route(ctx, fp, hist.OrganizationID, ObjectiveBalanced, nil, candidates, hist.PolicyVersion)
		if err != nil {
			continue
		}

		match := (res.SelectedAgentID == hist.SelectedAgentID)
		if match {
			summary.AgreementCount++
		}

		// Estimate counterfactual outcomes based on candidate reliability profile if available
		var candCost float64 = hist.Cost
		var candLat int64 = hist.LatencyMs
		status := StatusObserved

		if !match {
			// Find selected candidate's profile
			status = StatusEstimated
			for _, c := range candidates {
				if c.AgentID == res.SelectedAgentID && c.ReliabilityProfile != nil {
					candCost = c.ReliabilityProfile.AverageCostUSD
					candLat = c.ReliabilityProfile.P50LatencyMs
					break
				}
			}
		}

		costDelta := candCost - hist.Cost
		latDelta := candLat - hist.LatencyMs

		totalHistCost += hist.Cost
		totalCandidateCost += candCost
		totalLatencyDelta += latDelta

		comp := ReplayComparison{
			TaskID:                hist.TaskID,
			CapabilityID:          hist.CapabilityID,
			HistoricalSelected:    hist.SelectedAgentID,
			CandidateRouterChoice: res.SelectedAgentID,
			DecisionsMatch:        match,
			EstimatedCostDeltaUSD: costDelta,
			EstimatedLatencyDelta: latDelta,
			Regret: RegretMetrics{
				CostRegret:           math.Max(0, costDelta),
				LatencyRegretMs:      int64(math.Max(0, float64(latDelta))),
				QualityRegret:        0.0,
				CounterfactualStatus: status,
			},
		}

		summary.Comparisons = append(summary.Comparisons, comp)
	}

	if summary.TotalReplayedTasks > 0 {
		summary.AgreementRate = float64(summary.AgreementCount) / float64(summary.TotalReplayedTasks)
		summary.TotalCostDeltaUSD = totalCandidateCost - totalHistCost
		summary.AvgLatencyDeltaMs = float64(totalLatencyDelta) / float64(summary.TotalReplayedTasks)
		if totalHistCost > 0 {
			summary.CostImprovementPct = -(summary.TotalCostDeltaUSD / totalHistCost) * 100.0
		}
	}

	return summary, nil
}
