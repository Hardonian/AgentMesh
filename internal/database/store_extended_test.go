package database

import (
	"context"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/a2a"
	"github.com/agentmesh/agentmesh/internal/evaluation"
	"github.com/agentmesh/agentmesh/internal/identity"
	"github.com/agentmesh/agentmesh/internal/mcp"
	"github.com/agentmesh/agentmesh/internal/outcome"
	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/pkg/contracts"
	"github.com/agentmesh/agentmesh/pkg/graph"
	"github.com/agentmesh/agentmesh/pkg/spec"
)

func TestMemoryStore_CoreEntities(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	tenant := "tenant-alpha"

	// 1. Agents: Save, Get, List, Delete
	agent := &AgentRecord{
		ID:       "agent-01",
		TenantID: tenant,
		Name:     "Agent 01",
		Status:   "HEALTHY",
		Contract: &contracts.AgentContract{
			Metadata: contracts.Metadata{
				Name: "Agent 01",
			},
		},
	}
	if err := store.SaveAgent(ctx, agent); err != nil {
		t.Fatalf("failed to save agent: %v", err)
	}
	gotAgent, err := store.GetAgent(ctx, tenant, "agent-01")
	if err != nil || gotAgent.ID != "agent-01" {
		t.Fatalf("failed to get agent: %v", err)
	}
	agents, err := store.ListAgents(ctx, tenant)
	if err != nil || len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if err := store.DeleteAgent(ctx, tenant, "agent-01"); err != nil {
		t.Fatalf("failed to delete agent: %v", err)
	}
	if _, err := store.GetAgent(ctx, tenant, "agent-01"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after deletion, got %v", err)
	}

	// 2. Policies: Save, Get, List
	pol := &policy.Policy{
		ID:       "pol-01",
		TenantID: tenant,
		Name:     "Policy 01",
		Rules: []policy.Rule{
			{Name: "r1", Effect: policy.EffectAllow},
		},
	}
	if err := store.SavePolicy(ctx, pol); err != nil {
		t.Fatalf("failed to save policy: %v", err)
	}
	gotPol, err := store.GetPolicy(ctx, tenant, "pol-01")
	if err != nil || gotPol.ID != "pol-01" {
		t.Fatalf("failed to get policy: %v", err)
	}
	pols, err := store.ListPolicies(ctx, tenant)
	if err != nil || len(pols) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(pols))
	}

	// 3. Credentials: Save, GetByHash, List
	cred := &identity.Credential{
		ID:        "cred-01",
		TenantID:  tenant,
		SubjectID: "admin-01",
		HashedKey: "sha256_dummy_hash",
		Scopes:    []string{identity.ScopeAdmin},
	}
	if err := store.SaveCredential(ctx, cred); err != nil {
		t.Fatalf("failed to save credential: %v", err)
	}
	gotCred, err := store.GetCredentialByHash(ctx, "sha256_dummy_hash")
	if err != nil || gotCred.ID != "cred-01" {
		t.Fatalf("failed to get credential: %v", err)
	}
	creds, err := store.ListCredentials(ctx, tenant)
	if err != nil || len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}

	// 4. Tools: Save, List
	tool := &ToolRecord{
		ID:       "tool-01",
		TenantID: tenant,
		Name:     "Tool 01",
	}
	if err := store.SaveTool(ctx, tool); err != nil {
		t.Fatalf("failed to save tool: %v", err)
	}
	tools, err := store.ListTools(ctx, tenant)
	if err != nil || len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	// 5. Graphs: Save, Get, List
	g := &graph.AgentGraph{
		GraphID: "graph-01",
		AgentID: "agent-01",
		Nodes:   []graph.Node{{ID: "n1", Name: "root"}},
	}
	if err := store.SaveGraph(ctx, tenant, g); err != nil {
		t.Fatalf("failed to save graph: %v", err)
	}
	gotG, err := store.GetGraph(ctx, tenant, "graph-01")
	if err != nil || gotG.GraphID != "graph-01" {
		t.Fatalf("failed to get graph: %v", err)
	}
	graphs, err := store.ListGraphs(ctx, tenant)
	if err != nil || len(graphs) != 1 {
		t.Fatalf("expected 1 graph, got %d", len(graphs))
	}

	// 6. Tool Passports: Save, Get, List
	tp := &mcp.ToolPassport{
		ToolID:    "tp-01",
		ToolName:  "Tool Passport 01",
		RiskClass: mcp.RiskClassRead,
	}
	if err := store.SaveToolPassport(ctx, tenant, tp); err != nil {
		t.Fatalf("failed to save tool passport: %v", err)
	}
	gotTP, err := store.GetToolPassport(ctx, tenant, "tp-01")
	if err != nil || gotTP.ToolID != "tp-01" {
		t.Fatalf("failed to get tool passport: %v", err)
	}
	tps, err := store.ListToolPassports(ctx, tenant)
	if err != nil || len(tps) != 1 {
		t.Fatalf("expected 1 tool passport, got %d", len(tps))
	}

	// 7. A2A Profiles: Save, Get, List
	a2aProf := &a2a.A2ACompatibilityProfile{
		TesterVersion: "v1.0",
		Status:        a2a.StatusCompatible,
	}
	if err := store.SaveA2AProfile(ctx, tenant, a2aProf); err != nil {
		t.Fatalf("failed to save a2a profile: %v", err)
	}
	profiles, err := store.ListA2AProfiles(ctx, tenant)
	if err != nil || len(profiles) != 1 {
		t.Fatalf("expected 1 a2a profile, got %d", len(profiles))
	}

	// 8. Route Outcomes V1: Save, List
	rOut := &routing.RouteOutcome{
		ID:            "out-01",
		TenantID:      tenant,
		Capability:    "search",
		SelectedAgent: "agent-01",
		Success:       true,
	}
	if err := store.SaveRouteOutcome(ctx, rOut); err != nil {
		t.Fatalf("failed to save route outcome: %v", err)
	}
	rOutcomes, err := store.ListRouteOutcomes(ctx, tenant)
	if err != nil || len(rOutcomes) != 1 {
		t.Fatalf("expected 1 route outcome, got %d", len(rOutcomes))
	}

	// 9. Evaluation Suites: Save, Get, List
	suite := &evaluation.EvaluationSuite{
		ID:          "suite-01",
		TenantID:    tenant,
		Capability:  "search",
		Description: "Safety Benchmark",
	}
	if err := store.SaveEvaluationSuite(ctx, suite); err != nil {
		t.Fatalf("failed to save eval suite: %v", err)
	}
	gotSuite, err := store.GetEvaluationSuite(ctx, tenant, "suite-01")
	if err != nil || gotSuite.ID != "suite-01" {
		t.Fatalf("failed to get eval suite: %v", err)
	}
	suites, err := store.ListEvaluationSuites(ctx, tenant)
	if err != nil || len(suites) != 1 {
		t.Fatalf("expected 1 eval suite, got %d", len(suites))
	}

	// 10. Optimization Actions: Save, Get, List
	action := &spec.AgentOptimizationAction{
		ActionID:       "act-01",
		OrganizationID: tenant,
		CapabilityID:   "search",
		TargetType:     "ROUTE",
		TargetID:       "target-1",
		Reason:         "optimization",
		CreatedAt:      time.Now().UTC(),
	}
	if err := store.SaveOptimizationAction(ctx, action); err != nil {
		t.Fatalf("failed to save optimization action: %v", err)
	}
	gotAction, err := store.GetOptimizationAction(ctx, tenant, "act-01")
	if err != nil || gotAction.ActionID != "act-01" {
		t.Fatalf("failed to get optimization action: %v", err)
	}
	actions, err := store.ListOptimizationActions(ctx, tenant)
	if err != nil || len(actions) != 1 {
		t.Fatalf("expected 1 optimization action, got %d", len(actions))
	}

	// 11. Routing Specs: Save, Get, List
	routingSpec := &spec.AgentRoutingSpec{
		CapabilityID:   "search",
		OrganizationID: tenant,
		Version:        "1.0.0",
	}
	if err := store.SaveRoutingSpec(ctx, routingSpec); err != nil {
		t.Fatalf("failed to save routing spec: %v", err)
	}
	gotSpec, err := store.GetRoutingSpec(ctx, tenant, "search")
	if err != nil || gotSpec.CapabilityID != "search" {
		t.Fatalf("failed to get routing spec: %v", err)
	}
	rSpecs, err := store.ListRoutingSpecs(ctx, tenant)
	if err != nil || len(rSpecs) != 1 {
		t.Fatalf("expected 1 routing spec, got %d", len(rSpecs))
	}

	// 12. Production Outcomes: Save, List
	prodOutcome := &outcome.AgentProductionOutcome{
		OutcomeID:      "prod-out-01",
		OrganizationID: tenant,
		CapabilityID:   "search",
		Status:         outcome.OutcomeVerified,
	}
	if err := store.SaveProductionOutcome(ctx, prodOutcome); err != nil {
		t.Fatalf("failed to save production outcome: %v", err)
	}
	prodOutcomes, err := store.ListProductionOutcomes(ctx, tenant, "search")
	if err != nil || len(prodOutcomes) != 1 {
		t.Fatalf("expected 1 production outcome, got %d", len(prodOutcomes))
	}

	// 13. Automation Policies: Save, Get
	autoPol := &policy.AutomationPolicy{
		OrganizationID: tenant,
		ProjectID:      "proj-alpha",
		Mode:           policy.ModeGuardedAutomation,
	}
	if err := store.SaveAutomationPolicy(ctx, autoPol); err != nil {
		t.Fatalf("failed to save automation policy: %v", err)
	}
	gotAutoPol, err := store.GetAutomationPolicy(ctx, tenant, "proj-alpha")
	if err != nil || gotAutoPol.ProjectID != "proj-alpha" {
		t.Fatalf("failed to get automation policy: %v", err)
	}
}

