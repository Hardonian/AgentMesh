package tests_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/adk"
	"github.com/agentmesh/agentmesh/internal/canary"
	"github.com/agentmesh/agentmesh/internal/mcp"
	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/providers"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/pkg/agentbom"
	"github.com/agentmesh/agentmesh/pkg/contracts"
	"github.com/agentmesh/agentmesh/pkg/graph"
	"github.com/agentmesh/agentmesh/pkg/passport"
)

// 1. ADK Graph Inspection, Validation, and Cycle Detection
func TestPhase2_ADKGraphInspectionAndValidation(t *testing.T) {
	// Create a temporary Go project directory
	tmpDir := t.TempDir()
	goCode := `package main
func main() {}
func RunAgent() {
	// Uses bigquery.read tool
	_ = "bigquery.read"
	// Delegates to analyst-agent
	_ = "analyst-agent"
}
`
	err := os.WriteFile(filepath.Join(tmpDir, "agent.go"), []byte(goCode), 0644)
	if err != nil {
		t.Fatalf("failed to write temp go file: %v", err)
	}

	inspector := adk.NewStaticProjectInspector()
	res, err := inspector.InspectProject(tmpDir, "test-agent", "tenant-alpha")
	if err != nil {
		t.Fatalf("InspectProject failed: %v", err)
	}

	if res.Graph == nil {
		t.Fatal("expected non-nil AgentGraph")
	}
	if err := res.Graph.Validate(); err != nil {
		t.Fatalf("valid graph failed validation: %v", err)
	}

	// Deterministic hashing verification
	hash1, _ := res.Graph.Hash()
	hash2, _ := res.Graph.Hash()
	if hash1 != hash2 {
		t.Fatalf("graph hash must be deterministic: %s != %s", hash1, hash2)
	}

	// Cycle detection verification
	cyclicGraph := graph.NewAgentGraph("cyclic", "tenant-alpha", "proj", "agent", "1.0")
	cyclicGraph.Nodes = []graph.Node{
		{ID: "n1", Name: "Step 1", Type: graph.NodeTypeDecision},
		{ID: "n2", Name: "Step 2", Type: graph.NodeTypeDecision},
	}
	cyclicGraph.Entrypoint = "n1"
	cyclicGraph.Edges = []graph.Edge{
		{FromID: "n1", ToID: "n2"},
		{FromID: "n2", ToID: "n1"}, // Cycle!
	}
	cycles := cyclicGraph.FindCycles()
	if len(cycles) == 0 {
		t.Fatal("expected cycle to be detected by Tarjan/DFS")
	}
}

// 2. Confused Deputy Protection across Multi-Hop Delegation
func TestPhase2_ConfusedDeputyAdversarialDelegation(t *testing.T) {
	// Caller has payment.execute explicitly denied
	callerContract := &contracts.AgentContract{
		Metadata: contracts.Metadata{Name: "restricted-caller", Version: "1.0.0"},
		Tools: contracts.ToolsConfig{
			Allow: []string{"web.search"},
			Deny:  []string{"payment.execute"},
		},
	}

	// Deputy has payment.execute in allowed list
	deputyContract := &contracts.AgentContract{
		Metadata: contracts.Metadata{Name: "privileged-deputy", Version: "1.0.0"},
		Tools: contracts.ToolsConfig{
			Allow: []string{"web.search", "payment.execute"},
		},
	}

	sc := adk.NewSecurityContext("restricted-caller", 3)
	err := sc.PushDelegation("privileged-deputy", deputyContract)
	if err != nil {
		t.Fatalf("PushDelegation failed: %v", err)
	}

	// Invariant: Delegation can only narrow privileges, never expand them!
	allowed, reason := sc.CanExecuteTool("payment.execute", callerContract)
	if allowed {
		t.Fatal("CONFUSED DEPUTY VIOLATION: delegated sub-agent gained access to payment.execute forbidden to original principal")
	}
	t.Logf("Confused deputy correctly blocked: %s", reason)

	allowedSafe, _ := sc.CanExecuteTool("web.search", callerContract)
	if !allowedSafe {
		t.Fatal("expected web.search to remain authorized for delegated deputy")
	}
}

