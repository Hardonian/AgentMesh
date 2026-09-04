package slo

import (
	"errors"
	"sync"
	"time"

	"github.com/agentmesh/agentmesh/internal/reliability"
)

// SLOStatus defines the compliance state of an SLO.
type SLOStatus string

const (
	StatusHealthy  SLOStatus = "HEALTHY"
	StatusAtRisk   SLOStatus = "AT_RISK"
	StatusBreached SLOStatus = "BREACHED"
	StatusUnknown  SLOStatus = "UNKNOWN"
)

// CapabilityHealthStatus defines aggregate health across eligible agents.
type CapabilityHealthStatus string

const (
	CapHealthy     CapabilityHealthStatus = "HEALTHY"
	CapDegraded    CapabilityHealthStatus = "DEGRADED"
	CapUnavailable CapabilityHealthStatus = "UNAVAILABLE"
)

// AgentSLO specifies performance commitments for an agent and capability.
type AgentSLO struct {
	ID                   string    `json:"id"`
	TenantID             string    `json:"tenantId"`
	AgentID              string    `json:"agentId"`
	CapabilityID         string    `json:"capabilityId"`
	TargetSuccessRate    float64   `json:"targetSuccessRate"`    // e.g. 0.99
	MaxP95LatencyMs      int64     `json:"maxP95LatencyMs"`      // e.g. 4000
	MaxCostUSD           float64   `json:"maxCostUsd"`           // e.g. 0.10
	TargetToolSuccess    float64   `json:"targetToolSuccess"`    // e.g. 0.995
	CurrentStatus        SLOStatus `json:"currentStatus"`
	RemainingErrorBudget float64   `json:"remainingErrorBudget"` // 0.0 to 1.0 (1.0 = 100% budget left)
	LastEvaluatedAt      time.Time `json:"lastEvaluatedAt"`
}

// CapabilityHealth summarizes operational availability of a capability.
type CapabilityHealth struct {
	CapabilityID string                 `json:"capabilityId"`
	TenantID     string                 `json:"tenantId"`
	Status       CapabilityHealthStatus `json:"status"`
	TotalAgents  int                    `json:"totalAgents"`
	HealthyAgents int                   `json:"healthyAgents"`
	BreachedAgents int                  `json:"breachedAgents"`
	P95LatencyMs int64                  `json:"p95LatencyMs"`
	AverageCost  float64                `json:"averageCost"`
	LastUpdated  time.Time              `json:"lastUpdated"`
}

// Manager evaluates and stores AgentSLOs and CapabilityHealth.
type Manager struct {
	mu           sync.RWMutex
	slos         map[string]*AgentSLO        // tenant:agent:cap -> SLO
	capHealth    map[string]*CapabilityHealth // tenant:cap -> CapabilityHealth
}

// NewManager creates an SLO manager.
func NewManager() *Manager {
	return &Manager{
		slos:      make(map[string]*AgentSLO),
		capHealth: make(map[string]*CapabilityHealth),
	}
}

func sloKey(tenantID, agentID, capabilityID string) string {
	return tenantID + ":" + agentID + ":" + capabilityID
}

