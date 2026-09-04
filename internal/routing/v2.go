package routing

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/agentmesh/agentmesh/internal/policy"
)

// CapabilityEvidenceTier distinguishes declared claims from empirical proof.
type CapabilityEvidenceTier string

const (
	TierDeclared           CapabilityEvidenceTier = "DECLARED_CAPABILITY"
	TierEvaluated          CapabilityEvidenceTier = "EVALUATED_CAPABILITY"
	TierProductionObserved CapabilityEvidenceTier = "PRODUCTION_OBSERVED_CAPABILITY"
)

// Capability defines a registered, stable capability schema within an organization.
type Capability struct {
	ID                 string   `json:"id"`
	TenantID           string   `json:"tenantId"`
	Name               string   `json:"name"`
	Description        string   `json:"description,omitempty"`
	RequiredTools      []string `json:"requiredTools,omitempty"`
	AllowedDataClasses []string `json:"allowedDataClasses,omitempty"`
}

// RouteRequestV2 encapsulates rich multi-constraint task routing requirements.
type RouteRequestV2 struct {
	TenantID           string                 `json:"tenantId"`
	CallerAgentID      string                 `json:"callerAgentId"`
	RequiredCapability string                 `json:"requiredCapability"`
	RequiredTools      []string               `json:"requiredTools,omitempty"`
	DataClassification string                 `json:"dataClassification,omitempty"`
	AllowedModels      []string               `json:"allowedModels,omitempty"`
	TargetRegion       string                 `json:"targetRegion,omitempty"`
	CallerRegion       string                 `json:"callerRegion,omitempty"`
	AllowedRegions     []string               `json:"allowedRegions,omitempty"`
	Strategy           Strategy               `json:"strategy"`
	MaxLatencyMs       int64                  `json:"maxLatencyMs,omitempty"`
	MaxCostUSD         float64                `json:"maxCostUsd,omitempty"`
	MinEvidenceTier    CapabilityEvidenceTier `json:"minEvidenceTier,omitempty"`
}

// CandidateExplanationV2 gives an auditable reason for candidate inclusion or exclusion.
type CandidateExplanationV2 struct {
	AgentID                string                 `json:"agentId"`
	Version                string                 `json:"version"`
	Eligible               bool                   `json:"eligible"`
	DisqualificationReason string                 `json:"disqualificationReason,omitempty"`
	EvidenceTier           CapabilityEvidenceTier `json:"evidenceTier"`
	QualityScore           float64                `json:"qualityScore"`
	P95LatencyMs           int64                  `json:"p95LatencyMs"`
	CostUSD                float64                `json:"costUsd"`
	ReliabilityScore       float64                `json:"reliabilityScore"`
	CompositeScore         float64                `json:"compositeScore"`
}

// RouteDecisionV2 contains the final routing choice, confidence level, and auditable candidate breakdown.
type RouteDecisionV2 struct {
	SelectedAgentID            string                   `json:"selectedAgentId"`
	SelectedVersion            string                   `json:"selectedVersion"`
	EndpointURL                string                   `json:"endpointUrl"`
	Strategy                   Strategy                 `json:"strategy"`
	Score                      float64                  `json:"score"`
	Confidence                 float64                  `json:"confidence"` // 0.0 to 1.0
	ConfidenceAlgorithmVersion string                   `json:"confidenceAlgorithmVersion"`
	EvidenceTier               CapabilityEvidenceTier   `json:"evidenceTier"`
	Candidates                 []CandidateExplanationV2 `json:"candidates"`
	PolicyVersion              string                   `json:"policyVersion"`
	DecidedAt                  time.Time                `json:"decidedAt"`
}

// RouteOutcome captures runtime results for offline regret analysis and routing intelligence.
type RouteOutcome struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenantId"`
	Capability    string    `json:"capability"`
	SelectedAgent string    `json:"selectedAgent"`
	SelectedVer   string    `json:"selectedVer"`
	Success       bool      `json:"success"`
	LatencyMs     int64     `json:"latencyMs"`
	CostUSD       float64   `json:"costUsd"`
	Timestamp     time.Time `json:"timestamp"`
}

// RegretReport records whether another eligible candidate would have yielded a measurably better result.
type RegretReport struct {
	OutcomeID     string  `json:"outcomeId"`
	SelectedAgent string  `json:"selectedAgent"`
	OptimalAgent  string  `json:"optimalAgent"`
	RegretScore   float64 `json:"regretScore"` // 0.0 = zero regret (optimal choice)
	Reason        string  `json:"reason"`
}

