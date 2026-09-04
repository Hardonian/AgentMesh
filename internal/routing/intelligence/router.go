package intelligence

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/reliability"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/internal/slo"
	"github.com/agentmesh/agentmesh/pkg/task"
)

// RoutingObjective defines the primary optimization target.
type RoutingObjective string

const (
	ObjectiveReliability RoutingObjective = "RELIABILITY"
	ObjectiveQuality     RoutingObjective = "QUALITY"
	ObjectiveLatency     RoutingObjective = "LATENCY"
	ObjectiveCost        RoutingObjective = "COST"
	ObjectiveBalanced    RoutingObjective = "BALANCED"
	ObjectivePolicyFirst RoutingObjective = "POLICY_FIRST"
)

// CandidateAgent represents an agent evaluated during route ranking.
type CandidateAgent struct {
	AgentID            string                         `json:"agentId"`
	Version            string                         `json:"version"`
	EndpointURL        string                         `json:"endpointUrl"`
	HealthStatus       string                         `json:"healthStatus"` // HEALTHY, DEGRADED, UNHEALTHY
	SupportedTools     []string                       `json:"supportedTools"`
	Region             string                         `json:"region"`
	AllowedDataClasses []string                       `json:"allowedDataClasses"`
	EvidenceTier       routing.CapabilityEvidenceTier `json:"evidenceTier"`
	QualityScore       float64                        `json:"qualityScore"`
	ReliabilityProfile *reliability.ReliabilityProfile `json:"reliabilityProfile,omitempty"`
	SLO                *slo.AgentSLO                  `json:"slo,omitempty"`
}

// ScoredCandidate contains component score breakdown for explainability.
type ScoredCandidate struct {
	Candidate              *CandidateAgent `json:"candidate"`
	Eligible               bool            `json:"eligible"`
	DisqualificationReason string          `json:"disqualificationReason,omitempty"`

	// Component Dimensions (0.0 to 1.0)
	PolicyScore      float64 `json:"policyScore"`
	ReliabilityScore float64 `json:"reliabilityScore"`
	QualityScore     float64 `json:"qualityScore"`
	LatencyScore     float64 `json:"latencyScore"`
	CostScore        float64 `json:"costScore"`
	SLOScore         float64 `json:"sloScore"`

	CompositeScore float64 `json:"compositeScore"`
}

// RouteResult contains the final routing choice, confidence, and full explanation.
type RouteResult struct {
	SelectedAgentID        string             `json:"selectedAgentId"`
	SelectedVersion        string             `json:"selectedVersion"`
	EndpointURL            string             `json:"endpointUrl"`
	Objective              RoutingObjective   `json:"objective"`
	AlgorithmID            string             `json:"algorithmId"`
	AlgorithmVersion       string             `json:"algorithmVersion"`
	PolicyVersion          string             `json:"policyVersion"`
	Confidence             float64            `json:"confidence"`
	IsFailover             bool               `json:"isFailover"`
	FailoverOriginalAgent  string             `json:"failoverOriginalAgent,omitempty"`
	Candidates             []*ScoredCandidate `json:"candidates"`
	DecidedAt              time.Time          `json:"decidedAt"`
	DecisionExplanation    string             `json:"decisionExplanation"`
}

// BaselineRouterV1 implements deterministic 9-step routing intelligence.
type BaselineRouterV1 struct {
	AlgorithmID      string
	AlgorithmVersion string
	HysteresisDelta  float64 // Minimum score lift needed to displace active incumbent agent (e.g. 0.05)
	ActiveRoutes     map[string]string // tenant:cap -> currently preferred agent
}

// NewBaselineRouterV1 creates the standard deterministic baseline router.
func NewBaselineRouterV1() *BaselineRouterV1 {
	return &BaselineRouterV1{
		AlgorithmID:      "BASELINE_ROUTER_V1",
		AlgorithmVersion: "1.0.0",
		HysteresisDelta:  0.05,
		ActiveRoutes:     make(map[string]string),
	}
}

