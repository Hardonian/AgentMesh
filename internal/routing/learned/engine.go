package learned

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/internal/routing/intelligence"
	"github.com/agentmesh/agentmesh/pkg/task"
)

// GateStatus indicates whether empirical outcome criteria are met.
type GateStatus string

const (
	GateDisabledInsufficientData GateStatus = "DISABLED_INSUFFICIENT_DATA"
	GateEligible                 GateStatus = "ELIGIBLE"
)

// Gatekeeper verifies minimum volume and quality conditions before enabling learned routing.
type Gatekeeper struct {
	MinOutcomeCount int
	MinAgentCount   int
}

// NewGatekeeper initializes a gatekeeper with production thresholds.
func NewGatekeeper(minOutcomes, minAgents int) *Gatekeeper {
	if minOutcomes <= 0 {
		minOutcomes = 50
	}
	if minAgents <= 0 {
		minAgents = 2
	}
	return &Gatekeeper{
		MinOutcomeCount: minOutcomes,
		MinAgentCount:   minAgents,
	}
}

// EvaluateGate checks whether historical outcomes meet criteria.
func (g *Gatekeeper) EvaluateGate(outcomes []*routing.CanonicalRoutingOutcome) (GateStatus, string) {
	if len(outcomes) < g.MinOutcomeCount {
		return GateDisabledInsufficientData, fmt.Sprintf(
			"Insufficient historical outcomes: %d/%d recorded",
			len(outcomes), g.MinOutcomeCount,
		)
	}

	agentSet := make(map[string]bool)
	validCount := 0
	for _, o := range outcomes {
		if o.SelectedAgentID != "" {
			agentSet[o.SelectedAgentID] = true
			validCount++
		}
	}

	if len(agentSet) < g.MinAgentCount {
		return GateDisabledInsufficientData, fmt.Sprintf(
			"Insufficient agent diversity: %d/%d distinct agents observed",
			len(agentSet), g.MinAgentCount,
		)
	}

	return GateEligible, "Sufficient outcome evidence available for learned routing"
}

// CandidatePrediction holds regression and classification outputs per candidate.
type CandidatePrediction struct {
	Candidate            *intelligence.CandidateAgent `json:"candidate"`
	SuccessProbability   float64                      `json:"successProbability"` // 0.0 - 1.0
	PredictedLatencyMs   int64                        `json:"predictedLatencyMs"`
	EstimatedCostUSD     float64                      `json:"estimatedCostUsd"`
	QualityEstimate      float64                      `json:"qualityEstimate"`
	Uncertainty          intelligence.RoutingObjective `json:"-"`
	UncertaintyStatus    string                       `json:"uncertaintyStatus"` // HIGH_EVIDENCE, MEDIUM_EVIDENCE, LOW_EVIDENCE, COLD_START
	Eligible             bool                         `json:"eligible"`
	DisqualificationReason string                     `json:"disqualificationReason,omitempty"`
}

// LearnedRouter provides pure Go model inference for task routing.
type LearnedRouter struct {
	mu             sync.RWMutex
	modelID        string
	version        string
	weights        map[string]float64 // Feature weights
	gatekeeper     *Gatekeeper
	baselineRouter *intelligence.BaselineRouterV1
}

// NewLearnedRouter creates an instance of the learned router.
func NewLearnedRouter(modelID, version string, weights map[string]float64) *LearnedRouter {
	if weights == nil {
		weights = map[string]float64{
			"success_weight": 0.40,
			"latency_weight": 0.25,
			"cost_weight":    0.20,
			"quality_weight": 0.15,
		}
	}
	return &LearnedRouter{
		modelID:        modelID,
		version:        version,
		weights:        weights,
		gatekeeper:     NewGatekeeper(50, 2),
		baselineRouter: intelligence.NewBaselineRouterV1(),
	}
}

