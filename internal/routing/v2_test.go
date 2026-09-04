package routing_test

import (
	"context"
	"testing"

	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/pkg/contracts"
	"github.com/agentmesh/agentmesh/pkg/passport"
)

func TestRoutingV2FiltersAndRanking(t *testing.T) {
	cFinance := &contracts.AgentContract{
		Metadata:     contracts.Metadata{Name: "finance-agent", Version: "1.0.0"},
		Capabilities: []string{"invoice_analysis", "forecasting"},
		Tools:        contracts.ToolsConfig{Allow: []string{"bigquery.read", "sheets.write"}},
	}
	pFinance, _ := passport.GenerateFromContract(cFinance, "go", "google-adk")
	pFinance.RecordExecutionSample(true, 1500, 0.04, true)
	pFinance.RecordExecutionSample(true, 1400, 0.03, true)
	pFinance.RecordExecutionSample(true, 1600, 0.04, true)
	pFinance.RecordExecutionSample(true, 1500, 0.04, true)
	pFinance.RecordExecutionSample(true, 1450, 0.03, true) // 5 samples -> MEASURED / TierEvaluated

	cSlow := &contracts.AgentContract{
		Metadata:     contracts.Metadata{Name: "slow-agent", Version: "0.9.0"},
		Capabilities: []string{"invoice_analysis"},
		Tools:        contracts.ToolsConfig{Allow: []string{"bigquery.read"}},
	}

	cRestricted := &contracts.AgentContract{
		Metadata:     contracts.Metadata{Name: "restricted-agent", Version: "1.0.0"},
		Capabilities: []string{"invoice_analysis"},
		Tools: contracts.ToolsConfig{
			Allow: []string{"bigquery.read"},
			Deny:  []string{"RESTRICTED"},
		},
	}

	router := routing.NewRouter(nil)
	router.RegisterCandidate(&routing.AgentRouteCandidate{
		AgentID:      "finance-agent",
		EndpointURL:  "http://finance.mesh:8080",
		Status:       "HEALTHY",
		Contract:     cFinance,
		Passport:     pFinance,
		AverageCost:  0.035,
		P95LatencyMs: 1600,
		SuccessRate:  0.99,
		QualityScore: 0.96,
	})
	router.RegisterCandidate(&routing.AgentRouteCandidate{
		AgentID:      "slow-agent",
		EndpointURL:  "http://slow.mesh:8080",
		Status:       "HEALTHY",
		Contract:     cSlow,
		AverageCost:  0.01,
		P95LatencyMs: 8000, // 8 seconds latency
		SuccessRate:  0.85,
	})
	router.RegisterCandidate(&routing.AgentRouteCandidate{
		AgentID:      "restricted-agent",
		EndpointURL:  "http://restricted.mesh:8080",
		Status:       "HEALTHY",
		Contract:     cRestricted,
		AverageCost:  0.02,
		P95LatencyMs: 1200,
	})

	// Test 1: Max Latency Filter
	req := &routing.RouteRequestV2{
		TenantID:           "test_tenant",
		RequiredCapability: "invoice_analysis",
		MaxLatencyMs:       3000,
		Strategy:           routing.StrategyBalanced,
	}

	decision, err := router.RouteV2(context.Background(), req)
	if err != nil {
		t.Fatalf("expected successful route decision, got error: %v", err)
	}

	if decision.SelectedAgentID != "finance-agent" {
		t.Fatalf("expected finance-agent to be selected, got: %s", decision.SelectedAgentID)
	}

	// Verify slow-agent was disqualified due to latency
	foundSlow := false
	for _, c := range decision.Candidates {
		if c.AgentID == "slow-agent" {
			foundSlow = true
			if c.Eligible {
				t.Fatalf("expected slow-agent to be ineligible due to latency")
			}
			if c.DisqualificationReason == "" {
				t.Fatalf("expected disqualification reason for slow-agent")
			}
		}
	}
	if !foundSlow {
		t.Fatalf("slow-agent not present in candidate explanation list")
	}

	// Test 2: Tool requirement filter
	reqTool := &routing.RouteRequestV2{
		TenantID:           "test_tenant",
		RequiredCapability: "invoice_analysis",
		RequiredTools:      []string{"sheets.write"},
	}
	decTool, err := router.RouteV2(context.Background(), reqTool)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decTool.SelectedAgentID != "finance-agent" {
		t.Fatalf("expected finance-agent with sheets.write tool, got: %s", decTool.SelectedAgentID)
	}

	// Test 3: Data classification RESTRICTED filter
	reqRestricted := &routing.RouteRequestV2{
		TenantID:           "test_tenant",
		RequiredCapability: "invoice_analysis",
		DataClassification: "RESTRICTED",
	}
	decRestricted, err := router.RouteV2(context.Background(), reqRestricted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, cand := range decRestricted.Candidates {
		if cand.AgentID == "restricted-agent" && cand.Eligible {
			t.Fatalf("restricted-agent should be barred when data is RESTRICTED")
		}
	}
}

func TestRegretAnalysis(t *testing.T) {
	candidates := []routing.AgentRouteCandidate{
		{AgentID: "failing-agent", SuccessRate: 0.60},
		{AgentID: "backup-agent", SuccessRate: 0.99},
	}

	outcome := &routing.RouteOutcome{
		ID:            "out_123",
		TenantID:      "tenant",
		Capability:    "forecasting",
		SelectedAgent: "failing-agent",
		Success:       false,
	}

	report := routing.ComputeRegret(outcome, candidates)
	if report.RegretScore == 0.0 {
		t.Fatalf("expected non-zero regret score when selected agent failed while 99%% backup existed")
	}
	if report.OptimalAgent != "backup-agent" {
		t.Fatalf("expected backup-agent to be identified as optimal, got: %s", report.OptimalAgent)
	}
}
