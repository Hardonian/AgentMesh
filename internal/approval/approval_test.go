package approval_test

import (
	"errors"
	"testing"
	"time"

	"github.com/agentmesh/agentmesh/internal/approval"
)

func TestApprovalWorkflow(t *testing.T) {
	svc := approval.NewService()

	params := map[string]any{
		"recipient": "vendor@corp.internal",
		"amountUSD": 15000.0,
	}

	// 1. Create request
	req, err := svc.CreateRequest("tenant_acme", "procurement-agent", "internal.erp.payment", "execute", params, "pol_1", "v1", "Payment above threshold", 10*time.Minute)
	if err != nil {
		t.Fatalf("failed to create approval request: %v", err)
	}

	if req.Status != approval.StatusPending {
		t.Errorf("expected PENDING, got %s", req.Status)
	}

	// 2. Resolve request with approval
	resolved, err := svc.Resolve(req.ID, "user_compliance_officer", true, "Approved for Q3 procurement")
	if err != nil {
		t.Fatalf("failed to resolve request: %v", err)
	}
	if resolved.Status != approval.StatusApproved {
		t.Errorf("expected APPROVED, got %s", resolved.Status)
	}
	if resolved.ApprovalToken == "" {
		t.Fatal("expected non-empty approval token")
	}

	// 3. Validation with identical parameters should succeed
	err = svc.ValidateApproval(req.ID, "procurement-agent", "internal.erp.payment", params, resolved.ApprovalToken)
	if err != nil {
		t.Errorf("expected approval validation to succeed, got: %v", err)
	}

	// 4. Critical Invariant: Tampering with parameters MUST fail validation
	tamperedParams := map[string]any{
		"recipient": "attacker@evil.com", // Altered recipient!
		"amountUSD": 15000.0,
	}
	err = svc.ValidateApproval(req.ID, "procurement-agent", "internal.erp.payment", tamperedParams, resolved.ApprovalToken)
	if !errors.Is(err, approval.ErrApprovalTampered) {
		t.Errorf("expected ErrApprovalTampered on altered parameters, got: %v", err)
	}

	// 5. Tenant Isolation: Foreign tenant MUST NOT be able to use approval
	err = svc.ValidateApprovalForTenant(req.ID, "tenant_hostile", "procurement-agent", "internal.erp.payment", params, resolved.ApprovalToken)
	if !errors.Is(err, approval.ErrApprovalTenantMismatch) {
		t.Errorf("expected ErrApprovalTenantMismatch for foreign tenant, got: %v", err)
	}

	// 6. Replay Prevention: Once consumed, token cannot be re-used
	err = svc.ConsumeApproval(req.ID, "tenant_acme", "procurement-agent", "internal.erp.payment", params, resolved.ApprovalToken)
	if err != nil {
		t.Fatalf("first consumption should succeed, got: %v", err)
	}
	err = svc.ConsumeApproval(req.ID, "tenant_acme", "procurement-agent", "internal.erp.payment", params, resolved.ApprovalToken)
	if !errors.Is(err, approval.ErrApprovalConsumed) {
		t.Errorf("expected ErrApprovalConsumed on token reuse, got: %v", err)
	}

	// 7. Expiration Invariant: Expired approval must be rejected
	expiredReq, _ := svc.CreateRequest("tenant_acme", "agent-x", "tool-x", "execute", nil, "pol_1", "v1", "Test Expire", 1*time.Millisecond)
	resolvedExp, _ := svc.Resolve(expiredReq.ID, "reviewer", true, "ok")
	time.Sleep(5 * time.Millisecond) // Let TTL expire
	err = svc.ValidateApproval(expiredReq.ID, "agent-x", "tool-x", nil, resolvedExp.ApprovalToken)
	if !errors.Is(err, approval.ErrApprovalExpired) {
		t.Errorf("expected ErrApprovalExpired for expired token, got: %v", err)
	}
}
