package identity_test

import (
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/identity"
)

func TestAPIKeyGenerationAndValidation(t *testing.T) {
	rawKey, cred, err := identity.GenerateAPIKey(
		"tenant_acme",
		"agent_finance",
		[]string{identity.ScopeAgentsInvoke, identity.ScopePoliciesRead},
		1*time.Hour,
		"Test finance key",
	)
	if err != nil {
		t.Fatalf("failed to generate API key: %v", err)
	}

	if err := cred.ValidateKey(rawKey); err != nil {
		t.Errorf("validation failed with correct key: %v", err)
	}

	if err := cred.ValidateKey("mesh_invalid_key"); err == nil {
		t.Error("expected validation to fail for invalid key, got nil")
	}

	if !cred.HasScope(identity.ScopeAgentsInvoke) {
		t.Errorf("expected credential to have %s", identity.ScopeAgentsInvoke)
	}
	if cred.HasScope(identity.ScopePoliciesWrite) {
		t.Errorf("expected credential to NOT have %s", identity.ScopePoliciesWrite)
	}
}

func TestRevocationAndExpiry(t *testing.T) {
	rawKey, cred, _ := identity.GenerateAPIKey(
		"tenant_acme",
		"agent_finance",
		[]string{identity.ScopeAdmin},
		-1*time.Minute, // Expired
		"Expired key",
	)

	if err := cred.ValidateKey(rawKey); err == nil {
		t.Error("expected error for expired key, got nil")
	}

	// Revocation
	rawKey2, cred2, _ := identity.GenerateAPIKey(
		"tenant_acme",
		"agent_finance",
		[]string{identity.ScopeAdmin},
		1*time.Hour,
		"Revoked key",
	)
	cred2.Revoked = true
	if err := cred2.ValidateKey(rawKey2); err == nil {
		t.Error("expected error for revoked key, got nil")
	}
}