// RouteV2 executes capability-aware multi-stage routing across eligible candidates.
func (r *Router) RouteV2(ctx context.Context, req *RouteRequestV2) (*RouteDecisionV2, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if req.RequiredCapability == "" {
		return nil, errors.New("requiredCapability cannot be empty")
	}
	if req.Strategy == "" {
		req.Strategy = StrategyBalanced
	}

	decision := &RouteDecisionV2{
		Strategy:                   req.Strategy,
		ConfidenceAlgorithmVersion: "v2.0-evidence-weighted",
		Candidates:                 make([]CandidateExplanationV2, 0),
		PolicyVersion:              "1.0.0",
		DecidedAt:                  time.Now().UTC(),
	}

	var eligibleCandidates []AgentRouteCandidate
	var candidateExps []CandidateExplanationV2

	for _, candPtr := range r.candidates {
		cand := *candPtr
		exp := CandidateExplanationV2{
			AgentID:          cand.AgentID,
			Eligible:         true,
			EvidenceTier:     TierDeclared,
			QualityScore:     cand.QualityScore,
			P95LatencyMs:     cand.P95LatencyMs,
			CostUSD:          cand.AverageCost,
			ReliabilityScore: cand.SuccessRate,
		}
		if cand.Contract != nil {
			exp.Version = cand.Contract.Metadata.Version
		}

		// Check Evidence Tier from Passport if present
		if cand.Passport != nil {
			if cand.Passport.Reliability.SampleCount >= 20 {
				exp.EvidenceTier = TierProductionObserved
			} else if cand.Passport.Reliability.SampleCount >= 5 {
				exp.EvidenceTier = TierEvaluated
			}
		}

		// 1. Availability Filter
		if cand.Status == "DISABLED" || cand.Status == "UNHEALTHY" {
			exp.Eligible = false
			exp.DisqualificationReason = fmt.Sprintf("status is %s", cand.Status)
			candidateExps = append(candidateExps, exp)
			continue
		}

		// 2. Capability Matching
		hasCap := false
		if cand.Contract != nil {
			for _, capName := range cand.Contract.Capabilities {
				if strings.EqualFold(capName, req.RequiredCapability) {
					hasCap = true
					break
				}
			}
		}
		if !hasCap {
			exp.Eligible = false
			exp.DisqualificationReason = fmt.Sprintf("capability %q not advertised in contract", req.RequiredCapability)
			candidateExps = append(candidateExps, exp)
			continue
		}

		// 3. Tool Requirements Check
		if len(req.RequiredTools) > 0 && cand.Contract != nil {
			missingTool := ""
			for _, reqTool := range req.RequiredTools {
				toolFound := false
				for _, allowed := range cand.Contract.Tools.Allow {
					if allowed == "*" || allowed == reqTool {
						toolFound = true
						break
					}
				}
				if !toolFound {
					missingTool = reqTool
					break
				}
			}
			if missingTool != "" {
				exp.Eligible = false
				exp.DisqualificationReason = fmt.Sprintf("tool %q unavailable in agent contract", missingTool)
				candidateExps = append(candidateExps, exp)
				continue
			}
		}

		// 4. Data Classification Policy Check
		if req.DataClassification != "" && cand.Contract != nil {
			if req.DataClassification == "RESTRICTED" {
				// Verify if candidate is restricted
				for _, denied := range cand.Contract.Tools.Deny {
					if denied == "RESTRICTED" {
						exp.Eligible = false
						exp.DisqualificationReason = "policy forbids RESTRICTED data access"
						break
					}
				}
				if !exp.Eligible {
					candidateExps = append(candidateExps, exp)
					continue
				}
			}
		}

		// 5. Budget Filter
		if req.MaxCostUSD > 0 && cand.AverageCost > req.MaxCostUSD {
			exp.Eligible = false
			exp.DisqualificationReason = fmt.Sprintf("cost $%.4f exceeds max task budget $%.4f", cand.AverageCost, req.MaxCostUSD)
			candidateExps = append(candidateExps, exp)
			continue
		}

		// 6. Latency SLO Filter
		if req.MaxLatencyMs > 0 && cand.P95LatencyMs > req.MaxLatencyMs {
			exp.Eligible = false
			exp.DisqualificationReason = fmt.Sprintf("P95 latency %dms exceeds max latency %dms", cand.P95LatencyMs, req.MaxLatencyMs)
			candidateExps = append(candidateExps, exp)
			continue
		}

		// 7. Data Residency Filter
		if len(req.AllowedRegions) > 0 {
			if ok, reason := ValidateDataResidency(cand.Region, req.AllowedRegions); !ok {
				exp.Eligible = false
				exp.DisqualificationReason = reason
				candidateExps = append(candidateExps, exp)
				continue
			}
		}

		// 8. Policy Engine Pre-Evaluation Check
		if r.policyEngine != nil {
			evalDec := r.policyEngine.Evaluate(ctx, &policy.EvaluationRequest{
				TenantID:       req.TenantID,
				SubjectAgentID: req.CallerAgentID,
				Capability:     req.RequiredCapability,
				Action:         "route",
				Resource:       cand.AgentID,
			})
			if evalDec.Effect == policy.EffectDeny {
				exp.Eligible = false
				exp.DisqualificationReason = fmt.Sprintf("policy denied invocation: %s", evalDec.Reason)
				candidateExps = append(candidateExps, exp)
				continue
			}
		}

		// Calculate composite score based on strategy with regional latency penalty
		scoredCandidate := cand
		if req.CallerRegion != "" && cand.Region != "" {
			penalty := EstimateRegionalLatencyPenalty(req.CallerRegion, cand.Region)
			scoredCandidate.P95LatencyMs += penalty
			exp.P95LatencyMs = scoredCandidate.P95LatencyMs
		}

		exp.CompositeScore = calculateStrategyScore(scoredCandidate, req.Strategy)
		eligibleCandidates = append(eligibleCandidates, scoredCandidate)
		candidateExps = append(candidateExps, exp)
	}

	decision.Candidates = candidateExps

	if len(eligibleCandidates) == 0 {
		return decision, fmt.Errorf("no eligible agent candidates found for capability %q", req.RequiredCapability)
	}

	// Sort eligible by composite score descending
	sort.Slice(candidateExps, func(i, j int) bool {
		if candidateExps[i].Eligible != candidateExps[j].Eligible {
			return candidateExps[i].Eligible
		}
		return candidateExps[i].CompositeScore > candidateExps[j].CompositeScore
	})

	winner := candidateExps[0]
	decision.SelectedAgentID = winner.AgentID
	decision.SelectedVersion = winner.Version
	decision.Score = winner.CompositeScore
	decision.EvidenceTier = winner.EvidenceTier

	// Calculate route confidence (samples + tier bonus)
	sampleBonus := 0.2
	switch winner.EvidenceTier {
	case TierProductionObserved:
		sampleBonus = 0.5
	case TierEvaluated:
		sampleBonus = 0.3
	}
	decision.Confidence = math.Min(1.0, 0.4+sampleBonus+(winner.ReliabilityScore*0.1))

	// Find endpoint URL
	for _, cand := range eligibleCandidates {
		if cand.AgentID == winner.AgentID {
			decision.EndpointURL = cand.EndpointURL
			break
		}
	}

	return decision, nil
}

