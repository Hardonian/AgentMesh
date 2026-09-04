package routing

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/pkg/contracts"
	"github.com/agentmesh/agentmesh/pkg/passport"
)

type Strategy string

const (
	StrategyLowestCost         Strategy = "LOWEST_COST"
	StrategyLowestLatency      Strategy = "LOWEST_LATENCY"
	StrategyHighestReliability Strategy = "HIGHEST_RELIABILITY"
	StrategyHighestQuality     Strategy = "HIGHEST_QUALITY"
	StrategyBalanced           Strategy = "BALANCED"
)

// AgentRouteCandidate represents an agent being evaluated for routing.
type AgentRouteCandidate struct {
	AgentID      string
	EndpointURL  string
	Status       string // HEALTHY, DEGRADED, UNHEALTHY, DISABLED
	Contract     *contracts.AgentContract
	Passport     *passport.AgentPassport
	AverageCost  float64
	P95LatencyMs int64
	SuccessRate  float64
	QualityScore float64
	Region       string
}

// RouteRequest contains caller requirements for dispatching a task.
type RouteRequest struct {
	TenantID           string   `json:"tenantId"`
	CallerAgentID      string   `json:"callerAgentId"`
	RequiredCapability string   `json:"requiredCapability"`
	Strategy           Strategy `json:"strategy"`
	MaxLatencyMs       int64    `json:"maxLatencyMs,omitempty"`
	MaxCostUSD         float64  `json:"maxCostUsd,omitempty"`
	PreferredRegion    string   `json:"preferredRegion,omitempty"`
}

// CandidateExplanation details the eligibility decision for a candidate.
type CandidateExplanation struct {
	AgentID    string   `json:"agentId"`
	Eligible   bool     `json:"eligible"`
	Score      float64  `json:"score"`
	Reasons    []string `json:"reasons"`
	Exclusions []string `json:"exclusions,omitempty"`
}

// RouteDecision is the final deterministic routing choice and proof.
type RouteDecision struct {
	SelectedAgentID string                 `json:"selectedAgentId"`
	EndpointURL     string                 `json:"endpointUrl"`
	Strategy        Strategy               `json:"strategy"`
	SummaryReason   string                 `json:"summaryReason"`
	Timestamp       time.Time              `json:"timestamp"`
	Explanations    []CandidateExplanation `json:"explanations"`
}

// Router executes capability routing across registered agents.
type Router struct {
	mu           sync.RWMutex
	candidates   map[string]*AgentRouteCandidate
	policyEngine *policy.Engine
}

// NewRouter constructs a router.
func NewRouter(engine *policy.Engine) *Router {
	return &Router{
		candidates:   make(map[string]*AgentRouteCandidate),
		policyEngine: engine,
	}
}

// RegisterCandidate adds or updates a routing candidate.
func (r *Router) RegisterCandidate(cand *AgentRouteCandidate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.candidates[cand.AgentID] = cand
}

// RemoveCandidate removes an agent from routing.
func (r *Router) RemoveCandidate(agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.candidates, agentID)
}

