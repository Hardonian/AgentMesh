package policy

import (
	"sync"
	"time"
)

// AutomationFreeze tracks emergency kill switch and freeze states.
type AutomationFreeze struct {
	Scope     string    `json:"scope"` // GLOBAL, TENANT, PROJECT, CAPABILITY
	ScopeID   string    `json:"scopeId"`
	Reason    string    `json:"reason"`
	FrozenBy  string    `json:"frozenBy"`
	FrozenAt  time.Time `json:"frozenAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// FreezeManager coordinates emergency freezes and kill switch states.
type FreezeManager struct {
	mu      sync.RWMutex
	freezes map[string]*AutomationFreeze // scopeKey -> Freeze
}

// NewFreezeManager creates a thread-safe freeze manager.
func NewFreezeManager() *FreezeManager {
	return &FreezeManager{
		freezes: make(map[string]*AutomationFreeze),
	}
}

// Freeze activates an emergency freeze on a scope.
func (m *FreezeManager) Freeze(scope, scopeID, reason, frozenBy string, expiresAt *time.Time) *AutomationFreeze {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := scope + ":" + scopeID
	freeze := &AutomationFreeze{
		Scope:     scope,
		ScopeID:   scopeID,
		Reason:    reason,
		FrozenBy:  frozenBy,
		FrozenAt:  time.Now().UTC(),
		ExpiresAt: expiresAt,
	}
	m.freezes[key] = freeze
	return freeze
}

// Unfreeze clears an emergency freeze.
func (m *FreezeManager) Unfreeze(scope, scopeID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := scope + ":" + scopeID
	if _, exists := m.freezes[key]; exists {
		delete(m.freezes, key)
		return true
	}
	return false
}

// IsFrozen checks if an action target is affected by any active freeze.
func (m *FreezeManager) IsFrozen(orgID, projectID, capabilityID string) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now().UTC()

	// Check global kill switch
	if f, exists := m.freezes["GLOBAL:all"]; exists {
		if f.ExpiresAt == nil || f.ExpiresAt.After(now) {
			return true, "global emergency kill switch active: " + f.Reason
		}
	}

	// Check organization scope
	if f, exists := m.freezes["TENANT:"+orgID]; exists {
		if f.ExpiresAt == nil || f.ExpiresAt.After(now) {
			return true, "organization freeze active: " + f.Reason
		}
	}

	// Check project scope
	if f, exists := m.freezes["PROJECT:"+projectID]; exists {
		if f.ExpiresAt == nil || f.ExpiresAt.After(now) {
			return true, "project freeze active: " + f.Reason
		}
	}

	// Check capability scope
	if f, exists := m.freezes["CAPABILITY:"+capabilityID]; exists {
		if f.ExpiresAt == nil || f.ExpiresAt.After(now) {
			return true, "capability freeze active: " + f.Reason
		}
	}

	return false, ""
}