// 3. Static Graph-Level Policy Analysis detecting Forbidden Paths
func TestPhase2_GraphPolicyAnalysisForbiddenPath(t *testing.T) {
	g := graph.NewAgentGraph("g_finance", "tenant-corp", "proj", "finance-agent", "1.1")
	g.Nodes = []graph.Node{
		{ID: "entry", Name: "Entry", Type: graph.NodeTypeDecision},
		{ID: "research", Name: "Research Agent", Type: graph.NodeTypeAgent, Target: "research-agent"},
		{ID: "gmail", Name: "Gmail Dispatcher", Type: graph.NodeTypeTool, Target: "gmail.send"},
	}
	g.Entrypoint = "entry"
	g.Edges = []graph.Edge{
		{FromID: "entry", ToID: "research"},
		{FromID: "research", ToID: "gmail"},
	}

	// Policy: Finance agent cannot use gmail.send
	pol := &policy.Policy{
		ID:       "corp-exfiltration-prevention",
		TenantID: "tenant-corp",
		Rules: []policy.Rule{
			{
				Name:    "Block finance from sending external email",
				Effect:  policy.EffectDeny,
				Agents:  []string{"finance-agent"},
				Tools:   []string{"gmail.send"},
				Actions: []string{"*"},
			},
		},
	}

	report := policy.AnalyzeGraphPolicy(g, pol)
	if report.Compliant {
		t.Fatal("expected static graph policy analysis to flag forbidden delegation path to gmail.send")
	}
	if len(report.Findings) == 0 {
		t.Fatal("expected findings for forbidden tool path")
	}
}

// 4. MCP Schema Drift and Fingerprint Detection
func TestPhase2_MCPSchemaDriftAndFingerprint(t *testing.T) {
	schemaV1 := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
		},
		"required": []string{"query"},
	}
	fp1, err := mcp.CalculateToolFingerprint("mcp-srv", "query-tool", "1.0.0", "google", mcp.RiskClassRead, schemaV1)
	if err != nil {
		t.Fatalf("CalculateToolFingerprint failed: %v", err)
	}

	// Identical schema must produce identical digest
	fp1Clone, _ := mcp.CalculateToolFingerprint("mcp-srv", "query-tool", "1.0.0", "google", mcp.RiskClassRead, schemaV1)
	if fp1.Digest != fp1Clone.Digest {
		t.Fatalf("tool fingerprint must be deterministic: %s != %s", fp1.Digest, fp1Clone.Digest)
	}

	// Compatible change: adding optional field
	schemaV2 := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
			"limit": map[string]any{"type": "integer"},
		},
		"required": []string{"query"},
	}
	fp2, _ := mcp.CalculateToolFingerprint("mcp-srv", "query-tool", "1.1.0", "google", mcp.RiskClassRead, schemaV2)
	driftCompat := mcp.DetectSchemaDrift(fp1, fp2)
	if driftCompat != mcp.DriftCompatibleChange {
		t.Fatalf("expected DriftCompatibleChange, got: %s", driftCompat)
	}

	// Breaking change: adding a required field
	schemaV3 := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query":     map[string]any{"type": "string"},
			"auth_token": map[string]any{"type": "string"},
		},
		"required": []string{"query", "auth_token"},
	}
	fp3, _ := mcp.CalculateToolFingerprint("mcp-srv", "query-tool", "2.0.0", "google", mcp.RiskClassRead, schemaV3)
	driftBreak := mcp.DetectSchemaDrift(fp1, fp3)
	if driftBreak != mcp.DriftBreaking {
		t.Fatalf("expected DriftBreaking, got: %s", driftBreak)
	}
}