func calculateStrategyScore(cand AgentRouteCandidate, strat Strategy) float64 {
	rel := cand.SuccessRate
	if rel == 0 {
		rel = 0.5
	}
	qual := cand.QualityScore
	if qual == 0 {
		qual = 0.5
	}

	costScore := 1.0
	if cand.AverageCost > 0 {
		costScore = 1.0 / (1.0 + cand.AverageCost*10.0)
	}

	latScore := 1.0
	if cand.P95LatencyMs > 0 {
		latScore = 1.0 / (1.0 + float64(cand.P95LatencyMs)/1000.0)
	}

	switch strat {
	case StrategyLowestCost:
		return costScore*0.7 + rel*0.3
	case StrategyLowestLatency:
		return latScore*0.7 + rel*0.3
	case StrategyHighestReliability:
		return rel*0.8 + qual*0.2
	case StrategyHighestQuality:
		return qual*0.8 + rel*0.2
	case StrategyBalanced:
		fallthrough
	default:
		return (rel * 0.35) + (qual * 0.25) + (latScore * 0.20) + (costScore * 0.20)
	}
}

// ComputeRegret evaluates if an alternative was measurably superior to the selected agent.
func ComputeRegret(outcome *RouteOutcome, eligibleCandidates []AgentRouteCandidate) *RegretReport {
	report := &RegretReport{
		OutcomeID:     outcome.ID,
		SelectedAgent: outcome.SelectedAgent,
		OptimalAgent:  outcome.SelectedAgent,
		RegretScore:   0.0,
		Reason:        "selected agent operated within optimal parameters",
	}

	if !outcome.Success {
		// If execution failed, search for a healthy alternative with >95% reliability
		for _, cand := range eligibleCandidates {
			if cand.AgentID != outcome.SelectedAgent && cand.SuccessRate > 0.95 {
				report.OptimalAgent = cand.AgentID
				report.RegretScore = 0.85
				report.Reason = fmt.Sprintf("selected agent failed task; candidate %s has 95%%+ success history", cand.AgentID)
				return report
			}
		}
	}

	return report
}
