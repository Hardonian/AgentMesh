package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EventType represents the category of audited action.
type EventType string

const (
	EventAgentRegistered    EventType = "agent.registered"
	EventAgentUpdated       EventType = "agent.updated"
	EventAgentDisabled      EventType = "agent.disabled"
	EventCredentialCreated  EventType = "credential.created"
	EventCredentialRevoked  EventType = "credential.revoked"
	EventPolicyPublished    EventType = "policy.published"
	EventRouteDecision      EventType = "route.decision"
	EventToolInvoked        EventType = "tool.invoked"
	EventApprovalRequested  EventType = "approval.requested"
	EventApprovalResolved   EventType = "approval.resolved"
	EventCanaryStarted      EventType = "canary.started"
	EventCanaryPromoted     EventType = "canary.promoted"
	EventCanaryRolledBack   EventType = "canary.rolled_back"
	EventEvaluationFinished EventType = "evaluation.completed"
)

// Entry is an append-only audit record in a tamper-evident chain.
type Entry struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenantId"`
	Type      EventType `json:"type"`
	Actor     string    `json:"actor"` // User, Agent, or System
	Subject   string    `json:"subject"`
	Details   any       `json:"details"`
	PrevHash  string    `json:"prevHash"`
	Hash      string    `json:"hash"` // SHA-256(PrevHash + Type + Subject + Timestamp)
	Timestamp time.Time `json:"timestamp"`
}

// Logger maintains an append-only audit trail per tenant.
type Logger struct {
	mu       sync.RWMutex
	entries  map[string][]*Entry // TenantID -> Slice of Entries
	lastHash map[string]string   // TenantID -> Last record hash
}

// NewLogger creates an audit logger.
func NewLogger() *Logger {
	return &Logger{
		entries:  make(map[string][]*Entry),
		lastHash: make(map[string]string),
	}
}

// Log appends a new audit record to the tenant's chain.
func (l *Logger) Log(tenantID string, eventType EventType, actor, subject string, details any) *Entry {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UTC()
	prevHash := l.lastHash[tenantID]
	if prevHash == "" {
		prevHash = "0000000000000000000000000000000000000000000000000000000000000000"
	}

	raw := fmt.Sprintf("%s|%s|%s|%s|%d", prevHash, eventType, actor, subject, now.UnixNano())
	h := sha256.Sum256([]byte(raw))
	currHash := hex.EncodeToString(h[:])

	entry := &Entry{
		ID:        "aud_" + uuid.NewString()[:12],
		TenantID:  tenantID,
		Type:      eventType,
		Actor:     actor,
		Subject:   subject,
		Details:   details,
		PrevHash:  prevHash,
		Hash:      currHash,
		Timestamp: now,
	}

	l.entries[tenantID] = append(l.entries[tenantID], entry)
	l.lastHash[tenantID] = currHash

	return entry
}

// List returns audit entries for a tenant.
func (l *Logger) List(tenantID string, limit int) []*Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	records := l.entries[tenantID]
	if limit <= 0 || len(records) <= limit {
		// Return copy
		out := make([]*Entry, len(records))
		copy(out, records)
		return out
	}

	// Return most recent entries
	start := len(records) - limit
	out := make([]*Entry, limit)
	copy(out, records[start:])
	return out
}

// VerifyChain verifies cryptographic integrity of a tenant's audit trail.
func (l *Logger) VerifyChain(tenantID string) (bool, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	records := l.entries[tenantID]
	expectedPrev := "0000000000000000000000000000000000000000000000000000000000000000"

	for idx, entry := range records {
		if entry.PrevHash != expectedPrev {
			return false, fmt.Errorf("hash chain broken at entry %d (%s): expected prevHash %s, got %s",
				idx, entry.ID, expectedPrev, entry.PrevHash)
		}

		raw := fmt.Sprintf("%s|%s|%s|%s|%d", entry.PrevHash, entry.Type, entry.Actor, entry.Subject, entry.Timestamp.UnixNano())
		h := sha256.Sum256([]byte(raw))
		computed := hex.EncodeToString(h[:])

		if computed != entry.Hash {
			return false, fmt.Errorf("tampering detected at entry %d (%s): computed hash %s != stored hash %s",
				idx, entry.ID, computed, entry.Hash)
		}

		expectedPrev = entry.Hash
	}

	return true, nil
}