func TestMemoryStore_FailClosedTenantIsolation(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Empty tenant should fail closed with ErrEmptyTenant
	if _, err := store.ListAgents(ctx, ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListAgents, got %v", err)
	}
	if _, err := store.ListPolicies(ctx, ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListPolicies, got %v", err)
	}
	if _, err := store.ListCredentials(ctx, ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListCredentials, got %v", err)
	}
	if _, err := store.ListTools(ctx, ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListTools, got %v", err)
	}
	if _, err := store.ListGraphs(ctx, ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListGraphs, got %v", err)
	}
	if _, err := store.ListToolPassports(ctx, ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListToolPassports, got %v", err)
	}
	if _, err := store.ListA2AProfiles(ctx, ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListA2AProfiles, got %v", err)
	}
	if _, err := store.ListRouteOutcomes(ctx, ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListRouteOutcomes, got %v", err)
	}
	if _, err := store.ListEvaluationSuites(ctx, ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListEvaluationSuites, got %v", err)
	}
	if _, err := store.ListRoutingOutcomesV3(ctx, "", ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListRoutingOutcomesV3, got %v", err)
	}
	if _, err := store.ListAgentSLOs(ctx, ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListAgentSLOs, got %v", err)
	}
	if _, err := store.ListProxyInstances(ctx, ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListProxyInstances, got %v", err)
	}
	if _, err := store.ListRoutingModels(ctx, ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListRoutingModels, got %v", err)
	}
	if _, err := store.ListOptimizationActions(ctx, ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListOptimizationActions, got %v", err)
	}
	if _, err := store.ListRoutingSpecs(ctx, ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListRoutingSpecs, got %v", err)
	}
	if _, err := store.ListProductionOutcomes(ctx, "", ""); err != ErrEmptyTenant {
		t.Errorf("expected ErrEmptyTenant on ListProductionOutcomes, got %v", err)
	}
}

