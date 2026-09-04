package tests_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/approval"
	"github.com/agentmesh/agentmesh/internal/budgets"
	"github.com/agentmesh/agentmesh/internal/canary"
	"github.com/agentmesh/agentmesh/internal/config"
	"github.com/agentmesh/agentmesh/internal/crypto"
	"github.com/agentmesh/agentmesh/internal/delegation"
	"github.com/agentmesh/agentmesh/internal/identity"
	"github.com/agentmesh/agentmesh/internal/mcp"
	"github.com/agentmesh/agentmesh/internal/policy"
	"github.com/agentmesh/agentmesh/internal/reliability"
	"github.com/agentmesh/agentmesh/internal/routing"
	"github.com/agentmesh/agentmesh/internal/telemetry"
	"github.com/agentmesh/agentmesh/pkg/contracts"
	"github.com/agentmesh/agentmesh/pkg/protocol"
)

// Invariant 1: Denied tool cannot execute through proxy.
func TestInvariant1_DeniedToolCannotExecute(t *testing.T) {
	pol := &policy.Policy{
		ID:       "pol_1",
		TenantID: "tenant_corp",
		Rules: []policy.Rule{
			{
				Name:    "Deny payment execution",
				Effect:  policy.EffectDeny,
				Agents:  []string{"*"},
				Tools:   []string{"payment.execute"},
				Actions: []string{"execute"},
			},
		},
	}
	engine := policy.NewEngine([]*policy.Policy{pol})
	gateway := mcp.NewGateway(engine, nil, nil, nil)

	reqBytes, _ := json.Marshal(protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"payment.execute"}`),
	})

	resp := gateway.HandleRPC(context.Background(), "tenant_corp", "agent-a", reqBytes)
	if resp.Error == nil || resp.Error.Code != protocol.MCPPolicyDenied {
		t.Fatalf("expected MCPPolicyDenied (-32001), got: %+v", resp.Error)
	}
}

// Invariant 2: Approval-required tool cannot execute before approval.
func TestInvariant2_ApprovalRequiredBlockedBeforeApproval(t *testing.T) {
	pol := &policy.Policy{
		ID:       "pol_2",
		TenantID: "tenant_corp",
		Rules: []policy.Rule{
			{
				Name:    "Require approval for delete",
				Effect:  policy.EffectRequireApproval,
				Agents:  []string{"*"},
				Tools:   []string{"data.delete"},
				Actions: []string{"execute"},
			},
		},
	}
	engine := policy.NewEngine([]*policy.Policy{pol})
	appSvc := approval.NewService()
	gateway := mcp.NewGateway(engine, appSvc, nil, nil)

	reqBytes, _ := json.Marshal(protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"data.delete","arguments":{"key":"val"}}`),
	})

	// Call without token -> must return MCPApprovalRequired
	resp := gateway.HandleRPC(context.Background(), "tenant_corp", "agent-a", reqBytes)
	if resp.Error == nil || resp.Error.Code != protocol.MCPApprovalRequired {
		t.Fatalf("expected MCPApprovalRequired (-32002), got: %+v", resp.Error)
	}
}

// Invariant 3: Agent A cannot access Tenant B.
func TestInvariant3_CrossTenantIsolation(t *testing.T) {
	pol := &policy.Policy{
		ID:       "pol_tenant_a",
		TenantID: "tenant_a",
		Rules: []policy.Rule{
			{
				Name:    "Allow tool for tenant A",
				Effect:  policy.EffectAllow,
				Agents:  []string{"agent-1"},
				Tools:   []string{"secret.tool"},
				Actions: []string{"read"},
			},
		},
	}
	engine := policy.NewEngine([]*policy.Policy{pol})

	dec := engine.Evaluate(context.Background(), &policy.EvaluationRequest{
		TenantID:       "tenant_b", // Cross tenant attempt
		SubjectAgentID: "agent-1",
		Tool:           "secret.tool",
		Action:         "read",
	})
	if dec.Effect != policy.EffectDeny {
		t.Fatalf("cross-tenant invariant breached: got %s, expected DENY", dec.Effect)
	}
}