// 5. Capability-Aware Routing V2 with Evidence Tiers & Explanation
func TestPhase2_CapabilityRoutingV2(t *testing.T) {
	allowPol := &policy.Policy{
		ID:       "pol-allow-route",
		TenantID: "tenant-alpha",
		Rules: []policy.Rule{
			{
				Name:    "Allow routing for tenant",
				Effect:  policy.EffectAllow,
				Agents:  []string{"*"},
				Actions: []string{"route", "invoke"},
			},
		},
	}
	router := routing.NewRouter(policy.NewEngine([]*policy.Policy{allowPol}))

	// Candidate A: Highly evaluated with production evidence
	candA := &routing.AgentRouteCandidate{
		AgentID:     "agent-prod",
		EndpointURL: "http://agent-prod.internal:8080",
		Status:      "HEALTHY",
		Contract: &contracts.AgentContract{
			Metadata:     contracts.Metadata{Name: "agent-prod", Version: "1.0.0"},
			Capabilities: []string{"invoice_analysis"},
			Tools:        contracts.ToolsConfig{Allow: []string{"bigquery.read"}},
			SLO:          contracts.SLOConfig{P95LatencyMs: 2500, SuccessRate: 0.99},
			Budgets:      contracts.BudgetConfig{MaxCostPerTask: 0.05},
		},
		SuccessRate:  0.995,
		P95LatencyMs: 2200,
		AverageCost:  0.03,
		Passport: &passport.AgentPassport{
			Reliability: passport.ReliabilityMetrics{
				SampleCount:     50,
				TaskSuccessRate: 0.995,
			},
		},
	}
	router.RegisterCandidate(candA)

	// Candidate B: Low reliability candidate
	candB := &routing.AgentRouteCandidate{
		AgentID:     "agent-restricted",
		EndpointURL: "http://agent-restricted.internal:8080",
		Status:      "HEALTHY",
		Contract: &contracts.AgentContract{
			Metadata:     contracts.Metadata{Name: "agent-restricted", Version: "0.5.0"},
			Capabilities: []string{"invoice_analysis"},
			Tools:        contracts.ToolsConfig{Allow: []string{"bigquery.read"}},
			SLO:          contracts.SLOConfig{P95LatencyMs: 8000, SuccessRate: 0.90},
			Budgets:      contracts.BudgetConfig{MaxCostPerTask: 0.20},
		},
		SuccessRate:  0.88,
		P95LatencyMs: 7500,
		AverageCost:  0.12,
	}
	router.RegisterCandidate(candB)

	// Execute capability routing
	req := &routing.RouteRequestV2{
		TenantID:           "tenant-alpha",
		CallerAgentID:      "caller-portal",
		RequiredCapability: "invoice_analysis",
		Strategy:           routing.StrategyHighestQuality,
	}

	decision, err := router.RouteV2(context.Background(), req)
	if err != nil {
		t.Fatalf("RouteV2 failed: %v", err)
	}

	if decision.SelectedAgentID != "agent-prod" {
		t.Fatalf("expected agent-prod to be selected, got: %s", decision.SelectedAgentID)
	}
	if decision.Confidence < 0.70 {
		t.Fatalf("expected high confidence for evaluated candidate, got: %.2f", decision.Confidence)
	}
	if len(decision.Candidates) != 2 {
		t.Fatalf("expected 2 candidates in auditable explanation, got: %d", len(decision.Candidates))
	}
}

// 6. Policy Shadow Canary Mode
func TestPhase2_PolicyShadowCanary(t *testing.T) {
	baselinePolicy := &policy.Policy{
		ID:       "pol-baseline",
		TenantID: "tenant-corp",
		Rules: []policy.Rule{
			{Name: "Allow bigquery", Effect: policy.EffectAllow, Tools: []string{"bigquery.read"}, Actions: []string{"*"}},
		},
	}
	candidatePolicy := &policy.Policy{
		ID:       "pol-candidate",
		TenantID: "tenant-corp",
		Rules: []policy.Rule{
			{Name: "Deny bigquery in candidate", Effect: policy.EffectDeny, Tools: []string{"bigquery.read"}, Actions: []string{"*"}},
		},
	}

	canaryObj := &policy.PolicyCanary{
		ID:              "canary-01",
		TenantID:        "tenant-corp",
		BaselinePolicy:  baselinePolicy,
		CandidatePolicy: candidatePolicy,
		ShadowMode:      true,
		CreatedAt:       time.Now().UTC(),
	}

	shadow := policy.NewShadowEvaluator(canaryObj)
	req := &policy.EvaluationRequest{
		TenantID:       "tenant-corp",
		SubjectAgentID: "agent-1",
		Tool:           "bigquery.read",
		Action:         "query",
	}

	enforcedDec, shadowDec, err := shadow.EvaluateShadow(context.Background(), "canary-01", req)
	if err != nil {
		t.Fatalf("EvaluateShadow failed: %v", err)
	}

	// Invariant: Live enforcement MUST follow baseline policy (ALLOW)
	if enforcedDec.Effect != policy.EffectAllow {
		t.Fatalf("enforced decision violated baseline: expected ALLOW, got: %s", enforcedDec.Effect)
	}

	// Shadow evaluation records candidate would-deny
	if shadowDec.Effect != policy.EffectDeny {
		t.Fatalf("shadow decision expected DENY, got: %s", shadowDec.Effect)
	}

	if canaryObj.WouldDenyCount != 1 {
		t.Fatalf("expected WouldDenyCount=1, got: %d", canaryObj.WouldDenyCount)
	}
}

