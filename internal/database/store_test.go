package database

import (
	"context"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/fleet"
	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/reliability"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/internal/routing/learned"
	"github.com/agentmesh/agentmesh/internal/slo"
	"github.com/agentmesh/agentmesh/pkg/spec"
	"github.com/agentmesh/agentmesh/pkg/task"
)

func TestMemoryStorePhase3Entities(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	tenant := "acme-corp"

	// 1. TaskFingerprint
	fp := task.NewTaskFingerprint("forecast", 100, 100, false, nil, "INTERNAL", "us-central1", 1000, 0.05, false, nil, false)
	if err := store.SaveTaskFingerprint(ctx, tenant, fp); err != nil {
		t.Fatalf("failed to save task fingerprint: %v", err)
	}
	retFP, err := store.GetTaskFingerprint(ctx, tenant, fp.FingerprintID)
	if err != nil || retFP.Capability != "forecast" {
		t.Errorf("failed to retrieve task fingerprint: %v", err)
	}

	// 2. RoutingOutcomeV3
	outcome := &routing.CanonicalRoutingOutcome{
		OutcomeID:       "out-v3-1",
		OrganizationID:  tenant,
		TaskID:          "task-1",
		CapabilityID:    "forecast",
		SelectedAgentID: "agent-a",
		Success:         true,
		CreatedAt:       time.Now().UTC(),
	}
	if err := store.SaveRoutingOutcomeV3(ctx, outcome); err != nil {
		t.Fatalf("failed to save routing outcome: %v", err)
	}
	outcomes, err := store.ListRoutingOutcomesV3(ctx, tenant, "forecast")
	if err != nil || len(outcomes) != 1 {
		t.Errorf("expected 1 outcome, got %d", len(outcomes))
	}

	// 3. Reliability Profile
	prof := &reliability.ReliabilityProfile{
		AgentID:      "agent-a",
		CapabilityID: "forecast",
		TenantID:     tenant,
		TotalSamples: 10,
	}
	if err := store.SaveReliabilityProfile(ctx, prof); err != nil {
		t.Fatalf("failed to save reliability profile: %v", err)
	}
	retProf, err := store.GetReliabilityProfile(ctx, tenant, "agent-a", "forecast")
	if err != nil || retProf.TotalSamples != 10 {
		t.Errorf("failed to retrieve profile: %v", err)
	}

	// 4. AgentSLO
	s := &slo.AgentSLO{
		ID:           "slo-1",
		TenantID:     tenant,
		AgentID:      "agent-a",
		CapabilityID: "forecast",
	}
	if err := store.SaveAgentSLO(ctx, s); err != nil {
		t.Fatalf("failed to save slo: %v", err)
	}
	slos, err := store.ListAgentSLOs(ctx, tenant)
	if err != nil || len(slos) != 1 {
		t.Errorf("expected 1 slo, got %d", len(slos))
	}

	// 5. Proxy Instance
	inst := &fleet.ProxyInstance{
		InstanceID: "inst-1",
		TenantID:   tenant,
		Cluster:    "gke-prod",
	}
	if err := store.SaveProxyInstance(ctx, inst); err != nil {
		t.Fatalf("failed to save proxy: %v", err)
	}
	proxies, err := store.ListProxyInstances(ctx, tenant)
	if err != nil || len(proxies) != 1 {
		t.Errorf("expected 1 proxy, got %d", len(proxies))
	}

	// 6. Routing Model
	m := &learned.RoutingModelRecord{
		ModelID:  "model-1",
		TenantID: tenant,
		Version:  "1.0.0",
		Status:   learned.StatusActive,
	}
	if err := store.SaveRoutingModel(ctx, m); err != nil {
		t.Fatalf("failed to save model: %v", err)
	}
	retM, err := store.GetRoutingModel(ctx, tenant, "model-1")
	if err != nil || retM.Version != "1.0.0" {
		t.Errorf("failed to get model: %v", err)
	}
}

func TestMemoryStorePhase4Entities(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	tenant := "enterprise-x"

	// 1. Optimization Action
	act := &spec.AgentOptimizationAction{
		ActionID:       "act-101",
		OrganizationID: tenant,
		ProjectID:      "proj-1",
		CapabilityID:   "rag",
		ActionType:     spec.ActionChangeRouteWeight,
		RiskClass:      spec.RiskLow,
	}
	if err := store.SaveOptimizationAction(ctx, act); err != nil {
		t.Fatalf("failed to save optimization action: %v", err)
	}
	retAct, err := store.GetOptimizationAction(ctx, tenant, "act-101")
	if err != nil || retAct.ActionType != spec.ActionChangeRouteWeight {
		t.Errorf("failed to retrieve optimization action: %v", err)
	}

	// 2. Routing Spec
	rs := &spec.AgentRoutingSpec{
		CapabilityID:   "rag",
		OrganizationID: tenant,
		Version:        "1.0",
		Weights:        map[string]int{"agent-a": 100},
	}
	if err := store.SaveRoutingSpec(ctx, rs); err != nil {
		t.Fatalf("failed to save routing spec: %v", err)
	}
	retRS, err := store.GetRoutingSpec(ctx, tenant, "rag")
	if err != nil || retRS.Weights["agent-a"] != 100 {
		t.Errorf("failed to retrieve routing spec: %v", err)
	}

	// 3. Automation Policy
	pol := &policy.AutomationPolicy{
		OrganizationID: tenant,
		ProjectID:      "proj-1",
		Mode:           policy.ModeGuardedAutomation,
	}
	if err := store.SaveAutomationPolicy(ctx, pol); err != nil {
		t.Fatalf("failed to save automation policy: %v", err)
	}
	retPol, err := store.GetAutomationPolicy(ctx, tenant, "proj-1")
	if err != nil || retPol.Mode != policy.ModeGuardedAutomation {
		t.Errorf("failed to retrieve automation policy: %v", err)
	}
}