// Invariant 4: Agent A cannot gain privilege through delegation.
func TestInvariant4_DelegationPrivilegeEscalationPrevented(t *testing.T) {
	pol := &policy.Policy{
		ID:       "pol_security",
		TenantID: "corp",
		Rules: []policy.Rule{
			{
				Name:    "Allow agent-privileged payment",
				Effect:  policy.EffectAllow,
				Agents:  []string{"agent-privileged"},
				Tools:   []string{"wire.transfer"},
				Actions: []string{"transfer"},
			},
			{
				Name:    "Deny agent-unprivileged payment",
				Effect:  policy.EffectDeny,
				Agents:  []string{"agent-unprivileged"},
				Tools:   []string{"wire.transfer"},
				Actions: []string{"transfer"},
			},
		},
	}
	engine := policy.NewEngine([]*policy.Policy{pol})

	chain := delegation.NewChain("agent-unprivileged", 5)
	_ = chain.Push("agent-privileged", "transfer", nil)

	err := chain.CheckPrivilegeEscalation(engine, "corp", "wire.transfer", "transfer")
	if !errors.Is(err, delegation.ErrPrivilegeEscalation) {
		t.Fatalf("delegation privilege escalation invariant breached: got %v", err)
	}
}

// Invariant 5: Delegation cycle terminates.
func TestInvariant5_DelegationCycleTerminates(t *testing.T) {
	chain := delegation.NewChain("agent-a", 5)
	_ = chain.Push("agent-b", "help", nil)
	_ = chain.Push("agent-c", "help", nil)

	// Attempting cycle: c -> a
	err := chain.Push("agent-a", "cycle", nil)
	if !errors.Is(err, delegation.ErrCycleDetected) {
		t.Fatalf("expected ErrCycleDetected, got: %v", err)
	}
}

// Invariant 6: Budget overflow stops further calls.
func TestInvariant6_BudgetOverflowStopsExecution(t *testing.T) {
	tracker := budgets.NewTracker()
	budget := budgets.TaskBudget{
		MaxCostUSD: 0.05,
		MaxTokens:  10000,
	}
	usage := &budgets.TaskUsage{}

	// Incremental spend 1: OK
	err := tracker.CheckAndRecordSpend("tenant-1", budget, usage, 500, 0.02, 1)
	if err != nil {
		t.Fatalf("unexpected budget error: %v", err)
	}

	// Incremental spend 2: Exceeds cost ceiling ($0.02 + $0.04 = $0.06 > $0.05)
	err = tracker.CheckAndRecordSpend("tenant-1", budget, usage, 100, 0.04, 1)
	if !errors.Is(err, budgets.ErrCostBudgetExceeded) {
		t.Fatalf("expected ErrCostBudgetExceeded, got: %v", err)
	}
}

// Invariant 7: Route never selects policy-ineligible agent.
func TestInvariant7_RouteNeverSelectsPolicyIneligible(t *testing.T) {
	pol := &policy.Policy{
		ID:       "pol_routing",
		TenantID: "corp",
		Rules: []policy.Rule{
			{
				Name:      "Explicit deny agent-evil from being invoked",
				Effect:    policy.EffectDeny,
				Agents:    []string{"*"},
				Actions:   []string{"invoke"},
				Resources: []string{"agent-evil"},
			},
		},
	}
	engine := policy.NewEngine([]*policy.Policy{pol})
	router := routing.NewRouter(engine)

	router.RegisterCandidate(&routing.AgentRouteCandidate{
		AgentID:     "agent-evil",
		Status:      "HEALTHY",
		Contract:    &contracts.AgentContract{Capabilities: []string{"search"}},
		AverageCost: 0.001,
	})

	_, err := router.Route(context.Background(), &routing.RouteRequest{
		TenantID:           "corp",
		CallerAgentID:      "user-agent",
		RequiredCapability: "search",
	})
	if err == nil {
		t.Fatal("invariant breached: policy-ineligible agent was selected")
	}
}

// Invariant 8: Invalid signed config rejected.
func TestInvariant8_InvalidSignedConfigRejected(t *testing.T) {
	kp, _ := crypto.GenerateKeyPair("key_1")
	keyRing := crypto.NewKeyRing()
	keyRing.RegisterKey("key_1", kp.PublicKey)

	bundle, _ := crypto.SignPayload(kp, "v1", 1*time.Hour, map[string]string{"foo": "bar"})

	// Tamper payload
	bundle.Payload = `{"foo":"hacked"}`
	err := keyRing.Verify(bundle)
	if err == nil {
		t.Fatal("tampered config bundle was not rejected")
	}
}

