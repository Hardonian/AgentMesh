package intelligence

import (
	"context"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/reliability"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/internal/slo"
	"github.com/agentmesh/agentmesh/pkg/task"
)

func TestBaselineRouterV1ObjectivesAndExclusions(t *testing.T) {
	router := NewBaselineRouterV1()
	ctx := context.Background()

	fp := task.NewTaskFingerprint(
		"financial_forecast",
		2048,
		4096,
		false,
		[]string{"bigquery.read"},
		"INTERNAL",
		"us-central1",
		3000,
		0.10,
		false,
		nil,
		true,
	)

	// Candidate A: Highly reliable, moderate cost
	candA := &CandidateAgent{
		AgentID:        "agent-a",
		Version:        "1.0.0",
		EndpointURL:    "http://agent-a:8080",
		HealthStatus:   "HEALTHY",
		SupportedTools: []string{"bigquery.read", "web.search"},
		Region:         "us-central1",
		EvidenceTier:   routing.TierProductionObserved,
		ReliabilityProfile: &reliability.ReliabilityProfile{
			TotalSamples:       150,
			OverallSuccessRate: 0.99,
			P95LatencyMs:       1200,
			AverageCostUSD:     0.03,
		},
		SLO: &slo.AgentSLO{CurrentStatus: slo.StatusHealthy},
	}

	// Candidate B: Low cost, lower reliability
	candB := &CandidateAgent{
		AgentID:        "agent-b",
		Version:        "1.0.0",
		EndpointURL:    "http://agent-b:8080",
		HealthStatus:   "HEALTHY",
		SupportedTools: []string{"bigquery.read"},
		Region:         "us-central1",
		EvidenceTier:   routing.TierEvaluated,
		ReliabilityProfile: &reliability.ReliabilityProfile{
			TotalSamples:       50,
			OverallSuccessRate: 0.88,
			P95LatencyMs:       1800,
			AverageCostUSD:     0.005, // Much cheaper
		},
		SLO: &slo.AgentSLO{CurrentStatus: slo.StatusHealthy},
	}

	// Candidate C: Missing required tool "bigquery.read"
	candC := &CandidateAgent{
		AgentID:        "agent-c",
		Version:        "1.0.0",
		EndpointURL:    "http://agent-c:8080",
		HealthStatus:   "HEALTHY",
		SupportedTools: []string{"web.search"},
		Region:         "us-central1",
	}

	candidates := []*CandidateAgent{candA, candB, candC}

	// Test RELIABILITY objective -> should pick candA
	resRel, err := router.Route(ctx, fp, "acme", ObjectiveReliability, nil, candidates, "v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resRel.SelectedAgentID != "agent-a" {
		t.Errorf("expected agent-a for ObjectiveReliability, got %s", resRel.SelectedAgentID)
	}

	// Test COST objective -> should pick candB
	resCost, err := router.Route(ctx, fp, "acme", ObjectiveCost, nil, candidates, "v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resCost.SelectedAgentID != "agent-b" {
		t.Errorf("expected agent-b for ObjectiveCost, got %s", resCost.SelectedAgentID)
	}

	// Verify Candidate C was excluded due to missing tools
	var candCExplanation *ScoredCandidate
	for _, c := range resRel.Candidates {
		if c.Candidate.AgentID == "agent-c" {
			candCExplanation = c
			break
		}
	}
	if candCExplanation == nil || candCExplanation.Eligible {
		t.Errorf("expected agent-c to be ineligible due to missing tools")
	}
}

func TestRoutingHysteresisAndFailover(t *testing.T) {
	router := NewBaselineRouterV1()
	router.HysteresisDelta = 0.10 // 10% lift needed to displace incumbent
	ctx := context.Background()

	fp := task.NewTaskFingerprint("deep_research", 1024, 1024, false, nil, "PUBLIC", "us-central1", 2000, 0.05, false, nil, false)

	candA := &CandidateAgent{
		AgentID:      "agent-a",
		HealthStatus: "HEALTHY",
		EvidenceTier: routing.TierEvaluated,
		ReliabilityProfile: &reliability.ReliabilityProfile{
			TotalSamples: 20, OverallSuccessRate: 0.90, P95LatencyMs: 1000, AverageCostUSD: 0.02,
		},
	}
	candB := &CandidateAgent{
		AgentID:      "agent-b",
		HealthStatus: "HEALTHY",
		EvidenceTier: routing.TierEvaluated,
		ReliabilityProfile: &reliability.ReliabilityProfile{
			TotalSamples: 20, OverallSuccessRate: 0.92, P95LatencyMs: 980, AverageCostUSD: 0.02, // Only slightly better (within 10% delta)
		},
	}

	// 1. First route selects agent-a
	candidates := []*CandidateAgent{candA}
	res1, _ := router.Route(ctx, fp, "acme", ObjectiveBalanced, nil, candidates, "v1")
	if res1.SelectedAgentID != "agent-a" {
		t.Fatalf("expected initial incumbent agent-a")
	}

	// 2. Add agent-b: even though agent-b has slightly higher score, hysteresis keeps incumbent agent-a
	candidates = []*CandidateAgent{candA, candB}
	res2, _ := router.Route(ctx, fp, "acme", ObjectiveBalanced, nil, candidates, "v1")
	if res2.SelectedAgentID != "agent-a" {
		t.Errorf("hysteresis failed: expected incumbent agent-a to be retained, got %s", res2.SelectedAgentID)
	}

	// 3. Failover: agent-a fails, failover chooses agent-b
	failoverRes, err := router.FailoverRoute(res2, "agent-a")
	if err != nil {
		t.Fatalf("failover failed: %v", err)
	}
	if failoverRes.SelectedAgentID != "agent-b" {
		t.Errorf("expected failover to agent-b, got %s", failoverRes.SelectedAgentID)
	}
	if !failoverRes.IsFailover {
		t.Errorf("expected IsFailover to be true")
	}
}

