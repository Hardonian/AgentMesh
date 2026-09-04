package optimize

import (
	"fmt"
	"sync"
	"time"

	"github.com/agentmesh/agentmesh/pkg/spec"
)

// RecommendationStatus indicates the optimizer's evaluation result.
type RecommendationStatus string

const (
	StatusNoChange       RecommendationStatus = "NO_CHANGE"
	StatusRecommendation RecommendationStatus = "RECOMMENDATION"
	StatusActionEligible RecommendationStatus = "ACTION_ELIGIBLE"
)

// CandidateOptimization contains an evaluated improvement opportunity.
type CandidateOptimization struct {
	RecommendationID string                       `json:"recommendationId"`
	CapabilityID     string                       `json:"capabilityId"`
	CurrentAgentID   string                       `json:"currentAgentId"`
	CandidateAgentID string                       `json:"candidateAgentId"`
	Status           RecommendationStatus         `json:"status"`
	Reason           string                       `json:"reason"`
	CostDeltaPercent float64                      `json:"costDeltaPercent"`
	LatencyDeltaMs   int64                        `json:"latencyDeltaMs"`
	ActionProposal   *spec.AgentOptimizationAction `json:"actionProposal,omitempty"`
	EvaluatedAt      time.Time                    `json:"evaluatedAt"`
}

// Scheduler evaluates continuous optimization opportunities without executing them directly.
type Scheduler struct {
	mu                     sync.RWMutex
	minCostImprovementPct  float64
	minLatencyImprovementMs int64
	cooldownDuration       time.Duration
	lastMutatedAt          map[string]time.Time // CapabilityID -> Timestamp
}

// NewScheduler creates an optimization scheduler.
func NewScheduler(minCostPct float64, minLatencyMs int64, cooldown time.Duration) *Scheduler {
	if minCostPct <= 0 {
		minCostPct = 10.0 // Minimum 10% cost reduction to prevent route churn
	}
	if minLatencyMs <= 0 {
		minLatencyMs = 50 // Minimum 50ms latency improvement
	}
	if cooldown <= 0 {
		cooldown = 15 * time.Minute // 15m cooldown between non-emergency changes
	}
	return &Scheduler{
		minCostImprovementPct:   minCostPct,
		minLatencyImprovementMs: minLatencyMs,
		cooldownDuration:        cooldown,
		lastMutatedAt:           make(map[string]time.Time),
	}
}

// RecordMutationTimestamp updates cooldown tracking for a capability.
func (s *Scheduler) RecordMutationTimestamp(capabilityID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastMutatedAt[capabilityID] = time.Now().UTC()
}

// EvaluateOpportunity checks if a candidate agent materially outperforms the current route.
func (s *Scheduler) EvaluateOpportunity(orgID, projID, capabilityID string,
	currentAgent string, currentCost float64, currentLatencyMs int64,
	candidateAgent string, candidateCost float64, candidateLatencyMs int64,
	candidateReliability float64,
) *CandidateOptimization {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	recID := fmt.Sprintf("rec_%s_%d", capabilityID, now.Unix())

	// 1. Check Cooldown
	if lastTime, exists := s.lastMutatedAt[capabilityID]; exists {
		if now.Sub(lastTime) < s.cooldownDuration {
			return &CandidateOptimization{
				RecommendationID: recID,
				CapabilityID:     capabilityID,
				CurrentAgentID:   currentAgent,
				CandidateAgentID: candidateAgent,
				Status:           StatusNoChange,
				Reason:           fmt.Sprintf("capability is in cooldown period until %s", lastTime.Add(s.cooldownDuration).Format(time.RFC3339)),
				EvaluatedAt:      now,
			}
		}
	}

	// 2. Compute Deltas
	var costImprovementPct float64
	if currentCost > 0 {
		costImprovementPct = ((currentCost - candidateCost) / currentCost) * 100.0
	}
	latencyImprovementMs := currentLatencyMs - candidateLatencyMs

	// 3. Minimum benefit threshold & reliability check
	if candidateReliability < 0.98 {
		return &CandidateOptimization{
			RecommendationID: recID,
			CapabilityID:     capabilityID,
			CurrentAgentID:   currentAgent,
			CandidateAgentID: candidateAgent,
			Status:           StatusNoChange,
			Reason:           fmt.Sprintf("candidate reliability (%.2f%%) is below required 98%% threshold", candidateReliability*100),
			EvaluatedAt:      now,
		}
	}

	if costImprovementPct >= s.minCostImprovementPct || latencyImprovementMs >= s.minLatencyImprovementMs {
		action := &spec.AgentOptimizationAction{
			ActionID:       fmt.Sprintf("act_opt_%s_%d", capabilityID, now.Unix()),
			OrganizationID: orgID,
			ProjectID:      projID,
			CapabilityID:   capabilityID,
			TargetType:     "ROUTE",
			TargetID:       capabilityID,
			ActionType:     spec.ActionChangeRouteWeight,
			CurrentState: map[string]any{
				"agent":  currentAgent,
				"weight": 100,
			},
			ProposedState: map[string]any{
				"agent":  candidateAgent,
				"weight": 100,
			},
			Reason: fmt.Sprintf("Candidate %s offers %.1f%% cost savings and %dms latency improvement",
				candidateAgent, costImprovementPct, latencyImprovementMs),
			CostDeltaEstimate:      candidateCost - currentCost,
			LatencyDeltaEstimateMs: -latencyImprovementMs,
			Confidence:             0.92,
			RiskClass:              spec.RiskLow,
			BlastRadius: spec.BlastRadius{
				TrafficPercent:       10, // Recommend initial canary
				CapabilitiesAffected: []string{capabilityID},
			},
			CreatedAt:     now,
			SchemaVersion: "v1",
		}

		return &CandidateOptimization{
			RecommendationID: recID,
			CapabilityID:     capabilityID,
			CurrentAgentID:   currentAgent,
			CandidateAgentID: candidateAgent,
			Status:           StatusActionEligible,
			Reason:           action.Reason,
			CostDeltaPercent: costImprovementPct,
			LatencyDeltaMs:   latencyImprovementMs,
			ActionProposal:   action,
			EvaluatedAt:      now,
		}
	}

	return &CandidateOptimization{
		RecommendationID: recID,
		CapabilityID:     capabilityID,
		CurrentAgentID:   currentAgent,
		CandidateAgentID: candidateAgent,
		Status:           StatusNoChange,
		Reason:           "cost or latency improvement below required threshold",
		CostDeltaPercent: costImprovementPct,
		LatencyDeltaMs:   latencyImprovementMs,
		EvaluatedAt:      now,
	}
}