// Route evaluates all candidates and returns the best eligible agent.
func (r *Router) Route(ctx context.Context, req *RouteRequest) (*RouteDecision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if req.Strategy == "" {
		req.Strategy = StrategyBalanced
	}

	var eligible []*AgentRouteCandidate
	var explanations []CandidateExplanation

	now := time.Now().UTC()

	for _, cand := range r.candidates {
		expl := CandidateExplanation{
			AgentID:  cand.AgentID,
			Eligible: true,
		}

		// 1. Availability check: Disabled or Unhealthy agents are excluded
		switch cand.Status {
		case "DISABLED":
			expl.Eligible = false
			expl.Exclusions = append(expl.Exclusions, "agent is DISABLED")
		case "UNHEALTHY":
			expl.Eligible = false
			expl.Exclusions = append(expl.Exclusions, "agent health status is UNHEALTHY")
		default:
			expl.Reasons = append(expl.Reasons, fmt.Sprintf("status %s", cand.Status))
		}

		// 2. Capability check
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
			expl.Eligible = false
			expl.Exclusions = append(expl.Exclusions, fmt.Sprintf("lacks capability %q", req.RequiredCapability))
		} else {
			expl.Reasons = append(expl.Reasons, fmt.Sprintf("supports capability %q", req.RequiredCapability))
		}

		// 3. Policy check
		if r.policyEngine != nil {
			polDecision := r.policyEngine.Evaluate(ctx, &policy.EvaluationRequest{
				TenantID:       req.TenantID,
				SubjectAgentID: req.CallerAgentID,
				Capability:     req.RequiredCapability,
				Action:         "invoke",
				Resource:       cand.AgentID,
			})
			if polDecision.Effect == policy.EffectDeny {
				expl.Eligible = false
				expl.Exclusions = append(expl.Exclusions, fmt.Sprintf("policy DENY: %s", polDecision.Reason))
			} else {
				expl.Reasons = append(expl.Reasons, "policy permits invocation")
			}
		}

		// 4. Budget constraints
		if req.MaxCostUSD > 0 && cand.AverageCost > req.MaxCostUSD {
			expl.Eligible = false
			expl.Exclusions = append(expl.Exclusions, fmt.Sprintf("average cost $%.4f exceeds ceiling $%.4f", cand.AverageCost, req.MaxCostUSD))
		} else if cand.AverageCost > 0 {
			expl.Reasons = append(expl.Reasons, fmt.Sprintf("cost $%.4f within budget", cand.AverageCost))
		}

		// 5. SLO constraints
		if req.MaxLatencyMs > 0 && cand.P95LatencyMs > req.MaxLatencyMs {
			expl.Eligible = false
			expl.Exclusions = append(expl.Exclusions, fmt.Sprintf("P95 latency %dms exceeds limit %dms", cand.P95LatencyMs, req.MaxLatencyMs))
		}

		// Compute score for ranking
		expl.Score = computeScore(cand, req.Strategy)

		if expl.Eligible {
			eligible = append(eligible, cand)
		}
		explanations = append(explanations, expl)
	}

	if len(eligible) == 0 {
		return &RouteDecision{
			Strategy:      req.Strategy,
			SummaryReason: "no eligible agent found satisfying capability, policy, health, and budget constraints",
			Timestamp:     now,
			Explanations:  explanations,
		}, errors.New("no eligible agent available")
	}

	// Sort eligible candidates deterministically by score descending, tie-break by AgentID
	sort.Slice(eligible, func(i, j int) bool {
		scoreI := computeScore(eligible[i], req.Strategy)
		scoreJ := computeScore(eligible[j], req.Strategy)
		if scoreI != scoreJ {
			return scoreI > scoreJ
		}
		return eligible[i].AgentID < eligible[j].AgentID // Deterministic tie break
	})

	selected := eligible[0]
	summary := fmt.Sprintf("selected %s based on %s strategy (success rate: %.1f%%, P95: %dms, avg cost: $%.4f)",
		selected.AgentID, req.Strategy, selected.SuccessRate*100, selected.P95LatencyMs, selected.AverageCost)

	return &RouteDecision{
		SelectedAgentID: selected.AgentID,
		EndpointURL:     selected.EndpointURL,
		Strategy:        req.Strategy,
		SummaryReason:   summary,
		Timestamp:       now,
		Explanations:    explanations,
	}, nil
}

func computeScore(c *AgentRouteCandidate, strategy Strategy) float64 {
	switch strategy {
	case StrategyLowestCost:
		// Lower cost gives higher score
		if c.AverageCost <= 0 {
			return 100.0
		}
		return 1.0 / c.AverageCost
	case StrategyLowestLatency:
		if c.P95LatencyMs <= 0 {
			return 1000.0
		}
		return 10000.0 / float64(c.P95LatencyMs)
	case StrategyHighestReliability:
		return c.SuccessRate * 100.0
	case StrategyHighestQuality:
		return c.QualityScore * 100.0
	case StrategyBalanced:
		fallthrough
	default:
		// Balanced composite: 40% reliability, 30% latency, 30% cost
		relPart := c.SuccessRate * 40.0
		latPart := 30.0
		if c.P95LatencyMs > 0 {
			latPart = 3000.0 / float64(c.P95LatencyMs)
			if latPart > 30.0 {
				latPart = 30.0
			}
		}
		costPart := 30.0
		if c.AverageCost > 0 {
			costPart = 3.0 / c.AverageCost
			if costPart > 30.0 {
				costPart = 30.0
			}
		}
		return relPart + latPart + costPart
	}
}
