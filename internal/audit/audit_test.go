package audit_test

import (
	"strings"
	"testing"

	"github.com/agentmesh/agentmesh/internal/audit"
)

func TestAuditLogger_BasicAndChaining(t *testing.T) {
	logger := audit.NewLogger()
	tenantID := "tenant-alpha"

	// Initial empty list
	entries := logger.List(tenantID, 10)
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries initially, got %d", len(entries))
	}

	// Verify empty chain
	ok, err := logger.VerifyChain(tenantID)
	if !ok || err != nil {
		t.Fatalf("expected empty chain to verify, got ok=%v, err=%v", ok, err)
	}

	// Add entries
	e1 := logger.Log(tenantID, audit.EventAgentRegistered, "user:alice", "agent:finance", map[string]string{"version": "v1"})
	if e1 == nil || e1.ID == "" || e1.TenantID != tenantID {
		t.Fatalf("invalid entry 1: %+v", e1)
	}
	if !strings.HasPrefix(e1.ID, "aud_") {
		t.Errorf("expected ID prefix 'aud_', got %s", e1.ID)
	}
	if e1.PrevHash != "0000000000000000000000000000000000000000000000000000000000000000" {
		t.Errorf("expected genesis prevHash, got %s", e1.PrevHash)
	}

	e2 := logger.Log(tenantID, audit.EventPolicyPublished, "user:bob", "policy:default", nil)
	if e2.PrevHash != e1.Hash {
		t.Errorf("expected e2.PrevHash == e1.Hash, got %s != %s", e2.PrevHash, e1.Hash)
	}

	e3 := logger.Log(tenantID, audit.EventToolInvoked, "agent:finance", "tool:bigquery.read", "query executed")
	if e3.PrevHash != e2.Hash {
		t.Errorf("expected e3.PrevHash == e2.Hash, got %s != %s", e3.PrevHash, e2.Hash)
	}

	// Verify chain integrity
	valid, err := logger.VerifyChain(tenantID)
	if !valid || err != nil {
		t.Fatalf("chain verification failed: %v", err)
	}

	// Test List with various limits
	all := logger.List(tenantID, -1)
	if len(all) != 3 {
		t.Errorf("expected 3 entries with limit=-1, got %d", len(all))
	}

	zeroLimit := logger.List(tenantID, 0)
	if len(zeroLimit) != 3 {
		t.Errorf("expected 3 entries with limit=0, got %d", len(zeroLimit))
	}

	lastTwo := logger.List(tenantID, 2)
	if len(lastTwo) != 2 {
		t.Errorf("expected 2 entries, got %d", len(lastTwo))
	}
	if lastTwo[0].ID != e2.ID || lastTwo[1].ID != e3.ID {
		t.Errorf("expected last two entries to be e2 and e3")
	}

	largeLimit := logger.List(tenantID, 100)
	if len(largeLimit) != 3 {
		t.Errorf("expected 3 entries with limit=100, got %d", len(largeLimit))
	}
}

func TestAuditLogger_TamperDetection(t *testing.T) {
	logger := audit.NewLogger()
	tenantID := "tenant-beta"

	logger.Log(tenantID, audit.EventCredentialCreated, "admin", "cred-1", nil)
	logger.Log(tenantID, audit.EventApprovalRequested, "agent-1", "tool-drop", nil)
	logger.Log(tenantID, audit.EventApprovalResolved, "admin", "tool-drop", "APPROVED")

	valid, err := logger.VerifyChain(tenantID)
	if !valid || err != nil {
		t.Fatalf("expected valid chain, got %v, %v", valid, err)
	}

	// Tamper with second entry content directly
	entries := logger.List(tenantID, 3)
	entries[1].Subject = "tool-read-modified"

	// Chain verification should fail on tampered record
	valid, err = logger.VerifyChain(tenantID)
	if valid || err == nil {
		t.Fatalf("expected tamper detection to fail verification, got valid=%v", valid)
	}
	if !strings.Contains(err.Error(), "tampering detected") && !strings.Contains(err.Error(), "hash chain broken") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestAuditLogger_MultiTenantIsolation(t *testing.T) {
	logger := audit.NewLogger()

	logger.Log("tenant-1", audit.EventAgentRegistered, "user", "agent-1", nil)
	logger.Log("tenant-1", audit.EventAgentUpdated, "user", "agent-1", nil)

	logger.Log("tenant-2", audit.EventCanaryStarted, "operator", "agent-2", nil)

	t1Entries := logger.List("tenant-1", 10)
	t2Entries := logger.List("tenant-2", 10)

	if len(t1Entries) != 2 {
		t.Errorf("expected 2 entries for tenant-1, got %d", len(t1Entries))
	}
	if len(t2Entries) != 1 {
		t.Errorf("expected 1 entry for tenant-2, got %d", len(t2Entries))
	}

	ok1, err1 := logger.VerifyChain("tenant-1")
	if !ok1 || err1 != nil {
		t.Errorf("tenant-1 chain invalid: %v", err1)
	}

	ok2, err2 := logger.VerifyChain("tenant-2")
	if !ok2 || err2 != nil {
		t.Errorf("tenant-2 chain invalid: %v", err2)
	}
}
