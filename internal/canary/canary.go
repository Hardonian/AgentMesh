package canary

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type RevisionState string

const (
	StateDraft      RevisionState = "DRAFT"
	StateTesting    RevisionState = "TESTING"
	StateCanary     RevisionState = "CANARY"
	StateActive     RevisionState = "ACTIVE"
	StateRolledBack RevisionState = "ROLLED_BACK"
	StateRetired    RevisionState = "RETIRED"
)

var (
	ErrCanaryNotFound          = errors.New("canary deployment not found")
	ErrCannotPromoteRolledBack = errors.New("cannot promote a rolled-back revision")
)

// MetricSnapshot records operational performance for comparison.
type MetricSnapshot struct {
	TotalRequests int64   `json:"totalRequests"`
	SuccessRate   float64 `json:"successRate"`
	P95LatencyMs  int64   `json:"p95LatencyMs"`
	AverageCost   float64 `json:"averageCost"`
	ErrorRate     float64 `json:"errorRate"`
}

// CanaryDeployment governs progressive rollout between baseline and candidate revisions.
type CanaryDeployment struct {
	ID                  string         `json:"id"`
	AgentID             string         `json:"agentId"`
	BaselineVersion     string         `json:"baselineVersion"`
	CandidateVersion    string         `json:"candidateVersion"`
	TrafficWeight       int            `json:"trafficWeight"` // Percentage to candidate: 1, 5, 10, 25, 50, 100
	State               RevisionState  `json:"state"`
	ShadowMode          bool           `json:"shadowMode"` // If true, mirrors traffic with writes blocked
	BaselineMetrics     MetricSnapshot `json:"baselineMetrics"`
	CandidateMetrics    MetricSnapshot `json:"candidateMetrics"`
	MaxAllowedErrorRate float64        `json:"maxAllowedErrorRate"`
	MaxAllowedLatencyMs int64          `json:"maxAllowedLatencyMs"`
	LastEvaluatedAt     time.Time      `json:"lastEvaluatedAt"`
	CreatedAt           time.Time      `json:"createdAt"`
}

// Manager coordinates canary deployments and evaluations.
type Manager struct {
	mu          sync.RWMutex
	deployments map[string]*CanaryDeployment // AgentID -> CanaryDeployment
	activeRevs  map[string]string            // AgentID -> currently ACTIVE version
}

// NewManager creates a canary manager.
func NewManager() *Manager {
	return &Manager{
		deployments: make(map[string]*CanaryDeployment),
		activeRevs:  make(map[string]string),
	}
}

// StartCanary begins a progressive rollout for an agent candidate version.
func (m *Manager) StartCanary(agentID, baselineVer, candidateVer string, initialWeight int, shadowMode bool, maxErrorRate float64, maxLatencyMs int64) (*CanaryDeployment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	dep := &CanaryDeployment{
		ID:                  fmt.Sprintf("canary_%s_%s", agentID, candidateVer),
		AgentID:             agentID,
		BaselineVersion:     baselineVer,
		CandidateVersion:    candidateVer,
		TrafficWeight:       initialWeight,
		State:               StateCanary,
		ShadowMode:          shadowMode,
		MaxAllowedErrorRate: maxErrorRate,
		MaxAllowedLatencyMs: maxLatencyMs,
		CreatedAt:           time.Now().UTC(),
		LastEvaluatedAt:     time.Now().UTC(),
	}

	m.deployments[agentID] = dep
	m.activeRevs[agentID] = baselineVer
	return dep, nil
}

// RecordCandidateSample updates metrics and checks automated rollback thresholds.
func (m *Manager) RecordCandidateSample(agentID string, success bool, latencyMs int64, costUSD float64) (rolledBack bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	dep, exists := m.deployments[agentID]
	if !exists {
		return false, ErrCanaryNotFound
	}
	if dep.State != StateCanary {
		return false, nil
	}

	met := &dep.CandidateMetrics
	prevTotal := float64(met.TotalRequests)
	newTotal := prevTotal + 1
	met.TotalRequests = int64(newTotal)

	var errVal float64
	var succVal float64
	if !success {
		errVal = 1.0
	} else {
		succVal = 1.0
	}

	met.ErrorRate = (met.ErrorRate*prevTotal + errVal) / newTotal
	met.SuccessRate = (met.SuccessRate*prevTotal + succVal) / newTotal
	met.AverageCost = (met.AverageCost*prevTotal + costUSD) / newTotal
	if latencyMs > met.P95LatencyMs {
		met.P95LatencyMs = latencyMs
	}
	dep.LastEvaluatedAt = time.Now().UTC()

	// Check automated rollback condition
	if met.TotalRequests >= 5 {
		if (dep.MaxAllowedErrorRate > 0 && met.ErrorRate > dep.MaxAllowedErrorRate) ||
			(dep.MaxAllowedLatencyMs > 0 && met.P95LatencyMs > dep.MaxAllowedLatencyMs) {
			// Automated Rollback!
			dep.State = StateRolledBack
			dep.TrafficWeight = 0
			// Active stays baseline
			return true, nil
		}
	}

	return false, nil
}

// Promote shifts traffic up or promotes to full ACTIVE.
func (m *Manager) Promote(agentID string, newWeight int) (*CanaryDeployment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	dep, exists := m.deployments[agentID]
	if !exists {
		return nil, ErrCanaryNotFound
	}
	if dep.State == StateRolledBack {
		return nil, ErrCannotPromoteRolledBack
	}

	if newWeight >= 100 {
		dep.TrafficWeight = 100
		dep.State = StateActive
		m.activeRevs[agentID] = dep.CandidateVersion
	} else {
		dep.TrafficWeight = newWeight
	}

	return dep, nil
}

// Rollback manually aborts a canary and restores the baseline revision.
func (m *Manager) Rollback(agentID string, reason string) (*CanaryDeployment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	dep, exists := m.deployments[agentID]
	if !exists {
		return nil, ErrCanaryNotFound
	}

	dep.State = StateRolledBack
	dep.TrafficWeight = 0
	m.activeRevs[agentID] = dep.BaselineVersion

	return dep, nil
}

// GetActiveVersion returns the current production traffic version for an agent.
func (m *Manager) GetActiveVersion(agentID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeRevs[agentID]
}