func TestMemoryStore_NotFoundErrors(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	tenant := "tenant-alpha"

	if _, err := store.GetAgent(ctx, tenant, "non-existent"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if _, err := store.GetPolicy(ctx, tenant, "non-existent"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if _, err := store.GetCredentialByHash(ctx, "non-existent"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if _, err := store.GetGraph(ctx, tenant, "non-existent"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if _, err := store.GetToolPassport(ctx, tenant, "non-existent"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if _, err := store.GetA2AProfile(ctx, tenant, "non-existent"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if _, err := store.GetEvaluationSuite(ctx, tenant, "non-existent"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if _, err := store.GetTaskFingerprint(ctx, tenant, "non-existent"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if _, err := store.GetReliabilityProfile(ctx, tenant, "non-existent", "non-existent"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if _, err := store.GetAgentSLO(ctx, tenant, "non-existent", "non-existent"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if _, err := store.GetRoutingModel(ctx, tenant, "non-existent"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if _, err := store.GetOptimizationAction(ctx, tenant, "non-existent"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if _, err := store.GetRoutingSpec(ctx, tenant, "non-existent"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	defaultPol, err := store.GetAutomationPolicy(ctx, tenant, "non-existent")
	if err != nil || defaultPol.Mode != policy.ModeAdvisory {
		t.Errorf("expected default advisory policy when not found, got %v, err=%v", defaultPol, err)
	}
}
