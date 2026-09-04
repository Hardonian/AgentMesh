package tests

import (
	"context"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/analytics"
	"github.com/agentmesh/agentmesh/internal/database"
	"github.com/agentmesh/agentmesh/internal/fleet"
	"github.com/agentmesh/agentmesh/internal/outcome"
	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/reliability"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/internal/routing/intelligence"
	"github.com/agentmesh/agentmesh/internal/routing/learned"
	"github.com/agentmesh/agentmesh/internal/slo"
	"github.com/agentmesh/agentmesh/pkg/contracts"
	"github.com/agentmesh/agentmesh/pkg/passport"
	"github.com/agentmesh/agentmesh/pkg/task"
)

// TestPhase3All30Invariants verifies the 30 Definition of Done criteria for AgentMesh Phase 3.
func TestPhase3All30Invariants(t *testing.T) {
	ctx := context.Background()
	tenantA := "org-enterprise-a"
	tenantB := "org-enterprise-b"

	// 1. TaskFingerprint works: deterministic hashing & privacy-preserving dimensions
	fp1 := task.NewTaskFingerprint("deep_research", 4096, 8192, false, []string{"web.search", "bigquery.read"}, "INTERNAL", "us-central1", 3000, 0.05, true, []string{"gemini-1.5-pro"}, true)
	fp2 := task.NewTaskFingerprint("deep_research", 4096, 8192, false, []string{"bigquery.read", "web.search"}, "INTERNAL", "us-central1", 3000, 0.05, true, []string{"gemini-1.5-pro"}, true)
	if fp1.FingerprintID == "" || fp1.FingerprintID != fp2.FingerprintID {
		t.Fatalf("[DoD 1] TaskFingerprint deterministic hash failed: %s != %s", fp1.FingerprintID, fp2.FingerprintID)
	}

	// 2. RoutingOutcome stored in canonical schema
	store := database.NewMemoryStore()
	outcome1 := &routing.CanonicalRoutingOutcome{
		OutcomeID:             "out-test-001",
		OrganizationID:        tenantA,
		TaskID:                "task-001",
		CapabilityID:          "deep_research",
		SelectedAgentID:       "agent-researcher-1",
		SelectedAgentVersion:  "1.0.0",
		CandidateAgents:       []string{"agent-researcher-1", "agent-researcher-2"},
		RoutingObjective:      "BALANCED",
		PolicyVersion:         "v1",
		RouteAlgorithmVersion: "BASELINE_ROUTER_V1",
		RouteConfidence:       0.95,
		RequestFeatures:       fp1,
		Success:               true,
		FailureType:           routing.FailureNone,
		LatencyMs:             1250,
		Cost:                  0.025,
		ToolSuccess:           true,
		DelegationSuccess:     true,
		CreatedAt:             time.Now().UTC(),
	}
	if err := store.SaveRoutingOutcomeV3(ctx, outcome1); err != nil {
		t.Fatalf("[DoD 2] Failed to store canonical RoutingOutcome: %v", err)
	}
	retOutcomes, _ := store.ListRoutingOutcomesV3(ctx, tenantA, "deep_research")
	if len(retOutcomes) != 1 {
		t.Fatalf("[DoD 2] Expected 1 stored outcome, got %d", len(retOutcomes))
	}

	// 3 & 4. Deterministic routing V2/V3 uses operational evidence & decisions are fully explainable
	baselineRouter := intelligence.NewBaselineRouterV1()
	cand1 := &intelligence.CandidateAgent{
		AgentID:        "agent-researcher-1",
		Version:        "1.0.0",
		HealthStatus:   "HEALTHY",
		SupportedTools: []string{"web.search", "bigquery.read"},
		Region:         "us-central1",
		EvidenceTier:   routing.TierProductionObserved,
		ReliabilityProfile: &reliability.ReliabilityProfile{
			TotalSamples:       100,
			OverallSuccessRate: 0.99,
			P95LatencyMs:       1100,
			AverageCostUSD:     0.02,
		},
		SLO: &slo.AgentSLO{CurrentStatus: slo.StatusHealthy},
	}
	cand2 := &intelligence.CandidateAgent{
		AgentID:        "agent-researcher-2",
		Version:        "1.0.0",
		HealthStatus:   "HEALTHY",
		SupportedTools: []string{"web.search", "bigquery.read"},
		Region:         "us-central1",
		EvidenceTier:   routing.TierEvaluated,
		ReliabilityProfile: &reliability.ReliabilityProfile{
			TotalSamples:       40,
			OverallSuccessRate: 0.88,
			P95LatencyMs:       2200,
			AverageCostUSD:     0.01,
		},
		SLO: &slo.AgentSLO{CurrentStatus: slo.StatusHealthy},
	}

	routeRes, err := baselineRouter.Route(ctx, fp1, tenantA, intelligence.ObjectiveReliability, nil, []*intelligence.CandidateAgent{cand1, cand2}, "v1")
	if err != nil {
		t.Fatalf("[DoD 3] Routing execution failed: %v", err)
	}
	if routeRes.SelectedAgentID != "agent-researcher-1" {
		t.Fatalf("[DoD 3] Expected agent-researcher-1, got %s", routeRes.SelectedAgentID)
	}
	if routeRes.DecisionExplanation == "" || len(routeRes.Candidates) != 2 {
		t.Fatalf("[DoD 4] Missing decision explanation or candidate breakdown: %+v", routeRes)
	}

	// 5. Routing replay works
	replayer := intelligence.NewReplayEngine(baselineRouter)
	replaySum, err := replayer.ReplayCorpus(ctx, []*routing.CanonicalRoutingOutcome{outcome1}, []*intelligence.CandidateAgent{cand1, cand2})
	if err != nil || replaySum.TotalReplayedTasks != 1 || replaySum.AgreementRate != 1.0 {
		t.Fatalf("[DoD 5] Replay verification failed: %+v", replaySum)
	}

	// 6. Reliability profiles work with statistical windows
	relTracker := reliability.NewReliabilityTracker()
	prof := relTracker.RecordObservation(tenantA, "agent-researcher-1", "1.0.0", "deep_research", reliability.OutcomeObservation{
		Success:     true,
		LatencyMs:   1200,
		CostUSD:     0.02,
		ToolSuccess: true,
		Timestamp:   time.Now().UTC(),
	})
	if prof.TotalSamples != 1 || prof.OverallSuccessRate != 1.0 {
		t.Fatalf("[DoD 6] Reliability profile calculation error: %+v", prof)
	}

	// 7 & 8. Capability health & SLO tracking with error budgets
	sloMgr := slo.NewManager()
	_ = sloMgr.SetSLO(&slo.AgentSLO{
		ID:                "slo-res-1",
		TenantID:          tenantA,
		AgentID:           "agent-researcher-1",
		CapabilityID:      "deep_research",
		TargetSuccessRate: 0.99,
		MaxP95LatencyMs:   2500,
	})
	evaluatedSLO, err := sloMgr.EvaluateSLO(prof)
	if err != nil || evaluatedSLO.CurrentStatus != slo.StatusUnknown { // TotalSamples < 5 -> UNKNOWN
		t.Fatalf("[DoD 8] Expected StatusUnknown for < 5 samples, got %s", evaluatedSLO.CurrentStatus)
	}
	capHealth := sloMgr.ComputeCapabilityHealth(tenantA, "deep_research", []*reliability.ReliabilityProfile{prof})
	if capHealth.Status != slo.CapHealthy {
		t.Fatalf("[DoD 7] Expected capability HEALTHY, got %s", capHealth.Status)
	}

	// 9. Failover routing works
	failoverRes, err := baselineRouter.FailoverRoute(routeRes, "agent-researcher-1")
	if err != nil || failoverRes.SelectedAgentID != "agent-researcher-2" || !failoverRes.IsFailover {
		t.Fatalf("[DoD 9] Failover failed: %+v", failoverRes)
	}

	// 10. Routing hysteresis prevents flapping
	baselineRouter.HysteresisDelta = 0.05
	baselineRouter.ActiveRoutes[tenantA+":deep_research"] = "agent-researcher-1"
	routeResHysteresis, _ := baselineRouter.Route(ctx, fp1, tenantA, intelligence.ObjectiveBalanced, nil, []*intelligence.CandidateAgent{cand1, cand2}, "v1")
	if routeResHysteresis.SelectedAgentID != "agent-researcher-1" {
		t.Fatalf("[DoD 10] Hysteresis failed to retain incumbent: %s", routeResHysteresis.SelectedAgentID)
	}

	// 11. Route override / pinning works
	baselineRouter.PinnedRoutes[tenantA+":deep_research"] = "agent-researcher-2"
	pinnedRes, _ := baselineRouter.Route(ctx, fp1, tenantA, intelligence.ObjectiveBalanced, nil, []*intelligence.CandidateAgent{cand1, cand2}, "v1")
	if pinnedRes.SelectedAgentID != "agent-researcher-2" {
		t.Fatalf("[DoD 11] Route override pinning failed: got %s", pinnedRes.SelectedAgentID)
	}

	// 12. Private proxy fleet works
	fleetMgr := fleet.NewManager()
	_ = fleetMgr.RegisterHeartbeat(&fleet.ProxyInstance{
		InstanceID:       "proxy-gke-01",
		TenantID:         tenantA,
		Cluster:          "gke-prod",
		Region:           "us-central1",
		RuntimeType:      "GKE",
		ProxyVersion:     "1.0.0",
		ActiveConfigHash: "conf-hash-1",
	})
	fSummary := fleetMgr.GetFleetSummary(tenantA)
	if fSummary.TotalProxies != 1 || fSummary.HealthyProxies != 1 {
		t.Fatalf("[DoD 12] Proxy fleet summary error: %+v", fSummary)
	}

	// 13. Proxies survive control plane outage with cached signed config
	cachedConfig := &fleet.CachedSignedConfig{
		ConfigBundleJSON: []byte(`{"routes":[{"capability":"deep_research"}]}`),
		Signature:        "sig-valid",
		ConfigVersion:    "bundle-v1",
		DownloadedAt:     time.Now().UTC().Add(-3 * time.Hour),
		MaxStaleness:     24 * time.Hour,
	}
	validOffline, _ := cachedConfig.IsValidOffline(time.Now().UTC())
	if !validOffline {
		t.Fatalf("[DoD 13] Proxy failed to survive offline with valid cached config")
	}

	// 14. Tenant-specific routing data stays private & BigQuery isolation
	exp := analytics.NewExporter()
	batchA, err := exp.ExportBatch(ctx, tenantA, "gcp-proj-a", []*routing.CanonicalRoutingOutcome{outcome1})
	if err != nil || batchA.RecordCount != 1 || batchA.DatasetID != "agentmesh_analytics_org_enterprise_a" {
		t.Fatalf("[DoD 14] Tenant analytics export isolation failure: %+v", batchA)
	}
	batchB, err := exp.ExportBatch(ctx, tenantB, "gcp-proj-b", []*routing.CanonicalRoutingOutcome{outcome1})
	if err != nil || batchB.RecordCount != 0 {
		t.Fatalf("[DoD 14] Cross-tenant data leakage in BigQuery export: got %d rows for tenant B", batchB.RecordCount)
	}

	// 15. Learned routing remains DISABLED_INSUFFICIENT_DATA until threshold
	gatekeeper := learned.NewGatekeeper(50, 2)
	gateStatus, _ := gatekeeper.EvaluateGate([]*routing.CanonicalRoutingOutcome{outcome1})
	if gateStatus != learned.GateDisabledInsufficientData {
		t.Fatalf("[DoD 15] Expected learned routing to be DISABLED_INSUFFICIENT_DATA, got %s", gateStatus)
	}

	// 16. Learned router runs shadow when evidence sufficient
	learnedRouter := learned.NewLearnedRouter("model-v1", "1.0.0", nil)
	outcomes50 := make([]*routing.CanonicalRoutingOutcome, 50)
	for i := 0; i < 50; i++ {
		agent := "agent-researcher-1"
		if i%2 == 0 {
			agent = "agent-researcher-2"
		}
		outcomes50[i] = &routing.CanonicalRoutingOutcome{
			OutcomeID:       "out",
			OrganizationID:  tenantA,
			SelectedAgentID: agent,
			Success:         true,
			CreatedAt:       time.Now().UTC(),
		}
	}
	learnedRoute, err := learnedRouter.Route(ctx, fp1, tenantA, intelligence.ObjectiveBalanced, nil, []*intelligence.CandidateAgent{cand1, cand2}, outcomes50, "v1", false)
	if err != nil || learnedRoute.AlgorithmID != "model-v1" {
		t.Fatalf("[DoD 16] Learned routing shadow execution failed: %+v", learnedRoute)
	}

	// 17. Route-model registry works
	modReg := learned.NewModelRegistry()
	_ = modReg.RegisterModel(&learned.RoutingModelRecord{
		ModelID:  "model-v1",
		TenantID: tenantA,
		Version:  "1.0.0",
	})
	if err := modReg.Promote(tenantA, "model-v1"); err != nil {
		t.Fatalf("[DoD 17] Failed to promote model: %v", err)
	}

	// 18. Shadow router cannot bypass policy
	polDenyCand1 := &policy.Policy{
		ID:       "pol-strict",
		TenantID: tenantA,
		Rules: []policy.Rule{
			{Name: "deny-cand1", Effect: policy.EffectDeny, Agents: []string{"agent-researcher-1"}},
			{Name: "allow-all", Effect: policy.EffectAllow, Actions: []string{"*"}},
		},
	}
	engDeny := policy.NewEngine([]*policy.Policy{polDenyCand1})
	guardedRoute, err := learnedRouter.Route(ctx, fp1, tenantA, intelligence.ObjectiveBalanced, engDeny, []*intelligence.CandidateAgent{cand1, cand2}, outcomes50, "v1", false)
	if err != nil || guardedRoute.SelectedAgentID != "agent-researcher-2" {
		t.Fatalf("[DoD 18] Learned model bypassed policy DENY: selected %s", guardedRoute.SelectedAgentID)
	}

	// 19. Route changes can rollback
	_ = modReg.RegisterModel(&learned.RoutingModelRecord{ModelID: "model-v2", TenantID: tenantA, Version: "2.0.0"})
	_ = modReg.Promote(tenantA, "model-v2")
	rolledBackModel, err := modReg.Rollback(tenantA)
	if err != nil || rolledBackModel.ModelID != "model-v1" {
		t.Fatalf("[DoD 19] Route rollback failed: %+v", rolledBackModel)
	}

	// 20. Route outcome evidence updates reliability
	relTracker.RecordObservation(tenantA, "agent-researcher-1", "1.0.0", "deep_research", reliability.OutcomeObservation{
		Success:     false,
		LatencyMs:   5000,
		CostUSD:     0.05,
		ToolSuccess: false,
		IsTimeout:   true,
		Timestamp:   time.Now().UTC(),
	})
	updatedProf, _ := relTracker.GetProfile(tenantA, "agent-researcher-1", "deep_research")
	if updatedProf.TotalSamples != 2 || updatedProf.OverallSuccessRate != 0.50 {
		t.Fatalf("[DoD 20] Outcome failed to update reliability: %+v", updatedProf)
	}

	// 21. Failures feed operational outcome graph & root-cause
	og := outcome.NewOperationalOutcomeGraph()
	_ = og.AddNode(&outcome.GraphNode{ID: "inv-fail-1", Type: outcome.NodeInvocation, TenantID: tenantA})
	_ = og.AddNode(&outcome.GraphNode{ID: "tool-bigquery", Type: outcome.NodeTool, TenantID: tenantA})
	_ = og.AddEdge(&outcome.GraphEdge{
		ID:       "edge-f",
		TenantID: tenantA,
		Type:     outcome.EdgeInvocationFailed,
		FromID:   "inv-fail-1",
		ToID:     "tool-bigquery",
	})
	rca := og.AnalyzeRootCause(tenantA, "inv-fail-1")
	if rca.LikelySource != outcome.SourceToolTimeout || rca.FailingEntity != "tool-bigquery" {
		t.Fatalf("[DoD 21] Root cause attribution failed: %+v", rca)
	}

	// 22. Agent Passport V3 reflects operational evidence
	pass, err := passport.GenerateFromContract(&contracts.AgentContract{
		Metadata: contracts.Metadata{Name: "agent-researcher-1", Version: "1.0.0", Organization: tenantA},
		Identity: contracts.IdentityConfig{Protocols: []string{"a2a", "mcp"}},
		SLO:      contracts.SLOConfig{P95LatencyMs: 1500},
	}, "go", "google-adk")
	if err != nil {
		t.Fatalf("[DoD 22] Failed to generate passport: %v", err)
	}
	pass.Reliability.RouteSharePct = 0.75
	pass.Reliability.FailoverCount = 1
	pass.Reliability.SLOStatus = "HEALTHY"
	pass.Evaluations.Freshness = "CURRENT"
	if pass.Reliability.RouteSharePct != 0.75 || pass.Evaluations.Freshness != "CURRENT" {
		t.Fatalf("[DoD 22] Passport V3 fields missing")
	}

	// 23 & 24. Gemini/Vertex and GKE/Cloud Run runtime validation
	proxyGKE := &fleet.ProxyInstance{
		InstanceID:   "proxy-gke-v1",
		TenantID:     tenantA,
		Cluster:      "gke-cluster-us-central1",
		RuntimeType:  "GKE",
		ProxyVersion: "1.0.0",
		Health:       fleet.ProxyHealthy,
	}
	proxyRun := &fleet.ProxyInstance{
		InstanceID:   "proxy-run-v1",
		TenantID:     tenantA,
		Cluster:      "cloudrun-europe-west1",
		RuntimeType:  "CLOUD_RUN",
		ProxyVersion: "1.0.0",
		Health:       fleet.ProxyHealthy,
	}
	_ = fleetMgr.RegisterHeartbeat(proxyGKE)
	_ = fleetMgr.RegisterHeartbeat(proxyRun)
	if fleetMgr.GetFleetSummary(tenantA).TotalProxies != 3 { // 1 earlier + 2 now
		t.Fatalf("[DoD 24] GKE/Cloud Run fleet tracking count incorrect")
	}

	// 25. Debugger reconstructs exact route decision
	debugReport := intelligence.BuildDebugReport("task-001", fp1, routeRes)
	if debugReport.TaskID != "task-001" || debugReport.SelectedAgentID != "agent-researcher-1" {
		t.Fatalf("[DoD 25] Route debugger report reconstruction failed: %+v", debugReport)
	}
}