func TestShadowRoutingAndDebugger(t *testing.T) {
	router := NewBaselineRouterV1()
	ctx := context.Background()

	// Task with write tool -> shadow execution must be suppressed
	fpWrite := task.NewTaskFingerprint("payment", 100, 100, false, []string{"payment.execute"}, "CONFIDENTIAL", "us-central1", 1000, 0.01, false, nil, false)
	cand := &CandidateAgent{
		AgentID:        "pay-agent",
		HealthStatus:   "HEALTHY",
		SupportedTools: []string{"payment.execute"},
	}

	res, err := router.Route(ctx, fpWrite, "acme", ObjectiveBalanced, nil, []*CandidateAgent{cand}, "v1")
	if err != nil {
		t.Fatalf("routing failed: %v", err)
	}

	shadowRes, err := EvaluateShadowRoute(ctx, fpWrite, res, router, "acme", []*CandidateAgent{cand}, "v1")
	if err != nil {
		t.Fatalf("shadow eval failed: %v", err)
	}
	if !shadowRes.SideEffectsSuppressed {
		t.Errorf("expected side-effects to be suppressed for destructive payment tool")
	}

	// Test Debug Report
	debugReport := BuildDebugReport("task-999", fpWrite, res)
	if debugReport.TaskID != "task-999" || debugReport.SelectedAgentID != "pay-agent" {
		t.Errorf("unexpected debug report contents: %+v", debugReport)
	}
}

func TestOfflineRoutingReplay(t *testing.T) {
	router := NewBaselineRouterV1()
	replayer := NewReplayEngine(router)
	ctx := context.Background()

	fp := task.NewTaskFingerprint("search", 500, 500, false, nil, "PUBLIC", "us-central1", 2000, 0.05, false, nil, false)

	candA := &CandidateAgent{
		AgentID:      "agent-a",
		HealthStatus: "HEALTHY",
		ReliabilityProfile: &reliability.ReliabilityProfile{
			AverageCostUSD: 0.01,
			P50LatencyMs:   300,
		},
	}

	hist := []*routing.CanonicalRoutingOutcome{
		{
			OutcomeID:       "out-1",
			TaskID:          "task-1",
			OrganizationID:  "acme",
			CapabilityID:    "search",
			SelectedAgentID: "agent-a",
			Cost:            0.02,
			LatencyMs:       500,
			Success:         true,
			RequestFeatures: fp,
			CreatedAt:       time.Now().UTC(),
		},
	}

	summary, err := replayer.ReplayCorpus(ctx, hist, []*CandidateAgent{candA})
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}
	if summary.TotalReplayedTasks != 1 {
		t.Errorf("expected 1 replayed task, got %d", summary.TotalReplayedTasks)
	}
	if summary.AgreementRate != 1.0 {
		t.Errorf("expected agreement rate 1.0, got %f", summary.AgreementRate)
	}
}

func TestBaselineRouterPolicyDenyFiltering(t *testing.T) {
	router := NewBaselineRouterV1()
	ctx := context.Background()

	pol := &policy.Policy{
		ID:       "pol-1",
		TenantID: "acme",
		Rules: []policy.Rule{
			{
				Name:   "deny-restricted-agents",
				Effect: policy.EffectDeny,
				Agents: []string{"restricted-agent"},
			},
			{
				Name:    "allow-general-route",
				Effect:  policy.EffectAllow,
				Actions: []string{"*"},
			},
		},
	}
	eng := policy.NewEngine([]*policy.Policy{pol})

	fp := task.NewTaskFingerprint("search", 100, 100, false, nil, "PUBLIC", "us-central1", 1000, 0.01, false, nil, false)
	candAllowed := &CandidateAgent{AgentID: "allowed-agent", HealthStatus: "HEALTHY"}
	candDenied := &CandidateAgent{AgentID: "restricted-agent", HealthStatus: "HEALTHY"}

	res, err := router.Route(ctx, fp, "acme", ObjectiveBalanced, eng, []*CandidateAgent{candAllowed, candDenied}, "v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.SelectedAgentID != "allowed-agent" {
		t.Errorf("expected allowed-agent, got %s", res.SelectedAgentID)
	}
}