// Invariant 9: Proxy continues using last valid config after control-plane outage.
func TestInvariant9_ProxyContinuesUsingCachedConfigDuringOutage(t *testing.T) {
	kp, _ := crypto.GenerateKeyPair("key_1")
	keyRing := crypto.NewKeyRing()
	keyRing.RegisterKey("key_1", kp.PublicKey)

	pol := &policy.Policy{
		ID:       "pol_cached",
		TenantID: "corp",
		Rules: []policy.Rule{
			{Name: "Allow cached", Effect: policy.EffectAllow, Agents: []string{"*"}, Tools: []string{"tool.cached"}},
		},
	}
	engine := policy.NewEngine([]*policy.Policy{})
	cache := config.NewProxyConfigCache(keyRing, engine)

	bundle, _ := crypto.SignPayload(kp, "v1", 24*time.Hour, pol)
	_ = cache.UpdateFromBundle(bundle, pol)

	// Verify policy is in engine
	dec := engine.Evaluate(context.Background(), &policy.EvaluationRequest{
		TenantID:       "corp",
		SubjectAgentID: "agent-1",
		Tool:           "tool.cached",
	})
	if dec.Effect != policy.EffectAllow {
		t.Fatalf("expected ALLOW from cached policy, got: %s", dec.Effect)
	}

	// Simulate control plane outage (attempting invalid bundle does NOT overwrite last known good)
	badBundle := &crypto.SignedBundle{KeyID: "untrusted_key"}
	_ = cache.UpdateFromBundle(badBundle, nil)

	// Engine still has last valid policy
	dec2 := engine.Evaluate(context.Background(), &policy.EvaluationRequest{
		TenantID:       "corp",
		SubjectAgentID: "agent-1",
		Tool:           "tool.cached",
	})
	if dec2.Effect != policy.EffectAllow {
		t.Fatalf("invariant breached: proxy lost last known good config during outage")
	}
}

// Invariant 10: Secret never logged / scrubbed from traces.
func TestInvariant10_SecretsScrubbedFromTelemetry(t *testing.T) {
	raw := "Internal error with bearer sk-abcdef12345678901234567890123456 and AIzaSyD9876543210987654321098765432109"
	scrubbed := telemetry.ScrubSecrets(raw)

	if scrubbed == raw {
		t.Fatal("secret scrubbing failed to alter sensitive text")
	}
	if !telemetryScrubCheck(scrubbed) {
		t.Fatalf("secrets leaked in output: %s", scrubbed)
	}
}

func telemetryScrubCheck(s string) bool {
	return !containsAny(s, "sk-abcdef", "AIzaSy")
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	return false
}

// Invariant 11: Retry does not duplicate unsafe tool action.
func TestInvariant11_NonRetryableOperationsNeverDuplicated(t *testing.T) {
	calls := 0
	_ = reliability.ExecuteWithRetry(context.Background(), reliability.NonRetryable, reliability.DefaultRetryPolicy, func(ctx context.Context, attempt int) error {
		calls++
		return errors.New("write error")
	})

	if calls != 1 {
		t.Fatalf("invariant breached: non-retryable action was executed %d times", calls)
	}
}

// Invariant 12: Canary rollback restores previous agent version.
func TestInvariant12_CanaryRollbackRestoresPreviousVersion(t *testing.T) {
	mgr := canary.NewManager()
	_, _ = mgr.StartCanary("agent-x", "v1.0.0", "v2.0.0-rc", 10, false, 0.05, 1000)

	// Inject error rate regression
	for i := 0; i < 5; i++ {
		_, _ = mgr.RecordCandidateSample("agent-x", false, 1500, 0.01)
	}

	active := mgr.GetActiveVersion("agent-x")
	if active != "v1.0.0" {
		t.Fatalf("expected rollback to restore v1.0.0, got: %s", active)
	}
}

// Invariant 13: Disabled agent receives no new traffic.
func TestInvariant13_DisabledAgentExcludedFromRouting(t *testing.T) {
	router := routing.NewRouter(nil)
	router.RegisterCandidate(&routing.AgentRouteCandidate{
		AgentID:     "agent-dead",
		Status:      "DISABLED",
		Contract:    &contracts.AgentContract{Capabilities: []string{"work"}},
		AverageCost: 0.001,
	})

	_, err := router.Route(context.Background(), &routing.RouteRequest{
		RequiredCapability: "work",
	})
	if err == nil {
		t.Fatal("disabled agent was routed traffic")
	}
}

// Invariant 14: Expired credential fails.
func TestInvariant14_ExpiredCredentialFails(t *testing.T) {
	rawKey, cred, _ := identity.GenerateAPIKey("tenant", "agent", []string{identity.ScopeAdmin}, -5*time.Second, "expired")
	err := cred.ValidateKey(rawKey)
	if err == nil {
		t.Fatal("expired credential succeeded validation")
	}
}

// Invariant 15: API key scope enforced.
func TestInvariant15_APIKeyScopeEnforced(t *testing.T) {
	_, cred, _ := identity.GenerateAPIKey("tenant", "agent", []string{identity.ScopeAgentsRead}, 1*time.Hour, "read-only")

	if !cred.HasScope(identity.ScopeAgentsRead) {
		t.Fatal("expected ScopeAgentsRead")
	}
	if cred.HasScope(identity.ScopePoliciesWrite) {
		t.Fatal("invariant breached: read-only key granted write access")
	}
}