// Route executes the 9-step deterministic routing process.
func (r *BaselineRouterV1) Route(
	ctx context.Context,
	req *task.TaskFingerprint,
	tenantID string,
	objective RoutingObjective,
	policyEng *policy.Engine,
	candidates []*CandidateAgent,
	policyVersion string,
) (*RouteResult, error) {
	if len(candidates) == 0 {
		return nil, errors.New("no candidates available for routing")
	}
	if objective == "" {
		objective = ObjectiveBalanced
	}

	result := &RouteResult{
		Objective:        objective,
		AlgorithmID:      r.AlgorithmID,
		AlgorithmVersion: r.AlgorithmVersion,
		PolicyVersion:    policyVersion,
		DecidedAt:        time.Now().UTC(),
		Candidates:       make([]*ScoredCandidate, 0, len(candidates)),
	}

	// 1. Evaluate each candidate against the 9-step pipeline
	for _, c := range candidates {
		sc := &ScoredCandidate{
			Candidate: c,
			Eligible:  true,
		}

		// Step 1: Policy Eligibility (Strict Filtering - DENY is absolute)
		if policyEng != nil {
			evalReq := &policy.EvaluationRequest{
				TenantID:           tenantID,
				SubjectAgentID:     c.AgentID,
				Capability:         req.Capability,
				Action:             "route.invoke",
				DataClassification: req.DataClassification,
			}
			decision := policyEng.Evaluate(ctx, evalReq)
			if decision.Effect == policy.EffectDeny {
				sc.Eligible = false
				sc.DisqualificationReason = fmt.Sprintf("Policy DENY: %s", decision.Reason)
				sc.PolicyScore = 0.0
				result.Candidates = append(result.Candidates, sc)
				continue
			}
			sc.PolicyScore = 1.0
		} else {
			sc.PolicyScore = 1.0
		}

		// Step 2: Capability Evidence Tier
		switch c.EvidenceTier {
		case routing.TierProductionObserved:
			sc.QualityScore = 1.0
		case routing.TierEvaluated:
			sc.QualityScore = 0.8
		default:
			sc.QualityScore = 0.5
		}
		if c.QualityScore > 0 {
			sc.QualityScore = (sc.QualityScore + c.QualityScore) / 2.0
		}

		// Step 3: Current Health
		if c.HealthStatus == "UNHEALTHY" || (c.ReliabilityProfile != nil && c.ReliabilityProfile.IncidentActive) {
			sc.Eligible = false
			sc.DisqualificationReason = "Agent is currently UNHEALTHY or under active INCIDENT"
			result.Candidates = append(result.Candidates, sc)
			continue
		}

		// Step 4: Tool Requirement Matching
		missingTools := make([]string, 0)
		toolMap := make(map[string]bool)
		for _, t := range c.SupportedTools {
			toolMap[t] = true
		}
		for _, rt := range req.RequiredTools {
			if !toolMap[rt] {
				missingTools = append(missingTools, rt)
			}
		}
		if len(missingTools) > 0 {
			sc.Eligible = false
			sc.DisqualificationReason = fmt.Sprintf("Missing required tools: %s", strings.Join(missingTools, ", "))
			result.Candidates = append(result.Candidates, sc)
			continue
		}

		// Step 5: Region & Data Residency
		if req.TargetRegion != "" && c.Region != "" && req.TargetRegion != c.Region {
			// Check if data classification restricts cross-region transfer
			if req.DataClassification == "RESTRICTED" || req.DataClassification == "SOVEREIGN" {
				sc.Eligible = false
				sc.DisqualificationReason = fmt.Sprintf("Data residency restriction: request in %s cannot route to agent in %s", req.TargetRegion, c.Region)
				result.Candidates = append(result.Candidates, sc)
				continue
			}
		}

		// Step 6: SLO Compliance & Error Budget
		sc.SLOScore = 1.0
		if c.SLO != nil {
			if c.SLO.CurrentStatus == slo.StatusBreached {
				sc.SLOScore = 0.2
			} else if c.SLO.CurrentStatus == slo.StatusAtRisk {
				sc.SLOScore = 0.6
			}
		}

		// Step 7: Reliability Score
		sc.ReliabilityScore = 0.70 // default cold start
		if c.ReliabilityProfile != nil && c.ReliabilityProfile.TotalSamples > 0 {
			sc.ReliabilityScore = c.ReliabilityProfile.OverallSuccessRate
		}

		// Step 8: Cost & Latency Scores
		sc.LatencyScore = 0.8
		sc.CostScore = 0.8
		if c.ReliabilityProfile != nil && c.ReliabilityProfile.TotalSamples > 0 {
			// Lower latency -> higher score
			if c.ReliabilityProfile.P95LatencyMs > 0 {
				latSec := float64(c.ReliabilityProfile.P95LatencyMs) / 1000.0
				sc.LatencyScore = 1.0 / (1.0 + latSec/2.0)
			}
			// Lower cost -> higher score
			sc.CostScore = 1.0 / (1.0 + c.ReliabilityProfile.AverageCostUSD*20.0)
		}

		// Compute Composite Multi-Objective Score
		sc.CompositeScore = r.computeCompositeScore(sc, objective)
		result.Candidates = append(result.Candidates, sc)
	}

	// Filter eligible candidates
	eligible := make([]*ScoredCandidate, 0)
	for _, c := range result.Candidates {
		if c.Eligible {
			eligible = append(eligible, c)
		}
	}

	if len(eligible) == 0 {
		return nil, errors.New("no eligible candidates satisfied policy, capability, tool, and health constraints")
	}

	// Step 9: Sort with Deterministic Tie-Break
	sort.Slice(eligible, func(i, j int) bool {
		if math.Abs(eligible[i].CompositeScore-eligible[j].CompositeScore) > 0.0001 {
			return eligible[i].CompositeScore > eligible[j].CompositeScore
		}
		// Deterministic tie-break by AgentID
		return eligible[i].Candidate.AgentID < eligible[j].Candidate.AgentID
	})

	topCandidate := eligible[0]

	// Apply Routing Hysteresis: prevent rapid flapping if previous incumbent is still healthy
	activeKey := tenantID + ":" + req.Capability
	if incumbentAgentID, ok := r.ActiveRoutes[activeKey]; ok && incumbentAgentID != topCandidate.Candidate.AgentID {
		// Find incumbent in eligible list
		for _, ec := range eligible {
			if ec.Candidate.AgentID == incumbentAgentID {
				// Only switch if topCandidate beats incumbent by at least HysteresisDelta
				if (topCandidate.CompositeScore - ec.CompositeScore) < r.HysteresisDelta {
					topCandidate = ec
				}
				break
			}
		}
	}

	r.ActiveRoutes[activeKey] = topCandidate.Candidate.AgentID

	result.SelectedAgentID = topCandidate.Candidate.AgentID
	result.SelectedVersion = topCandidate.Candidate.Version
	result.EndpointURL = topCandidate.Candidate.EndpointURL
	result.Confidence = topCandidate.CompositeScore
	result.DecisionExplanation = fmt.Sprintf(
		"Selected %s (score %.4f) under objective %s. Component scores: Rel=%.2f, Qual=%.2f, Lat=%.2f, Cost=%.2f, SLO=%.2f",
		topCandidate.Candidate.AgentID,
		topCandidate.CompositeScore,
		objective,
		topCandidate.ReliabilityScore,
		topCandidate.QualityScore,
		topCandidate.LatencyScore,
		topCandidate.CostScore,
		topCandidate.SLOScore,
	)

	return result, nil
}