// SetSLO registers or updates an AgentSLO.
func (m *Manager) SetSLO(slo *AgentSLO) error {
	if slo == nil || slo.TenantID == "" || slo.AgentID == "" || slo.CapabilityID == "" {
		return errors.New("invalid slo parameters")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	k := sloKey(slo.TenantID, slo.AgentID, slo.CapabilityID)
	if slo.CurrentStatus == "" {
		slo.CurrentStatus = StatusUnknown
	}
	if slo.RemainingErrorBudget == 0 {
		slo.RemainingErrorBudget = 1.0
	}
	slo.LastEvaluatedAt = time.Now().UTC()
	m.slos[k] = slo
	return nil
}

// EvaluateSLO evaluates an AgentSLO against a statistical ReliabilityProfile.
func (m *Manager) EvaluateSLO(profile *reliability.ReliabilityProfile) (*AgentSLO, error) {
	if profile == nil {
		return nil, errors.New("profile cannot be nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	k := sloKey(profile.TenantID, profile.AgentID, profile.CapabilityID)
	s, exists := m.slos[k]
	if !exists {
		// Create default SLO target
		s = &AgentSLO{
			ID:                   "slo-" + profile.AgentID + "-" + profile.CapabilityID,
			TenantID:             profile.TenantID,
			AgentID:              profile.AgentID,
			CapabilityID:         profile.CapabilityID,
			TargetSuccessRate:    0.99,
			MaxP95LatencyMs:      5000,
			MaxCostUSD:           0.10,
			TargetToolSuccess:    0.99,
			CurrentStatus:        StatusUnknown,
			RemainingErrorBudget: 1.0,
		}
		m.slos[k] = s
	}

	now := time.Now().UTC()
	s.LastEvaluatedAt = now

	// If insufficient samples, status is UNKNOWN
	if profile.TotalSamples < 5 {
		s.CurrentStatus = StatusUnknown
		s.RemainingErrorBudget = 1.0
		return s, nil
	}

	// Calculate allowed failure budget
	allowedErrorRate := 1.0 - s.TargetSuccessRate
	if allowedErrorRate <= 0 {
		allowedErrorRate = 0.01
	}
	actualErrorRate := 1.0 - profile.OverallSuccessRate

	// Remaining error budget: 1.0 - (actual / allowed)
	budget := 1.0 - (actualErrorRate / allowedErrorRate)
	if budget < 0 {
		budget = 0.0
	}
	if budget > 1.0 {
		budget = 1.0
	}
	s.RemainingErrorBudget = budget

	// Breach evaluation
	breached := false
	atRisk := false

	if profile.OverallSuccessRate < s.TargetSuccessRate ||
		(s.MaxP95LatencyMs > 0 && profile.P95LatencyMs > s.MaxP95LatencyMs) ||
		(s.MaxCostUSD > 0 && profile.AverageCostUSD > s.MaxCostUSD) {
		breached = true
	} else if budget < 0.20 || (s.MaxP95LatencyMs > 0 && float64(profile.P95LatencyMs) > float64(s.MaxP95LatencyMs)*0.85) {
		atRisk = true
	}

	if breached {
		s.CurrentStatus = StatusBreached
	} else if atRisk {
		s.CurrentStatus = StatusAtRisk
	} else {
		s.CurrentStatus = StatusHealthy
	}

	return s, nil
}

// GetSLO returns an AgentSLO.
func (m *Manager) GetSLO(tenantID, agentID, capabilityID string) (*AgentSLO, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.slos[sloKey(tenantID, agentID, capabilityID)]
	return s, ok
}

// ListSLOs returns all registered SLOs for a tenant.
func (m *Manager) ListSLOs(tenantID string) []*AgentSLO {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*AgentSLO, 0)
	for _, s := range m.slos {
		if tenantID == "" || s.TenantID == tenantID {
			list = append(list, s)
		}
	}
	return list
}

// ComputeCapabilityHealth aggregates health from all agents supporting a capability.
func (m *Manager) ComputeCapabilityHealth(tenantID, capabilityID string, profiles []*reliability.ReliabilityProfile) *CapabilityHealth {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := &CapabilityHealth{
		CapabilityID: capabilityID,
		TenantID:     tenantID,
		Status:       CapUnavailable,
		TotalAgents:  len(profiles),
		LastUpdated:  time.Now().UTC(),
	}

	if len(profiles) == 0 {
		m.capHealth[tenantID+":"+capabilityID] = ch
		return ch
	}

	var totalLatency int64
	var totalCost float64

	for _, p := range profiles {
		totalLatency += p.P95LatencyMs
		totalCost += p.AverageCostUSD

		k := sloKey(tenantID, p.AgentID, capabilityID)
		s, exists := m.slos[k]
		if !exists || s.CurrentStatus == StatusHealthy || s.CurrentStatus == StatusUnknown {
			ch.HealthyAgents++
		} else if s.CurrentStatus == StatusBreached {
			ch.BreachedAgents++
		} else {
			// AT_RISK
			ch.HealthyAgents++
		}
	}

	ch.P95LatencyMs = totalLatency / int64(len(profiles))
	ch.AverageCost = totalCost / float64(len(profiles))

	if ch.HealthyAgents == 0 {
		ch.Status = CapUnavailable
	} else if ch.BreachedAgents > 0 || ch.HealthyAgents < ch.TotalAgents {
		ch.Status = CapDegraded
	} else {
		ch.Status = CapHealthy
	}

	m.capHealth[tenantID+":"+capabilityID] = ch
	return ch
}

// GetCapabilityHealth returns capability health.
func (m *Manager) GetCapabilityHealth(tenantID, capabilityID string) (*CapabilityHealth, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ch, ok := m.capHealth[tenantID+":"+capabilityID]
	return ch, ok
}