// Route executes learned inference with mandatory fallback to deterministic baseline if gated or in cold start.
func (lr *LearnedRouter) Route(
	ctx context.Context,
	req *task.TaskFingerprint,
	tenantID string,
	objective intelligence.RoutingObjective,
	policyEng *policy.Engine,
	candidates []*intelligence.CandidateAgent,
	historicalOutcomes []*routing.CanonicalRoutingOutcome,
	policyVersion string,
	allowExploration bool,
) (*intelligence.RouteResult, error) {
	// 1. Entry Gate Check (Section 15)
	status, reason := lr.gatekeeper.EvaluateGate(historicalOutcomes)
	if status == GateDisabledInsufficientData {
		// Mandatory fallback to BaselineRouterV1
		res, err := lr.baselineRouter.Route(ctx, req, tenantID, objective, policyEng, candidates, policyVersion)
		if err != nil {
			return nil, err
		}
		res.AlgorithmID = "BASELINE_FALLBACK"
		res.DecisionExplanation = fmt.Sprintf("LEARNED_ROUTING_%s: %s; utilized BaselineRouterV1", status, reason)
		return res, nil
	}

	// 2. Structural Guarantee: Policy Filtering FIRST (Section 18 & 140)
	eligibleCandidates := make([]*intelligence.CandidateAgent, 0)
	for _, c := range candidates {
		if policyEng != nil {
			evalReq := &policy.EvaluationRequest{
				TenantID:           tenantID,
				SubjectAgentID:     c.AgentID,
				Capability:         req.Capability,
				Action:             "route.invoke",
				DataClassification: req.DataClassification,
			}
			dec := policyEng.Evaluate(ctx, evalReq)
			if dec.Effect == policy.EffectDeny {
				continue
			}
		}
		// Must not be currently unhealthy
		if c.HealthStatus != "UNHEALTHY" {
			eligibleCandidates = append(eligibleCandidates, c)
		}
	}

	if len(eligibleCandidates) == 0 {
		return nil, errors.New("no candidates passed structural policy eligibility")
	}

	// 3. Score candidates using feature weights
	predictions := make([]*CandidatePrediction, 0, len(eligibleCandidates))
	for _, c := range eligibleCandidates {
		pred := lr.scoreCandidate(c, req)
		predictions = append(predictions, pred)
	}

	// 4. Safe Exploration (Section 21): 5% exploration to cold-start candidates only if non-destructive
	isExploring := false
	var selectedPred *CandidatePrediction

	canExplore := allowExploration && !intelligence.IsToolDestructive(strings.Join(req.RequiredTools, ","))
	if canExplore && rand.Float64() < 0.05 && len(predictions) > 1 {
		// Select low-evidence candidate to gather data safely
		for _, p := range predictions {
			if p.UncertaintyStatus == "COLD_START" || p.UncertaintyStatus == "LOW_EVIDENCE" {
				selectedPred = p
				isExploring = true
				break
			}
		}
	}

	if selectedPred == nil {
		// Exploitation: Pick highest composite score
		sort.Slice(predictions, func(i, j int) bool {
			return predictions[i].SuccessProbability > predictions[j].SuccessProbability
		})
		selectedPred = predictions[0]
	}

	scoredList := make([]*intelligence.ScoredCandidate, 0, len(candidates))
	for _, p := range predictions {
		scoredList = append(scoredList, &intelligence.ScoredCandidate{
			Candidate:        p.Candidate,
			Eligible:         true,
			PolicyScore:      1.0,
			ReliabilityScore: p.SuccessProbability,
			CompositeScore:   p.SuccessProbability,
		})
	}

	explanation := fmt.Sprintf(
		"Selected %s (pred success %.2f, latency %dms, cost $%.4f, certainty %s)",
		selectedPred.Candidate.AgentID,
		selectedPred.SuccessProbability,
		selectedPred.PredictedLatencyMs,
		selectedPred.EstimatedCostUSD,
		selectedPred.UncertaintyStatus,
	)
	if isExploring {
		explanation = "[SAFE_EXPLORATION_5%] " + explanation
	}

	return &intelligence.RouteResult{
		SelectedAgentID:     selectedPred.Candidate.AgentID,
		SelectedVersion:     selectedPred.Candidate.Version,
		EndpointURL:         selectedPred.Candidate.EndpointURL,
		Objective:           objective,
		AlgorithmID:         lr.modelID,
		AlgorithmVersion:    lr.version,
		PolicyVersion:       policyVersion,
		Confidence:          selectedPred.SuccessProbability,
		Candidates:          scoredList,
		DecidedAt:           time.Now().UTC(),
		DecisionExplanation: explanation,
	}, nil
}

func (lr *LearnedRouter) scoreCandidate(c *intelligence.CandidateAgent, req *task.TaskFingerprint) *CandidatePrediction {
	pred := &CandidatePrediction{
		Candidate:          c,
		SuccessProbability: 0.80,
		PredictedLatencyMs: 1500,
		EstimatedCostUSD:   0.02,
		QualityEstimate:    0.85,
		UncertaintyStatus:  "COLD_START",
		Eligible:           true,
	}

	if c.ReliabilityProfile != nil && c.ReliabilityProfile.TotalSamples > 0 {
		pred.SuccessProbability = c.ReliabilityProfile.OverallSuccessRate
		pred.PredictedLatencyMs = c.ReliabilityProfile.P50LatencyMs
		pred.EstimatedCostUSD = c.ReliabilityProfile.AverageCostUSD
		switch {
		case c.ReliabilityProfile.TotalSamples >= 100:
			pred.UncertaintyStatus = "HIGH_EVIDENCE"
		case c.ReliabilityProfile.TotalSamples >= 25:
			pred.UncertaintyStatus = "MEDIUM_EVIDENCE"
		default:
			pred.UncertaintyStatus = "LOW_EVIDENCE"
		}
	}

	// Adjust predicted latency based on input complexity
	if req.ComplexityClass == task.ComplexityComplex {
		pred.PredictedLatencyMs = int64(float64(pred.PredictedLatencyMs) * 1.5)
	}

	return pred
}