// FailoverRoute selects the next best eligible candidate when the primary selected candidate fails.
func (r *BaselineRouterV1) FailoverRoute(previousResult *RouteResult, failedAgentID string) (*RouteResult, error) {
	if previousResult == nil || len(previousResult.Candidates) == 0 {
		return nil, errors.New("cannot failover from empty previous route result")
	}

	eligibleRemaining := make([]*ScoredCandidate, 0)
	for _, c := range previousResult.Candidates {
		if c.Eligible && c.Candidate.AgentID != failedAgentID {
			eligibleRemaining = append(eligibleRemaining, c)
		}
	}

	if len(eligibleRemaining) == 0 {
		return nil, fmt.Errorf("no fallback candidates eligible after failure of %s", failedAgentID)
	}

	sort.Slice(eligibleRemaining, func(i, j int) bool {
		return eligibleRemaining[i].CompositeScore > eligibleRemaining[j].CompositeScore
	})

	nextBest := eligibleRemaining[0]
	failoverResult := &RouteResult{
		SelectedAgentID:       nextBest.Candidate.AgentID,
		SelectedVersion:       nextBest.Candidate.Version,
		EndpointURL:           nextBest.Candidate.EndpointURL,
		Objective:             previousResult.Objective,
		AlgorithmID:           previousResult.AlgorithmID,
		AlgorithmVersion:      previousResult.AlgorithmVersion,
		PolicyVersion:         previousResult.PolicyVersion,
		Confidence:            nextBest.CompositeScore * 0.90, // slight discount for failover path
		IsFailover:            true,
		FailoverOriginalAgent: failedAgentID,
		Candidates:            previousResult.Candidates,
		DecidedAt:             time.Now().UTC(),
		DecisionExplanation:   fmt.Sprintf("FAILOVER: Primary candidate %s failed; rerouted to next eligible agent %s (score %.4f)", failedAgentID, nextBest.Candidate.AgentID, nextBest.CompositeScore),
	}

	return failoverResult, nil
}

func (r *BaselineRouterV1) computeCompositeScore(sc *ScoredCandidate, obj RoutingObjective) float64 {
	switch obj {
	case ObjectiveReliability:
		return 0.50*sc.ReliabilityScore + 0.20*sc.SLOScore + 0.15*sc.QualityScore + 0.10*sc.LatencyScore + 0.05*sc.CostScore
	case ObjectiveQuality:
		return 0.50*sc.QualityScore + 0.20*sc.ReliabilityScore + 0.15*sc.SLOScore + 0.10*sc.LatencyScore + 0.05*sc.CostScore
	case ObjectiveLatency:
		return 0.50*sc.LatencyScore + 0.20*sc.ReliabilityScore + 0.15*sc.SLOScore + 0.10*sc.QualityScore + 0.05*sc.CostScore
	case ObjectiveCost:
		return 0.50*sc.CostScore + 0.20*sc.ReliabilityScore + 0.15*sc.SLOScore + 0.10*sc.QualityScore + 0.05*sc.LatencyScore
	case ObjectivePolicyFirst:
		return 0.40*sc.PolicyScore + 0.30*sc.SLOScore + 0.15*sc.ReliabilityScore + 0.15*sc.QualityScore
	case ObjectiveBalanced:
		fallthrough
	default:
		return 0.30*sc.ReliabilityScore + 0.25*sc.QualityScore + 0.20*sc.LatencyScore + 0.15*sc.CostScore + 0.10*sc.SLOScore
	}
}
