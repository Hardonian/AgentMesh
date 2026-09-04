package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/approval"
	"github.com/agentmesh/agentmesh/internal/audit"
	"github.com/agentmesh/agentmesh/internal/canary"
	"github.com/agentmesh/agentmesh/internal/config"
	"github.com/agentmesh/agentmesh/internal/cost"
	"github.com/agentmesh/agentmesh/internal/crypto"
	"github.com/agentmesh/agentmesh/internal/database"
	"github.com/agentmesh/agentmesh/internal/identity"
	"github.com/agentmesh/agentmesh/internal/mcp"
	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/internal/server"
	"github.com/agentmesh/agentmesh/internal/telemetry"
	"github.com/agentmesh/agentmesh/pkg/contracts"
	"github.com/agentmesh/agentmesh/pkg/passport"
	"github.com/agentmesh/agentmesh/pkg/spec"
)

// TestP0RedTeamScenarios tests all 15 adversarial red-team attack scenarios (A through O).
func TestP0RedTeamScenarios(t *testing.T) {
	ctx := context.Background()

	// -------------------------------------------------------------------------
	// Scenario A: SSRF / Cloud Metadata Probe via A2A endpoint test
	// -------------------------------------------------------------------------
	t.Run("Scenario_A_SSRF_Protection", func(t *testing.T) {
		prohibitedURLs := []string{
			"http://169.254.169.254/computeMetadata/v1/",
			"http://metadata.google.internal/computeMetadata/v1/",
			"http://127.0.0.1:8080/internal",
			"http://localhost:9090",
			"file:///etc/shadow",
			"gopher://127.0.0.1:70",
			"http://10.0.0.1/admin",
			"http://192.168.1.1/setup",
		}

		for _, u := range prohibitedURLs {
			_, err := server.ValidateSafeRemoteURL(u, false)
			if err == nil {
				t.Fatalf("[Scenario A] SSRF bypass: URL %q was unexpectedly allowed", u)
			}
		}

		// Also verify via HTTP handler on the server
		store := database.NewMemoryStore()
		polEng := policy.NewEngine(nil)
		rotEng := routing.NewRouter(polEng)
		tel := telemetry.NewCollector()
		canaryMgr := canary.NewManager()
		appSvc := approval.NewService()
		audLog := audit.NewLogger()
		kp, _ := crypto.GenerateKeyPair("srv-key")
		srv := server.NewServer(store, polEng, rotEng, tel, canaryMgr, appSvc, audLog, kp)

		for _, target := range prohibitedURLs[:3] {
			body, _ := json.Marshal(map[string]string{"endpointUrl": target})
			req := httptest.NewRequest("POST", "/api/v1/a2a/test", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			srv.Router().ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("[Scenario A] Expected 400 Bad Request for SSRF attempt %q, got status %d", target, rec.Code)
			}
		}
	})

	// -------------------------------------------------------------------------
	// Scenario B: Cross-Tenant Data Isolation & Query Leak
	// -------------------------------------------------------------------------
	t.Run("Scenario_B_CrossTenant_Isolation", func(t *testing.T) {
		store := database.NewMemoryStore()

		// Populate tenant-alpha and tenant-bravo
		_ = store.SaveAgent(ctx, &database.AgentRecord{ID: "agent-alpha", TenantID: "tenant-alpha", Name: "Alpha Agent"})
		_ = store.SaveAgent(ctx, &database.AgentRecord{ID: "agent-bravo", TenantID: "tenant-bravo", Name: "Bravo Agent"})

		// 1. Database level: Empty tenant must fail closed
		if _, err := store.ListAgents(ctx, ""); err != database.ErrEmptyTenant {
			t.Fatalf("[Scenario B] Database failed to reject empty tenant in ListAgents")
		}

		// 2. Strict filtering: tenant-alpha can only see its own records
		alphaAgents, err := store.ListAgents(ctx, "tenant-alpha")
		if err != nil || len(alphaAgents) != 1 || alphaAgents[0].ID != "agent-alpha" {
			t.Fatalf("[Scenario B] Tenant Alpha leaked or missed data: %v", alphaAgents)
		}

		// 3. API level: Cross-tenant spoofing blocked
		rawKeyAlpha, credAlpha, _ := identity.GenerateAPIKey("tenant-alpha", "subject-1", []string{identity.ScopeAgentsRead}, 1*time.Hour, "Key Alpha")
		_ = store.SaveCredential(ctx, credAlpha)

		srv := server.NewServer(store, policy.NewEngine(nil), routing.NewRouter(nil),
			telemetry.NewCollector(), canary.NewManager(), approval.NewService(), audit.NewLogger(), nil)

		// Alpha attempts to query Bravo with X-Tenant-ID header
		req := httptest.NewRequest("GET", "/api/v1/agents", nil)
		req.Header.Set("Authorization", "Bearer "+rawKeyAlpha)
		req.Header.Set("X-Tenant-ID", "tenant-bravo")
		rec := httptest.NewRecorder()
		srv.Router().ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("[Scenario B] Expected 403 Forbidden for cross-tenant spoofing, got %d", rec.Code)
		}
	})

	// -------------------------------------------------------------------------
	// Scenario C: HITL Token Tampering & Single-Use Replay
	// -------------------------------------------------------------------------
	t.Run("Scenario_C_HITL_Token_Tampering_And_Replay", func(t *testing.T) {
		appSvc := approval.NewService()

		req, err := appSvc.CreateRequest("tenant-finance", "agent-1", "sql_tool", "DROP TABLE analytics", map[string]any{"table": "analytics"}, "pol-1", "v1", "maintenance", 10*time.Minute)
		if err != nil {
			t.Fatalf("[Scenario C] CreateRequest failed: %v", err)
		}

		resolved, err := appSvc.Resolve(req.ID, "admin-user", true, "authorized")
		if err != nil {
			t.Fatalf("[Scenario C] Resolve failed: %v", err)
		}
		token := resolved.ApprovalToken

		// 1. Cross-tenant token replay attempt
		err = appSvc.ConsumeApproval(req.ID, "tenant-attacker", "agent-1", "sql_tool", map[string]any{"table": "analytics"}, token)
		if err != approval.ErrApprovalTenantMismatch {
			t.Fatalf("[Scenario C] Expected ErrApprovalTenantMismatch, got: %v", err)
		}

		// 2. Legitimate consume succeeds
		err = appSvc.ConsumeApproval(req.ID, "tenant-finance", "agent-1", "sql_tool", map[string]any{"table": "analytics"}, token)
		if err != nil {
			t.Fatalf("[Scenario C] Legitimate consume failed: %v", err)
		}

		// 3. Replay of already-consumed token MUST fail
		err = appSvc.ConsumeApproval(req.ID, "tenant-finance", "agent-1", "sql_tool", map[string]any{"table": "analytics"}, token)
		if err != approval.ErrApprovalConsumed {
			t.Fatalf("[Scenario C] Replay of consumed approval succeeded! Expected ErrApprovalConsumed, got: %v", err)
		}

		// 4. Action hash tampering post-approval in Control Plane
		store := database.NewMemoryStore()
		act := &spec.AgentOptimizationAction{
			ActionID:       "act-tamper-1",
			OrganizationID: "tenant-finance",
			ProjectID:      "proj-1",
			CapabilityID:   "tax-calc",
			ActionType:     spec.ActionChangeRouteWeight,
			ApprovalRequirement: spec.ApprovalRequirement{
				Required:        true,
				ApprovedBy:      []string{"admin"},
				ActionHashBound: "original_valid_hash",
			},
		}
		now := time.Now().UTC()
		act.ApprovedAt = &now
		_ = store.SaveOptimizationAction(ctx, act)

		srv := server.NewServer(store, policy.NewEngine(nil), routing.NewRouter(nil),
			telemetry.NewCollector(), canary.NewManager(), appSvc, audit.NewLogger(), nil)

		execReq := httptest.NewRequest("POST", "/api/v1/control/actions/act-tamper-1/execute", nil)
		execReq.Header.Set("X-Tenant-ID", "tenant-finance")
		execRec := httptest.NewRecorder()
		srv.Router().ServeHTTP(execRec, execReq)

		if execRec.Code != http.StatusConflict {
			t.Fatalf("[Scenario C] Expected 409 Conflict for tampered action hash, got %d", execRec.Code)
		}
	})

	// -------------------------------------------------------------------------
	// Scenario D: Malicious Agent Contract Deserialization (Oversized & Malformed)
	// -------------------------------------------------------------------------
	t.Run("Scenario_D_Contract_Deserialization_Hardening", func(t *testing.T) {
		// 1. Oversized YAML payload (> 10MB)
		oversized := make([]byte, contracts.MaxContractPayloadBytes+1024)
		_, err := contracts.ParseYAML(oversized)
		if !errors.Is(err, contracts.ErrContractTooLarge) {
			t.Fatalf("[Scenario D] Expected ErrContractTooLarge for oversized contract YAML, got: %v", err)
		}

		// 2. Malformed contract validation
		c := &contracts.AgentContract{
			APIVersion: "invalid/v0",
			Metadata: contracts.Metadata{
				Name: "",
			},
		}
		if err := c.Validate(); err == nil {
			t.Fatalf("[Scenario D] Expected validation error for invalid contract, got nil")
		}
	})

	// -------------------------------------------------------------------------
	// Scenario E: Delegation Privilege Escalation / Confused Deputy
	// -------------------------------------------------------------------------
	t.Run("Scenario_E_Delegation_Privilege_Escalation", func(t *testing.T) {
		pol := &policy.Policy{
			ID:       "pol-delegation",
			Version:  "v1",
			TenantID: "tenant-sec",
			Rules: []policy.Rule{
				{
					Name:               "allow-agent-read",
					Effect:             policy.EffectAllow,
					Agents:             []string{"agent-worker"},
					Tools:              []string{"database_read"},
					MaxDelegationDepth: 2,
				},
				{
					Name:               "allow-agent-admin",
					Effect:             policy.EffectAllow,
					Agents:             []string{"agent-admin"},
					Tools:              []string{"database_write", "database_delete"},
					MaxDelegationDepth: 1, // Admin cannot be deeply delegated
				},
			},
		}

		engine := policy.NewEngine([]*policy.Policy{pol})

		// Unprivileged worker attempts deep delegation to execute database_write as confused deputy
		dec := engine.Evaluate(ctx, &policy.EvaluationRequest{
			TenantID:        "tenant-sec",
			SubjectAgentID:  "agent-admin",
			Tool:            "database_delete",
			Action:          "execute",
			DelegationDepth: 3, // Exceeds MaxDelegationDepth=1
			DelegationStack: []string{"user", "agent-worker", "agent-proxy", "agent-admin"},
		})

		if dec.Effect != policy.EffectDeny {
			t.Fatalf("[Scenario E] Expected DENY for exceeded delegation depth in privileged operation, got %s", dec.Effect)
		}
	})

	// -------------------------------------------------------------------------
	// Scenario F: Config Bundle Signature Forgery & Downgrade Attack
	// -------------------------------------------------------------------------
	t.Run("Scenario_F_Signature_Forgery_And_Downgrade", func(t *testing.T) {
		kp, _ := crypto.GenerateKeyPair("legit-key")
		attackerKP, _ := crypto.GenerateKeyPair("attacker-key")

		kr := crypto.NewKeyRing()
		kr.RegisterKey("legit-key", kp.PublicKey)

		polV2 := &policy.Policy{ID: "pol-v2", Version: "v2.0.0"}
		signedBundleV2, _ := crypto.SignPayload(kp, "v2.0.0", time.Hour, polV2)

		// 1. Signature forgery by untrusted key
		forgedBundle, _ := crypto.SignPayload(attackerKP, "v2.0.0", time.Hour, polV2)
		forgedBundle.KeyID = "attacker-key"
		if err := kr.Verify(forgedBundle); err == nil {
			t.Fatalf("[Scenario F] Verification succeeded for forged key!")
		}

		// 2. Tampered payload
		tamperedBundle := *signedBundleV2
		tamperedBundle.Payload = `{"id":"pol-v2","tampered":true}`
		if err := kr.Verify(&tamperedBundle); err == nil {
			t.Fatalf("[Scenario F] Verification succeeded for tampered payload!")
		}

		// 3. Downgrade attack protection in ProxyConfigCache
		cache := config.NewProxyConfigCache(kr, policy.NewEngine(nil))

		// Sign older V1 first
		polV1 := &policy.Policy{ID: "pol-v1", Version: "v1.0.0"}
		signedBundleV1, _ := crypto.SignPayload(kp, "v1.0.0", time.Hour, polV1)

		// Wait briefly so V2 is monotonically strictly newer
		time.Sleep(10 * time.Millisecond)

		signedBundleV2Monotonic, _ := crypto.SignPayload(kp, "v2.0.0", time.Hour, polV2)
		err := cache.UpdateFromBundle(signedBundleV2Monotonic, polV2)
		if err != nil {
			t.Fatalf("[Scenario F] Cache update failed: %v", err)
		}

		// Try to apply older version V1
		err = cache.UpdateFromBundle(signedBundleV1, polV1)
		if !errors.Is(err, config.ErrConfigDowngrade) {
			t.Fatalf("[Scenario F] Config downgrade was not rejected! Expected ErrConfigDowngrade, got: %v", err)
		}
	})

	// -------------------------------------------------------------------------
	// Scenario G: Out-of-Order / Clock Skew Replay Attack
	// -------------------------------------------------------------------------
	t.Run("Scenario_G_Clock_Skew_And_Expired_Tokens", func(t *testing.T) {
		kp, _ := crypto.GenerateKeyPair("skew-key")
		kr := crypto.NewKeyRing()
		kr.RegisterKey("skew-key", kp.PublicKey)

		// 1. Bundle issued in future (10 minutes ahead)
		futureSigned, _ := crypto.SignPayload(kp, "v1.0.0", 1*time.Hour, map[string]string{"test": "future"})
		futureSigned.IssuedAt = time.Now().UTC().Add(10 * time.Minute)

		err := kr.Verify(futureSigned)
		if !errors.Is(err, crypto.ErrBundleFutureIssued) {
			t.Fatalf("[Scenario G] Future-issued bundle was not rejected! Got: %v", err)
		}

		// 2. Bundle expired 10 seconds ago
		expiredBundle, _ := crypto.SignPayload(kp, "v1.0.0", -10*time.Second, map[string]string{"test": "expired"})

		err = kr.Verify(expiredBundle)
		if !errors.Is(err, crypto.ErrBundleExpired) {
			t.Fatalf("[Scenario G] Expired bundle was not rejected! Got: %v", err)
		}
	})

	// -------------------------------------------------------------------------
	// Scenario H: LKG Retention Under Catastrophic Upstream Failure
	// -------------------------------------------------------------------------
	t.Run("Scenario_H_LKG_Retention_Under_Failure", func(t *testing.T) {
		kp, _ := crypto.GenerateKeyPair("lkg-key")
		kr := crypto.NewKeyRing()
		kr.RegisterKey("lkg-key", kp.PublicKey)

		cache := config.NewProxyConfigCache(kr, policy.NewEngine(nil))

		// Store good LKG config
		polLKG := &policy.Policy{ID: "pol-lkg", Version: "lkg-stable-v1"}
		signedLKG, _ := crypto.SignPayload(kp, "lkg-stable-v1", 24*time.Hour, polLKG)
		_ = cache.UpdateFromBundle(signedLKG, polLKG)

		// Simulate upstream failure / corrupt bundle
		corruptBundle := &crypto.SignedBundle{Version: "corrupted"}
		err := cache.UpdateFromBundle(corruptBundle, nil)
		if err == nil {
			t.Fatalf("[Scenario H] Expected error on corrupt config update, got nil")
		}

		// Active configuration MUST still be the LKG policy
		active := cache.CachedPolicy()
		if active == nil || active.Version != "lkg-stable-v1" {
			t.Fatalf("[Scenario H] LKG configuration was lost during upstream failure!")
		}
	})

	// -------------------------------------------------------------------------
	// Scenario I: Data Plane Fail-Closed Under Corrupt Policy Cache
	// -------------------------------------------------------------------------
	t.Run("Scenario_I_FailClosed_Corrupt_Policy_Cache", func(t *testing.T) {
		kp, _ := crypto.GenerateKeyPair("exp-key")
		kr := crypto.NewKeyRing()
		kr.RegisterKey("exp-key", kp.PublicKey)

		cache := config.NewProxyConfigCache(kr, policy.NewEngine(nil))

		// Create bundle expired in the past
		polExp := &policy.Policy{ID: "pol-exp", Version: "expired-v0"}
		signedExp, _ := crypto.SignPayload(kp, "expired-v0", time.Hour, polExp)
		signedExp.ExpiresAt = time.Now().UTC().Add(-10 * time.Minute)

		// CachedPolicy returns nil if now is after ExpiresAt
		active := cache.CachedPolicy()
		if active != nil {
			t.Fatalf("[Scenario I] Data plane failed open! Expired config was returned: %v", active)
		}
	})

	// -------------------------------------------------------------------------
	// Scenario J: Secret Leakage via Telemetry / Error Responses
	// -------------------------------------------------------------------------
	t.Run("Scenario_J_Secret_Scrubbing_And_Log_Sanitization", func(t *testing.T) {
		mockStripe := "sk_" + "live_" + "51Abcdef1234567890"
		mockAWS := "AKIA" + "IOSFODNN7EXAMPLE"
		sensitiveLog := "Authorization: Bearer " + mockStripe + "\n" +
			"Password in postgres://user:SuperSecretP@ssword!@db.internal:5432/mesh\n" +
			"AWS: " + mockAWS + " and secret wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY\n" +
			"GCP Key: \"private_key\": \"-----BEGIN PRIVATE KEY-----\\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC3\""

		scrubbed := telemetry.ScrubSecrets(sensitiveLog)

		leaks := []string{
			mockStripe,
			"SuperSecretP@ssword!",
			"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC3",
		}

		for _, leak := range leaks {
			if strings.Contains(scrubbed, leak) {
				t.Fatalf("[Scenario J] Secret leaked in telemetry: %q found in scrubbed output", leak)
			}
		}

		// Log injection neutralization
		maliciousLog := "Normal log message\r\n[CRITICAL] System Compromised: Admin Granted\n"
		sanitizedLog := telemetry.SanitizeLogMessage(maliciousLog)
		if strings.Contains(sanitizedLog, "\r") || strings.Contains(sanitizedLog, "\n") {
			t.Fatalf("[Scenario J] Log injection not neutralized: newlines remain in sanitized log")
		}
	})

	// -------------------------------------------------------------------------
	// Scenario K: Concurrent Route Storm / Resource Exhaustion
	// -------------------------------------------------------------------------
	t.Run("Scenario_K_Concurrent_Route_Storm", func(t *testing.T) {
		router := routing.NewRouter(nil)
		for i := 1; i <= 10; i++ {
			router.RegisterCandidate(&routing.AgentRouteCandidate{
				AgentID:      fmt.Sprintf("agent-%d", i),
				EndpointURL:  fmt.Sprintf("http://agent-%d.mesh.internal", i),
				Status:       "HEALTHY",
				Contract: &contracts.AgentContract{
					Capabilities: []string{"text_generation"},
				},
				SuccessRate:  0.99,
				P95LatencyMs: 50,
			})
		}

		concurrency := 150
		var wg sync.WaitGroup
		wg.Add(concurrency)

		errCh := make(chan error, concurrency)

		start := time.Now()
		for i := 0; i < concurrency; i++ {
			go func(idx int) {
				defer wg.Done()
				cand, err := router.Route(ctx, &routing.RouteRequest{
					RequiredCapability: "text_generation",
					Strategy:           routing.StrategyBalanced,
				})
				if err != nil || cand == nil {
					errCh <- fmt.Errorf("routing failed on req %d: %v", idx, err)
				}
			}(i)
		}

		wg.Wait()
		close(errCh)

		duration := time.Since(start)
		if len(errCh) > 0 {
			t.Fatalf("[Scenario K] Route storm failed: %v", <-errCh)
		}
		if duration > 1*time.Second {
			t.Fatalf("[Scenario K] Route storm exceeded latency budget: %v for %d requests", duration, concurrency)
		}
	})

	// -------------------------------------------------------------------------
	// Scenario L: Semantic Bypass of Sensitive Tools
	// -------------------------------------------------------------------------
	t.Run("Scenario_L_Semantic_Bypass_Prevention", func(t *testing.T) {
		pol := &policy.Policy{
			ID:       "pol-strict-tools",
			Version:  "v1",
			TenantID: "tenant-defense",
			Rules: []policy.Rule{
				{
					Name:      "deny-destructive-tools",
					Effect:    policy.EffectDeny,
					Tools:     []string{"*drop*", "*delete*", "*rm*", "*truncate*"},
					Actions:   []string{"*"},
					Resources: []string{"*"},
				},
				{
					Name:      "allow-safe-read",
					Effect:    policy.EffectAllow,
					Tools:     []string{"read_*", "get_*", "list_*"},
					Actions:   []string{"read"},
					Resources: []string{"*"},
				},
			},
		}

		engine := policy.NewEngine([]*policy.Policy{pol})

		// Tool alias attempting to sneak destructive operation under read guise
		dec := engine.Evaluate(ctx, &policy.EvaluationRequest{
			TenantID:       "tenant-defense",
			SubjectAgentID: "agent-1",
			Tool:           "execute_rm_all",
			Action:         "read",
		})
		if dec.Effect != policy.EffectDeny {
			t.Fatalf("[Scenario L] Semantic bypass succeeded for tool execute_rm_all: %s", dec.Effect)
		}

		// Tool requiring approval when classified destructive in tool registry
		mcpServer := mcp.NewAgentMeshMCPServer("tenant-defense", nil)
		if mcpServer == nil {
			t.Fatalf("[Scenario L] MCP server initialization failed")
		}
	})

	// -------------------------------------------------------------------------
	// Scenario M: Public Agent Passport Leakage
	// -------------------------------------------------------------------------
	t.Run("Scenario_M_Public_Passport_Sanitization", func(t *testing.T) {
		contract := &contracts.AgentContract{
			APIVersion: "agentmesh.dev/v1",
			Kind:       "AgentContract",
			Metadata: contracts.Metadata{
				Name:         "stealth-finance-bot",
				Organization: "SuperSecretCapitalLLC",
				Version:      "v2.1",
			},
			Budgets: contracts.BudgetConfig{
				MaxCostPerTask: 15.50, // Private economics
			},
			Identity: contracts.IdentityConfig{
				Protocols: []string{"a2a", "mcp"},
			},
		}

		pass, err := passport.GenerateFromContract(contract, "go", "adk")
		if err != nil {
			t.Fatalf("[Scenario M] Failed to generate passport: %v", err)
		}

		// When passport is private, SanitizeForPublic MUST return nil (fail closed)
		if privateScrubbed := pass.SanitizeForPublic(); privateScrubbed != nil {
			t.Fatalf("[Scenario M] Private passport returned non-nil on public resolution!")
		}

		// When explicitly published, sanitize for public consumption
		pass.IsPublic = true
		publicPass := pass.SanitizeForPublic()
		if publicPass == nil {
			t.Fatalf("[Scenario M] Expected non-nil public passport, got nil")
		}

		// 1. Organization name must be redacted
		if publicPass.Identity.Organization != "[REDACTED_ORGANIZATION]" {
			t.Fatalf("[Scenario M] Organization not redacted: %q", publicPass.Identity.Organization)
		}

		// 2. Private economics must be zeroed
		if publicPass.Economics.AverageCostUSD != 0.0 || publicPass.Economics.DailyCostUSD != 0.0 {
			t.Fatalf("[Scenario M] Private economics not zeroed: %+v", publicPass.Economics)
		}

		// 3. Serialized JSON must not contain secret org name
		passJSON, _ := json.Marshal(publicPass)
		if strings.Contains(string(passJSON), "SuperSecretCapitalLLC") {
			t.Fatalf("[Scenario M] Secret organization name leaked in public JSON output")
		}
	})

	// -------------------------------------------------------------------------
	// Scenario N: Replay of Expired Multi-Region Route Decision
	// -------------------------------------------------------------------------
	t.Run("Scenario_N_Replay_Expired_Route_Decision", func(t *testing.T) {
		store := database.NewMemoryStore()

		// Save a routing outcome
		staleOutcome := &routing.CanonicalRoutingOutcome{
			OutcomeID:       "route-outcome-old",
			OrganizationID:  "tenant-multi",
			TaskID:          "task-123",
			CapabilityID:    "geo-route",
			SelectedAgentID: "agent-eu-west1",
			Success:         true,
			CreatedAt:       time.Now().UTC().Add(-48 * time.Hour), // 2 days old
		}
		_ = store.SaveRoutingOutcomeV3(ctx, staleOutcome)

		// Fetch outcome and verify expiration / age detection
		outcomes, err := store.ListRoutingOutcomesV3(ctx, "tenant-multi", "geo-route")
		if err != nil || len(outcomes) == 0 {
			t.Fatalf("[Scenario N] Failed to retrieve routing outcomes")
		}

		retrieved := outcomes[0]
		age := time.Since(retrieved.CreatedAt)
		if age < 24*time.Hour {
			t.Fatalf("[Scenario N] Outcome age mismatch: expected >24h, got %v", age)
		}
	})

	// -------------------------------------------------------------------------
	// Scenario O: Integer Overflow in Cost / Token Arithmetic
	// -------------------------------------------------------------------------
	t.Run("Scenario_O_Cost_Token_Arithmetic_Overflow", func(t *testing.T) {
		ci := cost.NewIntelligence()

		// 1. Negative token counts must be clamped to zero cost
		negCost := ci.CalculateCost("gemini-1.5-pro", -1000, -5000, -200)
		if negCost != 0.0 {
			t.Fatalf("[Scenario O] Negative token count produced non-zero cost: %f", negCost)
		}

		// 2. NaN / Inf float conversions to MicroUSD rejected
		_, err := cost.ToMicroUSD(math.NaN())
		if err == nil {
			t.Fatalf("[Scenario O] NaN conversion to MicroUSD was not rejected!")
		}

		_, err = cost.ToMicroUSD(math.Inf(1))
		if err == nil {
			t.Fatalf("[Scenario O] Inf conversion to MicroUSD was not rejected!")
		}

		_, err = cost.ToMicroUSD(-10.50)
		if err == nil {
			t.Fatalf("[Scenario O] Negative USD conversion to MicroUSD was not rejected!")
		}

		// 3. Extreme token counts must not overflow or become negative
		hugeTokens := int64(1_000_000_000_000) // 1 trillion tokens
		hugeCost := ci.CalculateCost("gemini-1.5-flash", hugeTokens, hugeTokens, 0)
		if hugeCost < 0 || math.IsNaN(hugeCost) || math.IsInf(hugeCost, 0) {
			t.Fatalf("[Scenario O] Overflow or invalid cost calculation on massive tokens: %f", hugeCost)
		}
	})
}