// 7. Security-Sensitive Change Impact Analysis
func TestPhase2_AgentChangeImpactSecuritySensitive(t *testing.T) {
	current := &contracts.AgentContract{
		Metadata: contracts.Metadata{Name: "payment-agent", Version: "1.0.0"},
		Tools:    contracts.ToolsConfig{Allow: []string{"bigquery.read"}},
		Delegation: contracts.DelegationConfig{MaxDepth: 2},
	}

	// Candidate introduces destructive tool and increases delegation depth
	candidate := &contracts.AgentContract{
		Metadata: contracts.Metadata{Name: "payment-agent", Version: "1.1.0"},
		Tools:    contracts.ToolsConfig{Allow: []string{"bigquery.read", "payment.execute", "gke.cluster.delete"}},
		Delegation: contracts.DelegationConfig{MaxDepth: 5},
	}

	report, err := canary.AnalyzeChangeImpact(current, candidate)
	if err != nil {
		t.Fatalf("AnalyzeChangeImpact failed: %v", err)
	}

	if !report.RequiresExplicitPolicyReview {
		t.Fatal("expected security-sensitive changes to require explicit policy review")
	}
	if report.SafeToCanary {
		t.Fatal("expected SafeToCanary to be false when unreviewed destructive tools are added")
	}
	if len(report.SecuritySensitiveFlags) == 0 {
		t.Fatal("expected security sensitive flags to be populated")
	}
}

// 8. Policy-Governed Model Fallback
func TestPhase2_ModelRouterAndFallbackPolicy(t *testing.T) {
	router := providers.NewModelRouter()

	// Primary model target with UNAVAILABLE state
	router.RegisterTarget(&providers.ModelTarget{
		ModelID:      "gemini-1.5-pro",
		Provider:     "gemini",
		Region:       "us-central1",
		CostPer1kIn:  0.002,
		CostPer1kOut: 0.005,
		HealthStatus: "UNAVAILABLE",
	})

	// Fallback model target healthy
	router.RegisterTarget(&providers.ModelTarget{
		ModelID:      "gemini-1.5-flash",
		Provider:     "gemini",
		Region:       "us-central1",
		CostPer1kIn:  0.0005,
		CostPer1kOut: 0.0015,
		HealthStatus: "HEALTHY",
	})

	genReq := &providers.GenerateRequest{
		Prompt: "Financial analysis prompt",
	}

	// Policy allows gemini-1.5-flash fallback
	allowedModels := []string{"gemini-1.5-pro", "gemini-1.5-flash"}
	resp, fbEvent, err := router.GenerateWithFallback(context.Background(), "gemini-1.5-pro", "gemini-1.5-flash", allowedModels, genReq)
	if err != nil {
		t.Fatalf("GenerateWithFallback failed: %v", err)
	}

	if fbEvent == nil || !fbEvent.AllowedByPolicy {
		t.Fatalf("expected permitted fallback event, got: %+v", fbEvent)
	}
	if resp == nil {
		t.Fatal("expected non-nil response from fallback target")
	}

	// When fallback model is NOT in allowed models, it must fail safely
	restrictedModels := []string{"gemini-1.5-pro"} // Flash NOT permitted!
	_, _, err = router.GenerateWithFallback(context.Background(), "gemini-1.5-pro", "gemini-1.5-flash", restrictedModels, genReq)
	if err == nil {
		t.Fatal("expected failure when fallback model family is not authorized by policy")
	}
}

