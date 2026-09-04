package identity_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/identity"
)

func TestOIDCValidator_ValidateAndMapRoles(t *testing.T) {
	validator := identity.NewOIDCValidator(nil)

	// Create test JWT payload
	claims := map[string]any{
		"sub":    "usr_google_12345",
		"email":  "approver@acme.com",
		"groups": []string{"finance-admins"},
		"iss":    "https://accounts.google.com",
		"exp":    time.Now().Add(1 * time.Hour).Unix(),
	}
	payloadBytes, _ := json.Marshal(claims)
	fakeJWT := "header." + base64.RawURLEncoding.EncodeToString(payloadBytes) + ".signature"

	parsed, err := validator.ValidateIDToken(context.Background(), fakeJWT)
	if err != nil {
		t.Fatalf("ValidateIDToken failed: %v", err)
	}

	if parsed.Email != "approver@acme.com" {
		t.Fatalf("expected email approver@acme.com, got: %s", parsed.Email)
	}

	hasApprover := false
	for _, r := range parsed.MappedRoles {
		if r == "ROLE_APPROVER" {
			hasApprover = true
			break
		}
	}
	if !hasApprover {
		t.Fatalf("expected ROLE_APPROVER in mapped roles: %v", parsed.MappedRoles)
	}
}

func TestOIDCValidator_ApprovalTokenLifecycle(t *testing.T) {
	validator := identity.NewOIDCValidator(nil)

	tok, err := validator.SignApprovalToken("appr_999", "admin@acme.com", []string{"ROLE_APPROVER"}, 5*time.Minute)
	if err != nil {
		t.Fatalf("SignApprovalToken failed: %v", err)
	}

	valid, err := validator.VerifyApprovalToken(tok)
	if err != nil || !valid {
		t.Fatalf("expected valid approval token, err: %v", err)
	}

	// Tampered token must fail
	tok.ApproverEmail = "attacker@evil.com"
	valid, err = validator.VerifyApprovalToken(tok)
	if err == nil || valid {
		t.Fatal("tampered approval token must fail validation")
	}
}