// 9. Agent Passport V2 Sanitization and Badges
func TestPhase2_AgentPassportV2SanitizationAndBadges(t *testing.T) {
	contract := &contracts.AgentContract{
		Metadata:     contracts.Metadata{Name: "secret-agent", Version: "2.0.0", Organization: "acme-corp"},
		Identity:     contracts.IdentityConfig{Protocols: []string{"a2a", "mcp"}},
		Capabilities: []string{"internal_research"},
		Tools:        contracts.ToolsConfig{Allow: []string{"internal_db.read"}},
		SLO:          contracts.SLOConfig{P95LatencyMs: 1500, SuccessRate: 0.999},
		Budgets:      contracts.BudgetConfig{MaxCostPerTask: 0.01},
	}

	p, err := passport.GenerateFromContract(contract, "go", "google-adk")
	if err != nil {
		t.Fatalf("GenerateFromContract failed: %v", err)
	}

	// Verify badge generation
	badge := p.GenerateBadge()
	if badge == "" {
		t.Fatal("badge string cannot be empty")
	}

	// Verify private passport returns nil by default
	if p.IsPublic {
		t.Fatal("expected passport to be private by default")
	}
	if p.SanitizeForPublic() != nil {
		t.Fatal("private passport must return nil when sanitized for public")
	}

	// When explicitly marked public, sanitize and verify redactions
	p.IsPublic = true
	publicPass := p.SanitizeForPublic()
	if publicPass == nil {
		t.Fatal("expected non-nil sanitized public passport")
	}
	if publicPass.Identity.AgentID != "secret-agent" {
		t.Fatalf("expected agent ID to be retained, got: %s", publicPass.Identity.AgentID)
	}
	// Verify private operational details are scrubbed
	if len(publicPass.Graph.Tools) != 1 || publicPass.Graph.Tools[0] == "internal_db.read" {
		t.Fatalf("expected private tools to be redacted, got: %v", publicPass.Graph.Tools)
	}
}

// 10. AgentBOM V2 Linkage with Graph Hash and Tool Fingerprints
func TestPhase2_AgentBOMV2Linkage(t *testing.T) {
	contract := &contracts.AgentContract{
		Metadata:     contracts.Metadata{Name: "bom-agent", Version: "1.0.0", Organization: "acme-corp"},
		Identity:     contracts.IdentityConfig{Protocols: []string{"a2a", "mcp"}},
		Capabilities: []string{"quote_analysis"},
		Tools:        contracts.ToolsConfig{Allow: []string{"bigquery.read"}},
		SLO:          contracts.SLOConfig{P95LatencyMs: 2000, SuccessRate: 0.99},
	}

	bom, err := agentbom.GenerateFromContract(contract, "go", "google-adk")
	if err != nil {
		t.Fatalf("GenerateFromContract failed: %v", err)
	}

	// Attach Phase 2 linkage
	bom.GraphHash = "8f3b20c19a8421d509b552309f482a229cb0184c281048b6d3910c2a4f89d31b"
	bom.SoftwareSBOMDigest = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if len(bom.Tools) > 0 {
		bom.Tools[0].Fingerprint = "d9e810a42f5342a19b48c772e0a293817f0932c64e81fa214309a90184b2cd5e"
	}

	h1, err := bom.Hash()
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}
	h2, _ := bom.Hash()
	if h1 != h2 {
		t.Fatalf("AgentBOM V2 hash must be deterministic: %s != %s", h1, h2)
	}

	bomJSON, err := bom.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var parsed agentbom.AgentBOM
	if err := json.Unmarshal(bomJSON, &parsed); err != nil {
		t.Fatalf("unmarshal bom failed: %v", err)
	}
	if parsed.GraphHash != bom.GraphHash {
		t.Fatalf("graph hash not preserved: %s != %s", parsed.GraphHash, bom.GraphHash)
	}
}
